// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math"
	"math/big"
)

// Pow returns c ** exp (Ruby Complex#**).
//
// When exp is an exact integer and c has exact parts, MRI computes the power by
// exact repeated multiplication so the result stays exact (e.g. Complex(2,3)**5 ==
// (122-597i), Complex(2,3)**-1 == ((2/13)-(3/13)i)). Otherwise — a non-integer
// exponent, or any Float part — it falls back to the floating polar form
// exp(exp · log c), matching MRI's float results bit for bit (both use libm).
func (c *Complex) Pow(exp Num) *Complex {
	if exp.IsInt() && c.re.IsExact() && c.im.IsExact() {
		return c.powInt(exp.bigInt())
	}
	return c.powFloat(exp)
}

// powInt raises an exact Complex to an integer power by binary exponentiation,
// inverting for a negative exponent. The parts stay exact throughout.
func (c *Complex) powInt(n *big.Int) *Complex {
	base := c
	e := new(big.Int).Set(n)
	if e.Sign() < 0 {
		base = (&Complex{re: IntFromInt64(1), im: IntFromInt64(0)}).Div(c)
		e.Neg(e)
	}
	result := &Complex{re: IntFromInt64(1), im: IntFromInt64(0)}
	// Square-and-multiply over the bits of e (handles e == 0 → 1+0i).
	for i := e.BitLen() - 1; i >= 0; i-- {
		result = result.Mul(result)
		if e.Bit(i) == 1 {
			result = result.Mul(base)
		}
	}
	return result
}

// powFloat raises c to the real exponent exp via the polar identity
// c**exp = |c|**exp · (cos(exp·arg) + i sin(exp·arg)), producing Float parts. It
// mirrors MRI's f_complex_polar float path (both delegate to libm pow/cos/sin).
func (c *Complex) powFloat(exp Num) *Complex {
	e := exp.Float64()
	mag := math.Pow(c.Abs(), e)
	theta := e * c.Arg()
	return &Complex{re: Float(mag * math.Cos(theta)), im: Float(mag * math.Sin(theta))}
}
