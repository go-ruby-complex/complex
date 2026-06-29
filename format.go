// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math"
	"strconv"
	"strings"
)

// ToS returns the Ruby Complex#to_s form: the two parts joined by their sign with
// an "i" suffix and no surrounding parentheses, e.g. "1+2i", "1/2-3/4i",
// "Infinity+1i".
func (c *Complex) ToS() string { return c.format(false) }

// Inspect returns the Ruby Complex#inspect form: ToS wrapped in parentheses, with
// any Rational or non-finite-Float part itself parenthesised, e.g. "(1+2i)",
// "((1/2)+(3/4)*i)", "(Infinity+1i)".
func (c *Complex) Inspect() string { return "(" + c.format(true) + ")" }

// String implements fmt.Stringer with the inspect form (the most diagnostic).
func (c *Complex) String() string { return c.Inspect() }

// format renders the body shared by to_s and inspect. inspect adds parentheses
// around parts that are Rational or non-finite Float; both append "*i" rather than
// "i" for those same parts (matching MRI's nucomp_to_s / nucomp_inspect).
func (c *Complex) format(inspect bool) string {
	rs := formatNum(c.re, inspect)

	neg, imAbs := splitSign(c.im)
	sign := "+"
	if neg {
		sign = "-"
	}
	is := formatNum(imAbs, inspect)
	// The "*i" form (rather than "i") is used for a non-finite Float imaginary part
	// in both to_s and inspect, and for a Rational imaginary part only in inspect
	// (to_s renders e.g. "3/4i", inspect "(3/4)*i").
	star := ""
	if nonFiniteFloat(c.im) || (inspect && c.im.IsRat()) {
		star = "*"
	}
	return rs + sign + is + star + "i"
}

// splitSign reports whether n prints with a leading minus and returns the value to
// render for its magnitude. MRI keys the sign off the part's signbit (so -0.0
// prints with a minus) and off the rational/integer sign otherwise; NaN is treated
// as non-negative ("+").
func splitSign(n Num) (neg bool, abs Num) {
	switch {
	case n.IsFloat():
		if math.IsNaN(n.f) {
			return false, n
		}
		if math.Signbit(n.f) {
			return true, Float(math.Abs(n.f))
		}
		return false, n
	default:
		if n.Sign() < 0 {
			return true, numNeg(n)
		}
		return false, n
	}
}

// nonFiniteFloat reports whether n is a Float Infinity or NaN — the values MRI
// always renders with the "*i" multiplication operator (in both to_s and inspect).
func nonFiniteFloat(n Num) bool {
	return n.IsFloat() && (math.IsInf(n.f, 0) || math.IsNaN(n.f))
}

// formatNum renders a single component as Ruby would. For inspect, only a Rational
// is wrapped in parentheses (e.g. "(3/4)"); Floats — finite or not — are never
// parenthesised (MRI prints "Infinity", "NaN" bare).
func formatNum(n Num, inspect bool) string {
	switch {
	case n.IsInt():
		return n.bigInt().String()
	case n.IsRat():
		s := n.r.Num().String() + "/" + n.r.Denom().String()
		if inspect {
			return "(" + s + ")"
		}
		return s
	default:
		return formatFloat(n.f)
	}
}

// formatFloat renders a float64 exactly as Ruby's Float#to_s does: the shortest
// round-tripping decimal, always with a fractional part, switching to "e" notation
// when the base-10 exponent of the leading digit is < -4 or >= 15 (matching MRI's
// flo_to_s decpt thresholds), with a two-digit signed exponent and a normalised
// single-digit mantissa.
func formatFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}

	neg := math.Signbit(f)
	abs := math.Abs(f)

	// Shortest decimal digits and base-10 exponent of the value, via 'e' form:
	// d.dddde±XX where the part before 'e' is in [1,10) (or "0" for zero).
	mant := strconv.FormatFloat(abs, 'e', -1, 64)
	// mant looks like "d.ddde±dd" or "de±dd".
	ePos := strings.IndexByte(mant, 'e')
	digits := mant[:ePos]
	exp, _ := strconv.Atoi(mant[ePos+1:])
	digits = strings.Replace(digits, ".", "", 1) // pure digit string, no point

	var out string
	if abs == 0 {
		out = "0.0"
	} else if exp < -4 || exp >= 15 {
		out = sciNotation(digits, exp)
	} else {
		out = fixedNotation(digits, exp)
	}
	if neg {
		return "-" + out
	}
	return out
}

// fixedNotation renders the shortest digit string with the given base-10 exponent
// (exponent of the first digit) in plain decimal form, always keeping a fractional
// part ("100.0", "0.0001", "1.5").
func fixedNotation(digits string, exp int) string {
	var b strings.Builder
	if exp >= 0 {
		// Digits before the point: exp+1 of them, zero-padded if too few.
		intLen := exp + 1
		if intLen >= len(digits) {
			b.WriteString(digits)
			b.WriteString(strings.Repeat("0", intLen-len(digits)))
			b.WriteString(".0")
		} else {
			b.WriteString(digits[:intLen])
			b.WriteByte('.')
			b.WriteString(digits[intLen:])
		}
	} else {
		// 0.00…digits with -exp-1 leading zeros after the point.
		b.WriteString("0.")
		b.WriteString(strings.Repeat("0", -exp-1))
		b.WriteString(digits)
	}
	return b.String()
}

// sciNotation renders the digit string in Ruby's "d.ddde±XX" form: one leading
// digit, a fractional part (".0" when there is only one digit), and a signed,
// at-least-two-digit exponent.
func sciNotation(digits string, exp int) string {
	var b strings.Builder
	b.WriteByte(digits[0])
	b.WriteByte('.')
	if len(digits) > 1 {
		b.WriteString(digits[1:])
	} else {
		b.WriteByte('0')
	}
	b.WriteByte('e')
	if exp < 0 {
		b.WriteByte('-')
		exp = -exp
	} else {
		b.WriteByte('+')
	}
	es := strconv.Itoa(exp)
	if len(es) < 2 {
		b.WriteByte('0')
	}
	b.WriteString(es)
	return b.String()
}
