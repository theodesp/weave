package errors

import (
	"testing"

	"github.com/iov-one/weave/weavetest/assert"
)

func TestValidationErrors(t *testing.T) {
	validate := func() error {
		var errs ValidationErrors

		// General validation errors.
		errs = errs.With("", Wrap(ErrTimeout, "entity is expired"))
		errs = errs.With("", Wrap(ErrUnauthorized, "only admin can edit this entity"))

		// Specific for the field.
		errs = errs.With("Name", Wrap(ErrEmpty, "required"))
		errs = errs.With("Age", Wrap(ErrInput, "must be above 0"))

		return errs
	}

	result := validate()

	// Casting to ValidationErrors is necessary in tests. In non-tests code
	// we are not interested in this error precise type, similar as we
	// never use Error type directly.
	errs := result.(ValidationErrors)

	// The commonly used Is can be used to test the content of the
	// validation errors container.
	assert.Equal(t, true, ErrEmpty.Is(errs))
	assert.Equal(t, true, ErrInput.Is(errs))
	assert.Equal(t, true, ErrTimeout.Is(errs))
	assert.Equal(t, true, ErrUnauthorized.Is(errs))
	assert.Equal(t, false, ErrModel.Is(errs))
	assert.Equal(t, false, ErrHuman.Is(errs))

	// We can test if the error for a specific field was registered.
	assert.Equal(t, true, ErrEmpty.Is(errs.For("Name")))
	assert.Equal(t, false, ErrTimeout.Is(errs.For("Name")))

	// We can test if general validation errors were captured as well.
	assert.Equal(t, true, ErrTimeout.Is(errs.For("")))
	assert.Equal(t, true, ErrUnauthorized.Is(errs.For("")))
	assert.Equal(t, false, ErrEmpty.Is(errs.For("")))

}
