// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"errors"
	"math"
	"math/big"
)

// Num models a single component (real or imaginary) of a Ruby Complex. Ruby keeps
// the parts *exact* whenever it can: an Integer part stays Integer (arbitrary
// precision), a Rational part stays Rational, and only a Float part becomes Float.
// Num is the small numeric union that preserves that tower so this package can be
// byte-exact with MRI's `inspect` / `to_s` (e.g. Complex(1,2)*Complex(1,2) is the
// *exact* (-3+4i), and Complex(Rational(1,2), …) renders 1/2 not 0.5).
//
// The zero Num is the integer 0.
type Num struct {
	kind kind
	// i holds an Integer value (kind==kindInt). nil is treated as 0.
	i *big.Int
	// r holds a Rational value (kind==kindRat). It is always kept in lowest
	// terms with a positive denominator, matching Ruby's Rational invariant.
	r *big.Rat
	// f holds a Float value (kind==kindFloat).
	f float64
}

type kind uint8

const (
	kindInt kind = iota
	kindRat
	kindFloat
)

// errNotExact is returned by conversions that require an exact (non-Float) value.
var errNotExact = errors.New("complex: value is not exact")

// Int returns a Num holding the arbitrary-precision integer v.
func Int(v *big.Int) Num {
	if v == nil {
		v = new(big.Int)
	}
	return Num{kind: kindInt, i: new(big.Int).Set(v)}
}

// IntFromInt64 returns a Num holding the integer v.
func IntFromInt64(v int64) Num { return Num{kind: kindInt, i: big.NewInt(v)} }

// Rat returns a Num holding the rational v. Unlike Go's big.Rat, Ruby's Rational
// does *not* collapse to an Integer when the denominator is 1 — Rational(6,3) is
// (2/1), a Rational — so this keeps the Rational kind. Integer Nums arise only from
// Int / IntFromInt64 and from Integer×Integer arithmetic.
func Rat(v *big.Rat) Num {
	if v == nil {
		v = new(big.Rat)
	}
	return Num{kind: kindRat, r: new(big.Rat).Set(v)}
}

// RatFrom returns a Num holding the rational num/den, reduced to lowest terms
// (with a positive denominator). The Rational kind is preserved even when the
// value is whole, matching Ruby's Rational.
func RatFrom(num, den *big.Int) Num {
	return Rat(new(big.Rat).SetFrac(num, den))
}

// RatFromInt64 is RatFrom for int64 numerator and denominator.
func RatFromInt64(num, den int64) Num {
	return Rat(new(big.Rat).SetFrac64(num, den))
}

// Float returns a Num holding the float v.
func Float(v float64) Num { return Num{kind: kindFloat, f: v} }

// IsInt reports whether n is an Integer component.
func (n Num) IsInt() bool { return n.kind == kindInt }

// IsRat reports whether n is a Rational component.
func (n Num) IsRat() bool { return n.kind == kindRat }

// IsFloat reports whether n is a Float component.
func (n Num) IsFloat() bool { return n.kind == kindFloat }

// IsExact reports whether n is exact (Integer or Rational), i.e. not a Float.
func (n Num) IsExact() bool { return n.kind != kindFloat }

// bigInt returns the underlying *big.Int for an Integer Num (treating nil as 0).
func (n Num) bigInt() *big.Int {
	if n.i == nil {
		return new(big.Int)
	}
	return n.i
}

// asRat returns n as a *big.Rat. It panics for Float values; callers gate on
// IsExact first.
func (n Num) asRat() *big.Rat {
	switch n.kind {
	case kindInt:
		return new(big.Rat).SetInt(n.bigInt())
	default: // kindRat
		return new(big.Rat).Set(n.r)
	}
}

// Float64 returns n as a float64.
func (n Num) Float64() float64 {
	switch n.kind {
	case kindInt:
		f := new(big.Float).SetInt(n.bigInt())
		v, _ := f.Float64()
		return v
	case kindRat:
		v, _ := n.r.Float64()
		return v
	default:
		return n.f
	}
}

// Sign returns -1, 0 or +1 according to the sign of n. NaN reports 0.
func (n Num) Sign() int {
	switch n.kind {
	case kindInt:
		return n.bigInt().Sign()
	case kindRat:
		return n.r.Sign()
	default:
		switch {
		case n.f < 0:
			return -1
		case n.f > 0:
			return 1
		default:
			return 0
		}
	}
}

// IsZero reports whether n is exactly (or floating-point) zero.
func (n Num) IsZero() bool {
	switch n.kind {
	case kindInt:
		return n.bigInt().Sign() == 0
	case kindRat:
		return n.r.Sign() == 0
	default:
		return n.f == 0
	}
}

// IsFinite reports whether n is finite (always true for exact values; for Floats,
// not Inf and not NaN).
func (n Num) IsFinite() bool {
	if n.kind != kindFloat {
		return true
	}
	return !math.IsInf(n.f, 0) && !math.IsNaN(n.f)
}

// IsInfinite reports whether n is an infinite Float (exact values never are).
func (n Num) IsInfinite() bool {
	return n.kind == kindFloat && math.IsInf(n.f, 0)
}

// exactResult wraps an exact rational result with Ruby's kind rule: the result is
// an Integer only when both operands were Integers, otherwise it is a Rational
// (even when its value is whole, e.g. Rational(1,2)*2 == (1/1)).
func exactResult(a, b Num, r *big.Rat) Num {
	if a.kind == kindInt && b.kind == kindInt {
		// Integer×Integer (or ±) of exact integers: r.IsInt() always holds, except
		// for division, which Ruby keeps as a Rational — handled by numDiv directly.
		return Num{kind: kindInt, i: new(big.Int).Set(r.Num())}
	}
	return Num{kind: kindRat, r: new(big.Rat).Set(r)}
}

// numAdd, numSub, numMul, numDiv implement the numeric tower's binary ops with
// Ruby's coercion rule: if either operand is Float the result is Float; otherwise
// it is exact (Integer when both operands are Integer, else Rational).
func numAdd(a, b Num) Num {
	if a.kind == kindFloat || b.kind == kindFloat {
		return Float(a.Float64() + b.Float64())
	}
	return exactResult(a, b, new(big.Rat).Add(a.asRat(), b.asRat()))
}

func numSub(a, b Num) Num {
	if a.kind == kindFloat || b.kind == kindFloat {
		return Float(a.Float64() - b.Float64())
	}
	return exactResult(a, b, new(big.Rat).Sub(a.asRat(), b.asRat()))
}

func numMul(a, b Num) Num {
	if a.kind == kindFloat || b.kind == kindFloat {
		return Float(a.Float64() * b.Float64())
	}
	return exactResult(a, b, new(big.Rat).Mul(a.asRat(), b.asRat()))
}

// numDiv divides a by b. A Float operand, or exact division by zero (which Ruby
// turns into a Float Infinity/NaN), yields a Float. Exact division canonicalises
// its result (unlike +, -, *): a quotient that reduces to a whole number collapses
// to an Integer (so Complex(2,-2)/2 == (1-1i)), while a non-integral quotient is a
// Rational (Complex(10,2)/4 == ((5/2)+(1/2)i)). This mirrors Ruby's Complex#/.
func numDiv(a, b Num) Num {
	if a.kind == kindFloat || b.kind == kindFloat || b.IsZero() {
		return Float(a.Float64() / b.Float64())
	}
	return canonRat(new(big.Rat).Quo(a.asRat(), b.asRat()))
}

// canonRat returns an Integer Num when r is whole, else a Rational Num — the
// canonicalisation Ruby applies to the parts of a Complex division result.
func canonRat(r *big.Rat) Num {
	if r.IsInt() {
		return Num{kind: kindInt, i: new(big.Int).Set(r.Num())}
	}
	return Num{kind: kindRat, r: r}
}

// numNeg returns -n.
func numNeg(n Num) Num {
	switch n.kind {
	case kindInt:
		return Num{kind: kindInt, i: new(big.Int).Neg(n.bigInt())}
	case kindRat:
		return Num{kind: kindRat, r: new(big.Rat).Neg(n.r)}
	default:
		return Float(-n.f)
	}
}

// numEqual reports value equality across the tower (1 == 1.0 == 1/1), matching
// Ruby's `==` on numbers (used by Complex#==).
func numEqual(a, b Num) bool {
	if a.kind == kindFloat || b.kind == kindFloat {
		return a.Float64() == b.Float64()
	}
	return a.asRat().Cmp(b.asRat()) == 0
}

// numEql reports type-and-value equality (1.eql?(1.0) is false), matching Ruby's
// `eql?` which Complex#eql? applies component-wise.
func numEql(a, b Num) bool {
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case kindInt:
		return a.bigInt().Cmp(b.bigInt()) == 0
	case kindRat:
		return a.r.Cmp(b.r) == 0
	default:
		return a.f == b.f
	}
}

// ratNumDen returns the numerator and denominator of an exact Num as *big.Int.
func ratNumDen(n Num) (num, den *big.Int) {
	r := n.asRat()
	return new(big.Int).Set(r.Num()), new(big.Int).Set(r.Denom())
}
