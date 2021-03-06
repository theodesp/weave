package multisig

import (
	"testing"

	"github.com/iov-one/weave"
	"github.com/iov-one/weave/errors"
	"github.com/iov-one/weave/weavetest"
)

func TestValidateCreateContractMsg(t *testing.T) {
	cases := map[string]struct {
		Msg     weave.Msg
		WantErr *errors.Error
	}{
		"valid message": {
			Msg: &CreateContractMsg{
				Metadata:            &weave.Metadata{Schema: 1},
				ActivationThreshold: 2,
				AdminThreshold:      3,
				Participants: []*Participant{
					{Weight: 1, Signature: weavetest.NewCondition().Address()},
					{Weight: 2, Signature: weavetest.NewCondition().Address()},
				},
			},
		},
		"missing metadata": {
			Msg: &CreateContractMsg{
				ActivationThreshold: 2,
				AdminThreshold:      3,
				Participants: []*Participant{
					{Weight: 1, Signature: weavetest.NewCondition().Address()},
					{Weight: 2, Signature: weavetest.NewCondition().Address()},
				},
			},
			WantErr: errors.ErrMetadata,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Msg.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}
}

func TestValidateUpdateContractMsg(t *testing.T) {
	cases := map[string]struct {
		Msg     weave.Msg
		WantErr *errors.Error
	}{
		"valid message": {
			Msg: &UpdateContractMsg{
				Metadata:            &weave.Metadata{Schema: 1},
				ActivationThreshold: 2,
				AdminThreshold:      3,
				Participants: []*Participant{
					{Weight: 1, Signature: weavetest.NewCondition().Address()},
					{Weight: 2, Signature: weavetest.NewCondition().Address()},
				},
			},
		},
		"missing metadata": {
			Msg: &UpdateContractMsg{
				ActivationThreshold: 2,
				AdminThreshold:      3,
				Participants: []*Participant{
					{Weight: 1, Signature: weavetest.NewCondition().Address()},
					{Weight: 2, Signature: weavetest.NewCondition().Address()},
				},
			},
			WantErr: errors.ErrMetadata,
		},
	}

	for testName, tc := range cases {
		t.Run(testName, func(t *testing.T) {
			if err := tc.Msg.Validate(); !tc.WantErr.Is(err) {
				t.Fatalf("unexpected validation error: %s", err)
			}
		})
	}
}
