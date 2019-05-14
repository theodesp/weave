package errors

import (
	"fmt"
	"strings"
)

type ValidationErrors []error

func (errs ValidationErrors) Error() string {
	switch len(errs) {
	case 0:
		return "nil"
	case 1:
		return fmt.Sprintf("1 error\n\t* %s", errs[0].Error())
	default:
		msgs := make([]string, 0, len(errs))
		for _, err := range errs {
			msgs = append(msgs, "\t* "+err.Error())
		}
		return fmt.Sprintf("%d errors\n%s", len(errs), strings.Join(msgs, "\n"))
	}
}

// With returns ValidationErrors with an error added for given field. Use empty
// field name to add a global error (not related to a single field).
func (errs ValidationErrors) With(field string, err error) ValidationErrors {
	if field != "" {
		err = &fieldErr{field: field, err: err}
	}
	if multierr, ok := err.(ValidationErrors); ok {
		// If nested, then flatten.
		for _, other := range multierr {
			errs = append(errs, other)
		}
	} else {
		errs = append(errs, err)
	}
	return errs
}

// For returns all errors that are matching given field name. Use empty field
// name to receive a list of all errors that are not added for any specific
// field.
func (errs ValidationErrors) For(field string) ValidationErrors {
	var res ValidationErrors
	for _, e := range errs {
		switch ferr, ok := e.(*fieldErr); {
		case field == "" && !ok:
			res = append(res, e)
		case field != "" && ok:
			res = append(res, ferr.err)
		}
	}
	return res
}

func (ValidationErrors) ABCICode() uint32 {
	// Reserved for when the validation is not passing.
	return 9142
}

type fieldErr struct {
	field string
	err   error
}

func (e *fieldErr) Error() string {
	return fmt.Sprintf("%s: %s", e.field, e.err)
}

func (e *fieldErr) Cause() error {
	return e.err
}
