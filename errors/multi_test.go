package errors

import (
	"testing"
)

func TestEmptyMultiError(t *testing.T) {
	if err := MultiAdd(); err != nil {
		t.Fatalf("empty multi error must be nil, got %#v", err)
	}

	// The same must be true when returned from a function.
	err := func() error {
		var set []error
		return MultiAdd(set...)
	}()
	if err != nil {
		t.Fatalf("empty multi error must be nil, got %#v", err)
	}

	// XXX the same should be true for MultiAddNamed
}

func TestValidationExample(t *testing.T) {
	validate := func() error {
		var err error

		// General validation
		err = MultiAdd(err, Wrap(ErrTimeout, "entity is expired"))
		err = MultiAdd(err, Wrap(ErrUnauthorized, "only admin can edit this entity"))

		// Specific for the field.
		err = MultiAddNamed("Name", Wrap(ErrEmpty, "required"))
		err = MultiAddNamed("Age", Wrap(ErrInput, "must be above 0"))

		// A single field can cause failure for more than one reason.
		// This is not a usual case, but can happen for more complex
		// logic.
		err = MultiAddNamed("Auth", Wrap(ErrUnauthorized, "not an admin"))
		err = MultiAddNamed("Auth", Wrap(ErrExpired, "used user group is no longer active"))

		return err
	}

	err := validate()

	assertErrIs(t, true, err, ErrTimeout)
	assertErrIs(t, true, err, ErrUnauthorized)
	assertErrIs(t, true, err, ErrInput)
	assertErrIs(t, true, err, ErrDuplicate)
	assertErrIs(t, true, err, ErrExpired)

	assertErrIs(t, false, err, ErrNotFound)

	// XXX How to test if an error for a field was reported?
}

func assertErrIs(t testing.TB, ok bool, err error, want *Error) {
	t.Helper()
	if want.Is(err) != ok {
		if ok {
			t.Errorf("want error to be of type %q, got %q instead", want, err)
		} else {
			t.Errorf("want error to not be of type %q, got %q instead", want, err)
		}
	}
}
