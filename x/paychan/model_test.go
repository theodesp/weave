package paychan

import (
	"testing"

	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestPaymentChannelValidation(t *testing.T) {
	cases := map[string]struct {
		Model   *PaymentChannel
		WantErr *errors.Error
	}{
		"valid": {
			Model: &PaymentChannel{
				Src:          weavetest.NewCondition().Address(),
				SenderPubkey: weavetest.NewKey().PublicKey(),
				Recipient:    weavetest.NewCondition().Address(),
				Timeout:      1240984192,
				Total:        coin.NewCoinp(10, 0, "IOV"),
				Transferred:  coin.NewCoinp(0, 5, "IOV"),
				Memo:         "foo bar",
			},
			WantErr: nil,
		},
		"missing source and recipient - return ErrInvalidModel": {
			Model: &PaymentChannel{
				Src:          nil,
				SenderPubkey: weavetest.NewKey().PublicKey(),
				Recipient:    nil,
				Timeout:      1240984192,
				Total:        coin.NewCoinp(10, 0, "IOV"),
				Transferred:  coin.NewCoinp(0, 5, "IOV"),
				Memo:         "foo bar",
			},
			WantErr: errors.ErrInvalidModel,
		},
		"missing source and recipient - return ErrEmpty": {
			Model: &PaymentChannel{
				Src:          nil,
				SenderPubkey: weavetest.NewKey().PublicKey(),
				Recipient:    nil,
				Timeout:      1240984192,
				Total:        coin.NewCoinp(10, 0, "IOV"),
				Transferred:  coin.NewCoinp(0, 5, "IOV"),
				Memo:         "foo bar",
			},
			WantErr: errors.ErrEmpty,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			err := tc.Model.Validate()
			if !tc.WantErr.Is(err) {
				t.Fatalf("unexpected error: %+v", err)
			}
		})
	}
}
