package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// Global error registry, codes 1-99 are reserved for global errors, 0 is reserved for non-errors
var (

	// ErrInternal represents a general case issue that cannot be
	// categorized as any of the below cases.
	// We start as 1 as 0 is reserved for non-errors
	ErrInternal = Register(1, "internal")

	// ErrUnauthorized is used whenever a request without sufficient
	// authorization is handled.
	ErrUnauthorized = Register(2, "unauthorized")

	// ErrNotFound is used when a requested operation cannot be completed
	// due to missing data.
	ErrNotFound = Register(3, "not found")

	// ErrInvalidMsg is returned whenever an event is invalid and cannot be
	// handled.
	ErrInvalidMsg = Register(4, "invalid message")

	// ErrInvalidModel is returned whenever a message is invalid and cannot
	// be used (ie. persisted).
	ErrInvalidModel = Register(5, "invalid model")

	// ErrDuplicate is returned when there is a record already that has the same
	// unique key/index used
	ErrDuplicate = Register(6, "duplicate")

	// ErrHuman is returned when application reaches a code path which should not
	// ever be reached if the code was written as expected by the framework
	ErrHuman = Register(7, "coding error")

	// ErrCannotBeModified is returned when something that is considered immutable
	// gets modified
	ErrCannotBeModified = Register(8, "cannot be modified")

	// ErrEmpty is returned when a value fails a not empty assertion
	ErrEmpty = Register(9, "value is empty")

	// ErrInvalidState is returned when an object is in invalid state
	ErrInvalidState = Register(10, "invalid state")

	// ErrInvalidType is returned whenever the type is not what was expected
	ErrInvalidType = Register(11, "invalid type")

	// ErrInsufficientAmount is returned when an amount of currency is insufficient, e.g. funds/fees
	ErrInsufficientAmount = Register(12, "insufficient amount")

	// ErrInvalidAmount stands for invalid amount of whatever
	ErrInvalidAmount = Register(13, "invalid amount")

	// ErrInvalidInput stands for general input problems indication
	ErrInvalidInput = Register(14, "invalid input")

	// ErrExpired stands for expired entities, normally has to do with block height expirations
	ErrExpired = Register(15, "expired")

	// ErrPanic is only set when we recover from a panic, so we know to redact potentially sensitive system info
	ErrPanic = Register(111222, "panic")
)

// Register returns an error instance that should be used as the base for
// creating error instances during runtime.
//
// Popular root errors are declared in this package, but extensions may want to
// declare custom codes. This function ensures that no error code is used
// twice. Attempt to reuse an error code results in panic.
//
// Use this function only during a program startup phase.
func Register(code uint32, description string) Error {
	if e, ok := usedCodes[code]; ok {
		panic(fmt.Sprintf("error with code %d is already registered: %q", code, e.desc))
	}
	err := Error{
		code: code,
		desc: description,
	}
	usedCodes[err.code] = err
	return err
}

// usedCodes is keeping track of used codes to ensure uniqueness.
var usedCodes = map[uint32]Error{}

// Error represents a root error.
//
// Weave framework is using root error to categorize issues. Each instance
// created during the runtime should wrap one of the declared root errors. This
// allows error tests and returning all errors to the client in a safe manner.
//
// All popular root errors are declared in this package. If an extension has to
// declare a custom root error, always use Register function to ensure
// error code uniqueness.
type Error struct {
	code uint32
	desc string
}

// Error returns the stored description
func (e Error) Error() string { return e.desc }

// ABCILog returns the stored description, same as Error()
func (e Error) ABCILog() string { return e.desc }

// ABCICode returns the associated ABCICode
func (e Error) ABCICode() uint32 { return e.code }

// New returns a new error. Returned instance is having the root cause set to
// this error. Below two lines are equal
//   e.New("my description")
//   Wrap(e, "my description")
// Allows sprintf format and vararg
func (e Error) New(description string) error {
	return Wrap(e, description)
}

// Newf is basically New with formatting capabilities
func (e Error) Newf(description string, args ...interface{}) error {
	return e.New(fmt.Sprintf(description, args...))
}

// Is is a proxy helper for global Is to be able to easily instantiate and match error codes
// for example in tests
func (e Error) Is(err error) bool {
	return Is(e.New(""), err)
}

// Wrap extends given error with an additional information.
//
// If the wrapped error does not provide ABCICode method (ie. stdlib errors),
// it will be labeled as internal error.
//
// If err is nil, this returns nil, avoiding the need for an if statement when
// wrapping a error returned at the end of a function
func Wrap(err error, description string) TMError {
	if err == nil {
		return nil
	}
	return &wrappedError{
		Parent: err,
		Msg:    description,
	}
}

type wrappedError struct {
	// This error layer description.
	Msg string
	// The underlying error that triggered this one.
	Parent error
}

type coder interface {
	ABCICode() uint32
}

func (e *wrappedError) StackTrace() errors.StackTrace {
	// TODO: this is either to be implemented or expectation of it being
	// present removed completely. As this is an early stage of
	// refactoring, this is left unimplemented for now.
	return nil
}

func (e *wrappedError) Error() string {
	// if we have a real error code, show all logs recursively
	if e.Parent == nil {
		return e.Msg
	}
	return fmt.Sprintf("%s: %s", e.Msg, e.Parent.Error())
}

func (e *wrappedError) ABCICode() uint32 {
	if e.Parent == nil {
		return ErrInternal.code
	}
	if p, ok := e.Parent.(coder); ok {
		return p.ABCICode()
	}
	return ErrInternal.code
}

func (e *wrappedError) ABCILog() string {
	return e.Error()
}

func (e *wrappedError) Cause() error {
	type causer interface {
		Cause() error
	}
	// If there is no parent, this is the root error and the cause.
	if e.Parent == nil {
		return e
	}
	if c, ok := e.Parent.(causer); ok {
		if cause := c.Cause(); cause != nil {
			return cause
		}
	}
	return e.Parent
}

// Is returns true if both errors represent the same class of issue. For
// example, both errors' root cause is ErrNotFound.
//
// If two errors are not the same instance, Is always returns false if at least
// one of the errors is internal. This is because all external errors (created
// outside of weave package) are internal to the implementation and we cannot
// reason about their equality.
func Is(a, b error) bool {
	if a == b {
		return true
	}

	type coder interface {
		ABCICode() uint32
	}

	// Two errors are equal only if none of them is internal and they have
	// the same ABCICode.
	ac, ok := a.(coder)
	if !ok || ac.ABCICode() == ErrInternal.code {
		return false
	}
	bc, ok := b.(coder)
	if !ok || bc.ABCICode() == ErrInternal.code {
		return false
	}
	return ac.ABCICode() == bc.ABCICode()
}