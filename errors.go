// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import "fmt"

// RangeError mirrors Ruby's RangeError, raised when a Complex with a non-zero
// imaginary part is asked to collapse to a real (to_f / to_i / to_r).
type RangeError struct {
	Target string // the Ruby class the conversion targets ("Float", "Integer", …)
}

func (e *RangeError) Error() string {
	return fmt.Sprintf("can't convert %s into %s", "Complex", e.Target)
}

func errNotReal(target string) error { return &RangeError{Target: target} }

// ParseError mirrors Ruby's ArgumentError raised by Complex() on an unparseable
// string (e.g. Complex("hello")).
type ParseError struct {
	Input string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("invalid value for convert(): %q", e.Input)
}
