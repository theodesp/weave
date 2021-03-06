package assert

import (
	"reflect"
	"testing"
)

// Nil fails the test if given value is not nil.
func Nil(t testing.TB, value interface{}) {
	t.Helper()
	if !isNil(value) {
		// Use %+v so that if we are printing an error that supports
		// stack traces then a full stack trace is shown.
		t.Fatalf("want a nil value, got %+v", value)
	}
}

func isNil(value interface{}) (isnil bool) {
	if value == nil {
		return true
	}

	defer func() {
		if recover() != nil {
			isnil = false
		}
	}()

	// The argument must be a chan, func, interface, map, pointer, or slice
	// value; if it is not, IsNil panics.
	isnil = reflect.ValueOf(value).IsNil()

	return isnil
}

// Equal fails the test if two values are not equal.
func Equal(t testing.TB, want, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("values not equal \nwant %T %v\n got %T %v", want, want, got, got)
	}
}

// Panics will run given function and recover any panic. It will fail the test
// if given function call did not panic.
func Panics(t testing.TB, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("panic expected")
		}
	}()
	fn()
}
