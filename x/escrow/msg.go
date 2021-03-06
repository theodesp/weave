package escrow

import (
	"github.com/iov-one/weave"
	coin "github.com/iov-one/weave/coin"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/migration"
)

func init() {
	migration.MustRegister(1, &CreateEscrowMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReleaseEscrowMsg{}, migration.NoModification)
	migration.MustRegister(1, &ReturnEscrowMsg{}, migration.NoModification)
	migration.MustRegister(1, &UpdateEscrowPartiesMsg{}, migration.NoModification)
}

const (
	pathCreateEscrowMsg        = "escrow/create"
	pathReleaseEscrowMsg       = "escrow/release"
	pathReturnEscrowMsg        = "escrow/return"
	pathUpdateEscrowPartiesMsg = "escrow/update"

	maxMemoSize int = 128
)

var _ weave.Msg = (*CreateEscrowMsg)(nil)
var _ weave.Msg = (*ReleaseEscrowMsg)(nil)
var _ weave.Msg = (*ReturnEscrowMsg)(nil)
var _ weave.Msg = (*UpdateEscrowPartiesMsg)(nil)

//--------- Path routing --------

// Path fulfills weave.Msg interface to allow routing
func (CreateEscrowMsg) Path() string {
	return pathCreateEscrowMsg
}

// Path fulfills weave.Msg interface to allow routing
func (ReleaseEscrowMsg) Path() string {
	return pathReleaseEscrowMsg
}

// Path fulfills weave.Msg interface to allow routing
func (ReturnEscrowMsg) Path() string {
	return pathReturnEscrowMsg
}

// Path fulfills weave.Msg interface to allow routing
func (UpdateEscrowPartiesMsg) Path() string {
	return pathUpdateEscrowPartiesMsg
}

//--------- Validation --------

// NewCreateMsg is a helper to quickly build a create escrow message
func NewCreateMsg(
	sender weave.Address,
	recipient weave.Address,
	arbiter weave.Address,
	amount coin.Coins,
	timeout weave.UnixTime,
	memo string,
) *CreateEscrowMsg {
	return &CreateEscrowMsg{
		Metadata:  &weave.Metadata{Schema: 1},
		Src:       sender,
		Recipient: recipient,
		Arbiter:   arbiter,
		Amount:    amount,
		Timeout:   timeout,
		Memo:      memo,
	}
}

// Validate makes sure that this is sensible
func (m *CreateEscrowMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	if err := m.Arbiter.Validate(); err != nil {
		return errors.Wrap(err, "arbiter")
	}
	if err := m.Recipient.Validate(); err != nil {
		return errors.Wrap(err, "recipient")
	}
	if m.Timeout == 0 {
		// Zero timeout is a valid value that dates to 1970-01-01. We
		// know that this value is in the past and makes no sense. Most
		// likely value was not provided and a zero value remained.
		return errors.Wrap(errors.ErrInput, "timeout is required")
	}
	if err := m.Timeout.Validate(); err != nil {
		return errors.Wrap(err, "invalid timeout value")
	}
	if len(m.Memo) > maxMemoSize {
		return errors.Wrapf(errors.ErrInput, "memo %s", m.Memo)
	}
	if err := validateAmount(m.Amount); err != nil {
		return err
	}
	return nil
}

// Validate makes sure that this is sensible
func (m *ReleaseEscrowMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	err := validateEscrowID(m.EscrowId)
	if err != nil {
		return err
	}
	if m.Amount == nil {
		return nil
	}
	return validateAmount(m.Amount)
}

// Validate always returns true for no data
func (m *ReturnEscrowMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	return validateEscrowID(m.EscrowId)
}

// Validate makes sure any included items are valid permissions
// and there is at least one change
func (m *UpdateEscrowPartiesMsg) Validate() error {
	if err := m.Metadata.Validate(); err != nil {
		return errors.Wrap(err, "metadata")
	}
	err := validateEscrowID(m.EscrowId)
	if err != nil {
		return err
	}
	if m.Arbiter == nil && m.Sender == nil && m.Recipient == nil {
		return errors.Wrap(errors.ErrEmpty, "all conditions")
	}

	return validateAddresses(m.Sender, m.Recipient, m.Arbiter)
}

// validateAddresses returns an error if any address doesn't validate
// nil is considered valid here
func validateAddresses(addrs ...weave.Address) error {
	for _, a := range addrs {
		if a != nil {
			if err := a.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAmount(amount coin.Coins) error {
	// we enforce this is positive
	positive := amount.IsPositive()
	if !positive {
		return errors.Wrapf(errors.ErrAmount, "non-positive SendMsg: %#v", &amount)
	}
	// then make sure these are properly formatted coins
	return amount.Validate()
}

func validateEscrowID(id []byte) error {
	if len(id) != 8 {
		return errors.Wrapf(errors.ErrInput, "escrow id: %X", id)
	}
	return nil
}
