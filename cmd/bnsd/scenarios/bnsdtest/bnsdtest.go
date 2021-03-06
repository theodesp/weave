package bnsdtest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iov-one/weave"
	weaveClient "github.com/iov-one/weave/client"
	"github.com/iov-one/weave/cmd/bnsd/app"
	"github.com/iov-one/weave/cmd/bnsd/client"
	"github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/commands/server"
	"github.com/iov-one/weave/crypto"
	"github.com/iov-one/weave/migration"
	"github.com/iov-one/weave/weavetest"
	"github.com/iov-one/weave/x/cash"
	"github.com/iov-one/weave/x/distribution"
	"github.com/iov-one/weave/x/escrow"
	"github.com/iov-one/weave/x/multisig"
	"github.com/stellar/go/exp/crypto/derivation"
	"github.com/tendermint/tendermint/libs/log"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tm "github.com/tendermint/tendermint/types"
)

func StartBnsd(t testing.TB, opts ...StartBnsdOption) (env *EnvConf, cleanup func()) {
	env = &EnvConf{
		MinFee:           coin.Coin{},
		AntiSpamFee:      coin.Coin{Whole: 0, Fractional: 50000000, Ticker: "IOV"},
		Alice:            derivePrivateKey(t, *hexSeed, *derivationPath),
		Logger:           log.NewTMLogger(ioutil.Discard),
		MultiSigContract: multisig.MultiSigCondition(weavetest.SequenceID(1)),
		EscrowContract:   escrow.Condition(weavetest.SequenceID(1)),
		clientThrottle:   *delay,
		msgfees:          make(map[string]coin.Coin),
		governance: governance{
			votingPeriod: weave.AsUnixDuration(time.Hour),
			electors: []weave.Address{
				weavetest.NewCondition().Address(),
			},
		},
	}
	env.DistrContractAddr, _ = distribution.RevenueAccount(weavetest.SequenceID(1))

	for _, fn := range opts {
		fn(env)
	}

	if *tendermintAddress == TendermintLocalAddr {
		return env, startLocalBnsd(t, env)
	}
	return env, startRemoteBnsd(t, env)
}

type StartBnsdOption func(*EnvConf)

func startRemoteBnsd(t testing.TB, env *EnvConf) (cleanup func()) {
	cli := client.NewClient(client.NewHTTPConnection(*tendermintAddress))
	thCli := throttle(cli, env.clientThrottle)
	env.Client = thCli

	if chainID, err := cli.ChainID(); err != nil {
		t.Fatalf("failed to fetch chain id: %s", err)
	} else {
		env.ChainID = chainID
	}

	env.RpcAddress = *tendermintAddress
	return func() {
		thCli.Close()
	}
}

func startLocalBnsd(t testing.TB, env *EnvConf) (cleanup func()) {
	tmWorkDir := fmt.Sprintf("bnsd_%s_%d", t.Name(), time.Now().UnixNano())
	tmConf := buildConfig(t, tmWorkDir)

	tmConf.Moniker = "SetInTestMain"
	env.ChainID = tmConf.ChainID()

	env.RpcAddress = "http://localhost" + tmConf.RPC.ListenAddress[strings.LastIndex(tmConf.RPC.ListenAddress, ":"):]

	initGenesis(t, env, tmConf.GenesisFile())

	bnsd, err := app.GenerateApp(&server.Options{
		MinFee: env.MinFee,
		Home:   tmConf.RootDir,
		Logger: env.Logger,
		Debug:  false,
	})
	if err != nil {
		t.Fatalf("cannot generate application: %s", err)
	}

	env.Node = newTendermint(t, tmConf, bnsd)
	if err := env.Node.Start(); err != nil {
		t.Fatalf("cannot start tendermint node: %s", err)
	}

	waitForRPC(t, tmConf)
	waitForGRPC(t, tmConf)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := weaveClient.NewLocalClient(env.Node).WaitForNextBlock(ctx); err != nil {
		t.Fatalf("cannot start tendermint application: %s", err)
	}

	cli := client.NewClient(client.NewLocalConnection(env.Node))
	thCli := throttle(cli, env.clientThrottle)
	env.Client = thCli

	return func() {
		thCli.Close()
		env.Node.Stop()
		env.Node.Wait()
		os.RemoveAll(tmWorkDir)
	}
}

const TendermintLocalAddr = "localhost:46657"

var (
	tendermintAddress = flag.String("address", TendermintLocalAddr, "destination address of tendermint rpc")
	hexSeed           = flag.String("seed", "d34c1970ae90acf3405f2d99dcaca16d0c7db379f4beafcfdf667b9d69ce350d27f5fb440509dfa79ec883a0510bc9a9614c3d44188881f0c5e402898b4bf3c9", "private key seed in hex")
	delay             = flag.Duration("delay", 10*time.Millisecond, "duration to wait between test cases for rate limits")
	derivationPath    = flag.String("derivation", "", "bip44 derivation path: \"m/44'/234'/0'\"")
)

// derivePrivateKey derive a private key from hex and given path. Path can be empty to not derive.
func derivePrivateKey(t testing.TB, hexSeed, path string) *client.PrivateKey {
	if len(path) != 0 {
		seed, err := hex.DecodeString(hexSeed)
		if err != nil {
			t.Fatalf("failed to decode private key: %s", err)
		}
		k, err := derivation.DeriveForPath(path, seed)
		if err != nil {
			t.Fatalf("failed to derive private key using path=%q: %s", path, err)
		}
		pubKey, err := k.PublicKey()
		if err != nil {
			t.Fatalf("failed to derive public key: %s", err)
		}
		hexSeed = hex.EncodeToString(append(k.Key, pubKey...))
	}
	pk, err := client.DecodePrivateKeyFromSeed(hexSeed)
	if err != nil {
		t.Fatalf("failed to decode private key: %s", err)
	}
	return pk
}

func initGenesis(t testing.TB, env *EnvConf, filename string) {
	t.Helper()

	doc, err := tm.GenesisDocFromFile(filename)
	if err != nil {
		t.Fatalf("failed to load genesis from the file: %s", err)
	}
	doc.ConsensusParams.Block.TimeIotaMs = int64(1)
	type dict map[string]interface{}

	msgfees := make([]dict, 0, len(env.msgfees))
	for path, fee := range env.msgfees {
		msgfees = append(msgfees, dict{"msg_path": path, "fee": fee})
	}

	if len(env.governance.electors) == 0 {
		t.Fatal("gov electorate is empty")
	}
	electors := make([]dict, 0)
	for _, e := range env.governance.electors {
		electors = append(electors, dict{
			"weight":  1,
			"address": e,
		})
	}

	appState, err := json.MarshalIndent(dict{
		"cash": []interface{}{
			dict{
				"address": env.Alice.PublicKey().Address(),
				"coins": []interface{}{
					dict{
						"whole":  123456789,
						"ticker": "IOV",
					},
					dict{
						"whole":  123456789,
						"ticker": "CASH",
					},
					dict{
						"whole":  123456789,
						"ticker": "ALX",
					},
					dict{
						"whole":  123456789,
						"ticker": "PAJA",
					},
				},
			},
		},
		"currencies": []interface{}{
			dict{
				"ticker": "IOV",
				"name":   "Main token of this chain",
			}}, "update_validators": dict{"addresses": []interface{}{
			"cond:multisig/usage/0000000000000001",
		},
		},
		"multisig": []interface{}{
			dict{
				"participants": []interface{}{
					dict{"weight": 1, "signature": env.Alice.PublicKey().Address()},
				},
				"activation_threshold": 1,
				"admin_threshold":      1,
			},
		},
		"distribution": []interface{}{
			dict{
				"admin": "cond:multisig/usage/0000000000000001",
				"recipients": []interface{}{
					dict{"weight": 1, "address": env.Alice.PublicKey().Address()},
				},
			},
		},
		"escrow": []interface{}{
			dict{
				"sender":    "0000000000000000000000000000000000000000",
				"arbiter":   env.MultiSigContract.Address(),
				"recipient": "cond:dist/revenue/0000000000000001",
				"amount": []interface{}{
					dict{
						"whole":  1000000,
						"ticker": "IOV",
					}},
				"timeout": time.Now().Add(10000 * time.Hour),
			},
		},
		"conf": dict{
			"cash": cash.Configuration{
				CollectorAddress: weave.Condition("dist/revenue/0000000000000001").Address(),
				MinimalFee:       env.AntiSpamFee,
			},
			"migration": migration.Configuration{
				Admin: weave.Condition("multisig/usage/0000000000000001").Address(),
			},
		},
		"governance": dict{
			"electorate": []interface{}{
				dict{
					"title":    "first electorate",
					"admin":    env.governance.electors[0],
					"electors": electors,
				},
			},
			"rules": []interface{}{
				dict{
					"admin":         env.governance.electors[0],
					"title":         "first rule",
					"voting_period": env.governance.votingPeriod,
					"threshold": dict{
						// Almost rules of majority (50%)
						"numerator":   1,
						"denominator": 2,
					},
					"electorate_id": 1,
				},
			},
		},
		"msgfee": msgfees,
		"initialize_schema": []dict{
			dict{"ver": 1, "pkg": "batch"},
			dict{"ver": 1, "pkg": "cash"},
			dict{"ver": 1, "pkg": "currency"},
			dict{"ver": 1, "pkg": "distribution"},
			dict{"ver": 1, "pkg": "escrow"},
			dict{"ver": 1, "pkg": "gov"},
			dict{"ver": 1, "pkg": "msgfee"},
			dict{"ver": 1, "pkg": "multisig"},
			dict{"ver": 1, "pkg": "namecoin"},
			dict{"ver": 1, "pkg": "nft"},
			dict{"ver": 1, "pkg": "paychan"},
			dict{"ver": 1, "pkg": "sigs"},
			dict{"ver": 1, "pkg": "utils"},
			dict{"ver": 1, "pkg": "validators"},
		},
	}, "", "  ")
	if err != nil {
		t.Fatalf("cannot serialize genesis to JSON: %s", err)
	}
	doc.AppState = appState
	if err := doc.SaveAs(filename); err != nil {
		t.Fatalf("cannot save genesis into %q file: %s", filename, err)
	}
}

// SeedAccountWithTokens acts as a faucet that sends tokens to the given address.
func SeedAccountWithTokens(t testing.TB, env *EnvConf, dest weave.Address) {
	t.Helper()

	cc := coin.NewCoin(10, 0, "IOV")
	tx := client.BuildSendTx(env.Alice.PublicKey().Address(), dest, cc, "faucet")
	tx.Fee(env.Alice.PublicKey().Address(), env.AntiSpamFee)

	aNonce := client.NewNonce(env.Client, env.Alice.PublicKey().Address())
	seq, err := aNonce.Next()
	if err != nil {
		t.Fatalf("cannot get the nonce value: %s", err)
	}
	if err := client.SignTx(tx, env.Alice, env.ChainID, seq); err != nil {
		t.Fatalf("cannot sign the transaction: %s", err)
	}
	resp := env.Client.BroadcastTx(tx)
	if err := resp.IsError(); err != nil {
		t.Fatalf("transaction failed: %s", err)
	}
}

// MustSignTx will modify given transaction by signing it with provided private
// key. This function fails the test if any operation was not successful.
func MustSignTx(t testing.TB, env *EnvConf, tx *app.Tx, pk *crypto.PrivateKey) {
	t.Helper()

	nonce := client.NewNonce(env.Client, pk.PublicKey().Address())
	seq, err := nonce.Next()
	if err != nil {
		t.Fatalf("cannot acquire nonce sequence: %s", err)
	}
	if err := client.SignTx(tx, pk, env.ChainID, seq); err != nil {
		msg, _ := tx.GetMsg()
		t.Fatalf("cannot sing %T transaction: %s", msg, err)
	}
}

// MustBroadcastTx submits given transaction to the network and returns
// broadcast response. This function fails the test if any operation was not
// successful or broadcast response contain an error.
func MustBroadcastTx(t testing.TB, env *EnvConf, tx *app.Tx) *coretypes.ResultBroadcastTxCommit {
	t.Helper()

	resp := env.Client.BroadcastTx(tx)
	if resp.Error != nil {
		t.Fatalf("cannot broadcast: %s", resp.Error)
	}
	if resp.Response.CheckTx.IsErr() {
		r := resp.Response.CheckTx
		t.Fatalf("CheckTx failed with code %d: %s", r.Code, r.Log)
	}
	if resp.Response.DeliverTx.IsErr() {
		r := resp.Response.DeliverTx
		t.Fatalf("DeliverTx failed with code %d: %s", r.Code, r.Log)
	}
	return resp.Response
}
