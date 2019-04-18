package errors

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Errors []error
}

func (e *ValidationError) Empty() bool {
	return len(e.Errors) == 0
}

func (e *ValidationError) Error() string {
	switch len(e.Errors) {
	case 0:
		return "nil"
	case 1:
		return fmt.Sprintf("1 error\n\t* %s", e.Errors[0].Error())
	default:
		msgs := make([]string, 0, len(e.Errors))
		for _, err := range e.Errors {
			msgs = append(msgs, "\t* "+err.Error())
		}
		return fmt.Sprintf("%d errors\n%s", len(e.Errors), strings.Join(msgs, "\n"))
	}
}

func (e *ValidationError) Add(err error) {
	if multierr, ok := err.(*ValidationError); ok {
		// If nested, then flatten.
		for _, other := range multierr.Errors {
			e.Errors = append(e.Errors, other)
		}
	} else {
		e.Errors = append(e.Errors, err)
	}
}

func (e *ValidationError) ABCICode() uint32 {
	// Reserved for when the validation is not passing.
	return 9142
}
