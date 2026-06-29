// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math/big"
	"strconv"
	"strings"
)

// Parse converts a Ruby Complex-literal string into a Complex, matching the
// strings MRI's Complex() / String#to_c accept. Surrounding whitespace is ignored;
// internal whitespace is rejected. Recognised forms:
//
//	"1+2i"  "1-2i"  "-1+2i"  "1"  "i"  "-i"  "3i"   (rectangular / real / pure-imag)
//	"1/2+3/4i"  "2.5-1.5i"  "1.5e3+2i"            (rational and float parts)
//	"1@2"  "1.0@0.5"                              (polar: abs@arg, Float result)
//
// "j" is accepted as a synonym for the imaginary unit "i". An unparseable string
// yields a *ParseError (Ruby raises ArgumentError).
func Parse(s string) (*Complex, error) {
	orig := s
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, &ParseError{Input: orig}
	}

	p := &parser{s: s}
	c, ok := p.parse()
	if !ok || !p.atEnd() {
		return nil, &ParseError{Input: orig}
	}
	return c, nil
}

type parser struct {
	s   string
	pos int
}

func (p *parser) atEnd() bool { return p.pos >= len(p.s) }

func (p *parser) peek() byte {
	if p.atEnd() {
		return 0
	}
	return p.s[p.pos]
}

// parse consumes the whole grammar: a leading number, then either a polar tail
// (@angle), a rectangular tail (±imag i), a bare imaginary unit, or nothing.
func (p *parser) parse() (*Complex, bool) {
	// A leading imaginary unit with no real part: "i", "+i", "-i".
	if neg, ok := p.tryBareUnit(); ok {
		one := IntFromInt64(1)
		if neg {
			one = IntFromInt64(-1)
		}
		return &Complex{re: IntFromInt64(0), im: one}, true
	}

	first, ok := p.number()
	if !ok {
		return nil, false
	}

	switch p.peek() {
	case '@': // polar: first is abs, the rest is the angle
		p.pos++
		angle, ok := p.number()
		if !ok {
			return nil, false
		}
		return Polar(first, angle), true
	case 'i', 'j': // pure imaginary: "<num>i"
		p.pos++
		return &Complex{re: IntFromInt64(0), im: first}, true
	case '+', '-':
		// rectangular: first is real, parse the signed imaginary term.
		neg := p.peek() == '-'
		p.pos++
		im, ok := p.imaginaryTerm(neg)
		if !ok {
			return nil, false
		}
		return &Complex{re: first, im: im}, true
	default:
		// real only: "<num>"
		return &Complex{re: first, im: IntFromInt64(0)}, true
	}
}

// tryBareUnit consumes a leading "+i"/"-i"/"i"/"j" with no number before it,
// returning whether a unit was found and whether it was negated. It only succeeds
// when the unit is immediately followed by end-of-input.
func (p *parser) tryBareUnit() (neg bool, ok bool) {
	save := p.pos
	if c := p.peek(); c == '+' || c == '-' {
		neg = c == '-'
		p.pos++
	}
	if c := p.peek(); c == 'i' || c == 'j' {
		p.pos++
		if p.atEnd() {
			return neg, true
		}
	}
	p.pos = save
	return false, false
}

// imaginaryTerm parses the imaginary part after a sign in a rectangular literal.
// The coefficient may be implicit ("1+i") in which case it is ±1; otherwise it is
// the parsed number. The term must end in 'i' or 'j'.
func (p *parser) imaginaryTerm(neg bool) (Num, bool) {
	// Implicit coefficient: "+i" / "-i".
	if c := p.peek(); c == 'i' || c == 'j' {
		p.pos++
		if neg {
			return IntFromInt64(-1), true
		}
		return IntFromInt64(1), true
	}
	// The coefficient must be unsigned here — the sign was already consumed, so a
	// further "+"/"-" (e.g. "1+-2i") is a parse error.
	if c := p.peek(); c == '+' || c == '-' {
		return Num{}, false
	}
	n, ok := p.number()
	if !ok {
		return Num{}, false
	}
	if c := p.peek(); c != 'i' && c != 'j' {
		return Num{}, false
	}
	p.pos++
	if neg {
		return numNeg(n), true
	}
	return n, true
}

// number parses one numeric token — Integer, Rational ("a/b") or Float — with an
// optional leading sign and Ruby's underscore digit separators. It returns the
// Num and whether a token was consumed.
func (p *parser) number() (Num, bool) {
	start := p.pos
	if c := p.peek(); c == '+' || c == '-' {
		p.pos++
	}

	_, hasInt := p.digits()
	isFloat := false

	// Fractional part.
	frac := ""
	if p.peek() == '.' {
		// A '.' must be followed by at least one digit to be a fraction.
		if p.pos+1 < len(p.s) && isDigit(p.s[p.pos+1]) {
			p.pos++
			frac, _ = p.digits()
			isFloat = true
		}
	}
	if !hasInt && frac == "" {
		p.pos = start
		return Num{}, false
	}

	// Exponent.
	if c := p.peek(); c == 'e' || c == 'E' {
		save := p.pos
		p.pos++
		if c := p.peek(); c == '+' || c == '-' {
			p.pos++
		}
		if _, ok := p.digits(); ok {
			isFloat = true
		} else {
			p.pos = save // a stray 'e' is not part of this number
		}
	}

	if isFloat {
		// The slice is a well-formed float literal (sign, digits, '.', exponent),
		// so ParseFloat cannot fail.
		f, _ := strconv.ParseFloat(stripUnderscores(p.s[start:p.pos]), 64)
		return Float(f), true
	}

	// Rational "a/b". digits() already guaranteed both sides are digit runs, so the
	// only failure mode left is a zero denominator (Ruby raises ZeroDivisionError).
	if p.peek() == '/' {
		p.pos++
		if _, ok := p.digits(); !ok {
			return Num{}, false
		}
		parts := strings.SplitN(stripUnderscores(p.s[start:p.pos]), "/", 2)
		num, _ := new(big.Int).SetString(parts[0], 10)
		den, _ := new(big.Int).SetString(parts[1], 10)
		if den.Sign() == 0 {
			return Num{}, false
		}
		return RatFrom(num, den), true
	}

	// Plain integer (guaranteed digit-only by digits()).
	iv, _ := new(big.Int).SetString(stripUnderscores(p.s[start:p.pos]), 10)
	return Int(iv), true
}

// digits consumes a run of decimal digits with embedded underscores (which Ruby
// permits as separators), returning whether at least one digit was consumed. An
// underscore is only accepted between digits, so the run always begins with a
// digit; a trailing underscore is rejected as a non-separator.
func (p *parser) digits() (string, bool) {
	start := p.pos
	for !p.atEnd() {
		c := p.s[p.pos]
		// An underscore is a separator only directly between two digits, so it must
		// follow a digit (Ruby rejects "1_", "1__0").
		if isDigit(c) || (c == '_' && p.pos > start && isDigit(p.s[p.pos-1])) {
			p.pos++
			continue
		}
		break
	}
	got := p.s[start:p.pos]
	// A trailing underscore is not a valid separator; give it back.
	if strings.HasSuffix(got, "_") {
		p.pos--
		got = got[:len(got)-1]
	}
	return got, len(got) > 0
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func stripUnderscores(s string) string { return strings.ReplaceAll(s, "_", "") }
