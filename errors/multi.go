package errors

import (
	"fmt"
	"regexp"
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

// With returns ValidationErrors with an error tagged.
func (errs ValidationErrors) With(tag string, err error) ValidationErrors {
	if multierr, ok := err.(ValidationErrors); ok {
		// If nested, then flatten.
		for _, other := range multierr {
			if tag != "" {
				other = &taggedError{tag: tag, err: other}
			}
			errs = append(errs, other)
		}
		return errs
	}

	if tag != "" {
		err = &taggedError{tag: tag, err: err}
	}
	return append(errs, err)
}

func (ValidationErrors) ABCICode() uint32 {
	// Reserved for when the validation is not passing.
	return 9142
}

type taggedError struct {
	tag string
	err error
}

func (e *taggedError) Error() string {
	return e.err.Error()
}

func (e *taggedError) Cause() error {
	return e.err
}

func FilterTag(err error, pattern string) ValidationErrors {
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	var res ValidationErrors
	switch err := err.(type) {
	case *taggedError:
		if rx.MatchString(err.tag) {
			res = append(res, err.err)
		}
	case ValidationErrors:
		for _, e := range err {
			if te, ok := e.(*taggedError); ok {
				if rx.MatchString(te.tag) {
					res = append(res, te.err)
				}
			}
		}
	}
	return res
}
