// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package complex is a pure-Go (CGO=0), MRI-4.0.5-byte-exact reimplementation of
// Ruby's Complex (a+bi) number.
//
// Unlike Go's built-in complex128, Ruby's Complex keeps its parts on the numeric
// tower: an Integer part stays an arbitrary-precision Integer, a Rational part
// stays Rational, and only a Float part becomes Float. So Complex(1,2)*Complex(1,2)
// is the *exact* (-3+4i), Complex(Rational(1,2), Rational(3,4)) inspects as
// ((1/2)+(3/4)*i), and only operations that intrinsically need a float (abs, arg,
// non-integer powers) produce floats. This package reproduces that behaviour and
// MRI's exact `inspect` / `to_s` byte layout, with no Ruby runtime and no cgo.
//
// It is the Complex backend for go-embedded-ruby's rbgo, but is a standalone,
// reusable module.
package complex

import (
	"math"
	"math/big"
)

// Complex is a Ruby Complex value a+bi with parts that preserve the numeric tower
// (see Num). The zero value is the Complex 0+0i.
type Complex struct {
	re Num
	im Num
}

// New returns the Complex re+im*i. It is the Go-level constructor; Rect is the
// MRI Complex.rect / Complex.rectangular alias.
func New(re, im Num) *Complex { return &Complex{re: re, im: im} }

// Rect returns the Complex re+im*i (Ruby Complex.rect / Complex.rectangular).
func Rect(re, im Num) *Complex { return &Complex{re: re, im: im} }

// Polar returns the Complex with the given magnitude abs and angle arg, i.e.
// abs*(cos arg + i sin arg) (Ruby Complex.polar). The result has Float parts.
func Polar(abs, arg Num) *Complex {
	a, t := abs.Float64(), arg.Float64()
	return &Complex{re: Float(a * math.Cos(t)), im: Float(a * math.Sin(t))}
}

// Real returns the real part (Ruby Complex#real).
func (c *Complex) Real() Num { return c.re }

// Imaginary returns the imaginary part (Ruby Complex#imaginary / #imag).
func (c *Complex) Imaginary() Num { return c.im }

// Rectangular returns the [real, imaginary] parts (Ruby Complex#rectangular / #rect).
func (c *Complex) Rectangular() (Num, Num) { return c.re, c.im }

// Add returns c + o (Ruby Complex#+).
func (c *Complex) Add(o *Complex) *Complex {
	return &Complex{re: numAdd(c.re, o.re), im: numAdd(c.im, o.im)}
}

// Sub returns c - o (Ruby Complex#-).
func (c *Complex) Sub(o *Complex) *Complex {
	return &Complex{re: numSub(c.re, o.re), im: numSub(c.im, o.im)}
}

// Mul returns c * o (Ruby Complex#*): (a+bi)(c+di) = (ac-bd)+(ad+bc)i.
func (c *Complex) Mul(o *Complex) *Complex {
	ac := numMul(c.re, o.re)
	bd := numMul(c.im, o.im)
	ad := numMul(c.re, o.im)
	bc := numMul(c.im, o.re)
	return &Complex{re: numSub(ac, bd), im: numAdd(ad, bc)}
}

// Div returns c / o (Ruby Complex#/): multiply by the conjugate over |o|².
// Exact operands with a non-zero divisor yield exact parts (matching MRI, e.g.
// Complex(1,2)/Complex(3,4) == ((11/25)+(2/25)i)); a Float operand makes the
// result Float.
func (c *Complex) Div(o *Complex) *Complex {
	denom := numAdd(numMul(o.re, o.re), numMul(o.im, o.im))
	reNum := numAdd(numMul(c.re, o.re), numMul(c.im, o.im))
	imNum := numSub(numMul(c.im, o.re), numMul(c.re, o.im))
	return &Complex{re: numDiv(reNum, denom), im: numDiv(imNum, denom)}
}

// Neg returns -c (Ruby Complex#-@).
func (c *Complex) Neg() *Complex {
	return &Complex{re: numNeg(c.re), im: numNeg(c.im)}
}

// Conjugate returns the complex conjugate a-bi (Ruby Complex#conjugate / #conj).
func (c *Complex) Conjugate() *Complex {
	return &Complex{re: c.re, im: numNeg(c.im)}
}

// Abs2 returns the squared magnitude a²+b² (Ruby Complex#abs2). It stays exact
// when both parts are exact.
func (c *Complex) Abs2() Num {
	return numAdd(numMul(c.re, c.re), numMul(c.im, c.im))
}

// Abs returns the magnitude √(a²+b²) (Ruby Complex#abs / #magnitude). MRI always
// returns a Float (e.g. Complex(3,4).abs == 5.0). math.Hypot matches MRI's
// overflow-safe computation.
func (c *Complex) Abs() float64 {
	return math.Hypot(c.re.Float64(), c.im.Float64())
}

// Arg returns the phase angle atan2(b, a) in radians (Ruby Complex#arg / #angle /
// #phase). It is always a Float.
func (c *Complex) Arg() float64 {
	return math.Atan2(c.im.Float64(), c.re.Float64())
}

// PolarParts returns the [abs, arg] pair (Ruby Complex#polar).
func (c *Complex) PolarParts() (float64, float64) {
	return c.Abs(), c.Arg()
}

// Eql reports whether c eql? o: same parts under Ruby's eql? (type-sensitive, so
// Complex(1,2).eql?(Complex(1.0,2.0)) is false). For value equality (==) use Equal.
func (c *Complex) Eql(o *Complex) bool {
	return numEql(c.re, o.re) && numEql(c.im, o.im)
}

// Equal reports whether c == o under Ruby's == (value equality across the tower,
// so Complex(1,2) == Complex(1.0,2.0) is true).
func (c *Complex) Equal(o *Complex) bool {
	return numEqual(c.re, o.re) && numEqual(c.im, o.im)
}

// FiniteQ reports whether both parts are finite (Ruby Complex#finite?).
func (c *Complex) FiniteQ() bool { return c.re.IsFinite() && c.im.IsFinite() }

// InfiniteQ reports Ruby Complex#infinite?: it returns (1, true) when either part
// is infinite, otherwise (0, false) — Ruby yields 1 or nil, the bool distinguishes
// the nil case.
func (c *Complex) InfiniteQ() (int, bool) {
	if c.re.IsInfinite() || c.im.IsInfinite() {
		return 1, true
	}
	return 0, false
}

// ToF returns the real part as a float64 when the imaginary part is exactly zero
// (Ruby Complex#to_f). It errors otherwise, mirroring MRI's RangeError.
func (c *Complex) ToF() (float64, error) {
	if !c.im.IsZero() {
		return 0, errNotReal("Float")
	}
	return c.re.Float64(), nil
}

// ToI returns the real part as an Integer when the imaginary part is zero and the
// real part is an exact integer (Ruby Complex#to_i). It errors otherwise.
func (c *Complex) ToI() (*big.Int, error) {
	if !c.im.IsZero() {
		return nil, errNotReal("Integer")
	}
	switch {
	case c.re.IsInt():
		return new(big.Int).Set(c.re.bigInt()), nil
	case c.re.IsRat():
		// Ruby truncates a Rational real toward zero for to_i.
		q := new(big.Int).Quo(c.re.r.Num(), c.re.r.Denom())
		return q, nil
	default:
		// Float real: truncate toward zero like Float#to_i.
		return new(big.Int).SetInt64(int64(c.re.f)), nil
	}
}

// ToR returns the real part as an exact Rational Num when the imaginary part is
// zero (Ruby Complex#to_r). MRI raises RangeError only for a non-zero imaginary
// part; a Float real part is converted exactly via Float#to_r.
func (c *Complex) ToR() (Num, error) {
	if !c.im.IsZero() {
		return Num{}, errNotReal("Rational")
	}
	if c.re.IsFloat() {
		r := new(big.Rat).SetFloat64(c.re.f)
		if r == nil { // Inf or NaN have no rational value
			return Num{}, errNotExact
		}
		return Rat(r), nil
	}
	return c.re, nil
}

// ToC returns the receiver (Ruby Complex#to_c).
func (c *Complex) ToC() *Complex { return c }

// Numerator returns the Complex numerator over the common denominator of the two
// exact parts (Ruby Complex#numerator). E.g. Complex(1/2,3/4).numerator == (2+3i)
// with denominator 4.
func (c *Complex) Numerator() *Complex {
	_, num := c.commonDen()
	return num
}

// Denominator returns the least common denominator of the two parts (Ruby
// Complex#denominator). For integer parts it is 1.
func (c *Complex) Denominator() Num {
	den, _ := c.commonDen()
	return den
}

// commonDen computes the lcm of the two parts' denominators and the Complex whose
// parts are each part scaled to that denominator's numerator (the Numerator).
func (c *Complex) commonDen() (Num, *Complex) {
	_, rd := ratNumDen(c.re)
	_, id := ratNumDen(c.im)
	g := new(big.Int).GCD(nil, nil, rd, id)
	lcm := new(big.Int).Mul(rd, new(big.Int).Quo(id, g))
	den := Int(lcm)
	reN, _ := ratNumDen(numMul(c.re, den))
	imN, _ := ratNumDen(numMul(c.im, den))
	return den, &Complex{re: Int(reN), im: Int(imN)}
}
