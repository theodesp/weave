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

	err := validate()

	// The commonly used Is can be used to test the content of the
	// validation errors container.
	assert.Equal(t, true, ErrEmpty.Is(err))
	assert.Equal(t, true, ErrInput.Is(err))
	assert.Equal(t, true, ErrTimeout.Is(err))
	assert.Equal(t, true, ErrUnauthorized.Is(err))
	assert.Equal(t, false, ErrModel.Is(err))
	assert.Equal(t, false, ErrHuman.Is(err))

	// We can test if the error for a specific field was registered.
	assert.Equal(t, true, ErrEmpty.Is(FilterTag(err, "Name")))
	assert.Equal(t, false, ErrTimeout.Is(FilterTag(err, "Name")))

	// We can test if general validation errors were captured as well.
	assert.Equal(t, true, ErrTimeout.Is(FilterTag(err, "")))
	assert.Equal(t, true, ErrUnauthorized.Is(FilterTag(err, "")))
	assert.Equal(t, false, ErrEmpty.Is(FilterTag(err, "")))

}
