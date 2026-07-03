// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"errors"
	"math"
	"math/big"
	"math/bits"
)

// Num models a single component (real or imaginary) of a Ruby Complex. Ruby keeps
// the parts *exact* whenever it can: an Integer part stays Integer (arbitrary
// precision), a Rational part stays Rational, and only a Float part becomes Float.
// Num is the small numeric union that preserves that tower so this package can be
// byte-exact with MRI's `inspect` / `to_s` (e.g. Complex(1,2)*Complex(1,2) is the
// *exact* (-3+4i), and Complex(Rational(1,2), …) renders 1/2 not 0.5).
//
// An Integer component (kind==kindInt) has two disjoint representations, mirroring
// MRI's Fixnum/Bignum split: a machine-word value is stored inline in the int64
// field small (i == nil), and only a value that does not fit an int64 spills to the
// arbitrary-precision big.Int i. Keeping the common machine-word case allocation-free
// is what lets add/sub/mul run as int64 arithmetic (with overflow promotion to i) on
// the hot path instead of always routing through math/big — exactly MRI's Fixnum fast
// path with Bignum promotion. The representation is canonical: i is non-nil *iff* the
// value truly exceeds int64, so equal Integers always share a representation.
//
// The zero Num is the integer 0 (kind==kindInt, i==nil, small==0).
type Num struct {
	kind kind
	// small holds an Integer value that fits an int64, when kind==kindInt && i==nil.
	small int64
	// i holds an Integer value too large for int64 (kind==kindInt && i!=nil). A nil i
	// with kind==kindInt means the value is small.
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

// Int returns a Num holding the arbitrary-precision integer v. A value that fits an
// int64 is stored inline (allocation-free); only a larger value keeps the big.Int.
func Int(v *big.Int) Num {
	if v == nil {
		return Num{kind: kindInt}
	}
	return intNum(v)
}

// IntFromInt64 returns a Num holding the integer v. It is allocation-free — the value
// is kept inline in the int64 fast-path representation.
func IntFromInt64(v int64) Num { return Num{kind: kindInt, small: v} }

// intNum returns a canonical Integer Num from v: stored inline when it fits an int64,
// else as a freshly copied big.Int. Canonicalising keeps the inline and big
// representations disjoint, so equal Integers always share a representation and the
// int64 fast paths fire whenever the value is machine-word sized.
func intNum(v *big.Int) Num {
	if v.IsInt64() {
		return Num{kind: kindInt, small: v.Int64()}
	}
	return Num{kind: kindInt, i: new(big.Int).Set(v)}
}

// int64Val reports whether n is an Integer stored inline, returning its int64 value.
func (n Num) int64Val() (int64, bool) {
	if n.kind == kindInt && n.i == nil {
		return n.small, true
	}
	return 0, false
}

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

// bigInt returns the underlying value of an Integer Num as a *big.Int, materialising
// the inline int64 (or the zero Num) into a fresh big.Int when needed.
func (n Num) bigInt() *big.Int {
	if n.i != nil {
		return n.i
	}
	return big.NewInt(n.small)
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
		if n.i == nil {
			// int64→float64 rounds to nearest-even, matching big.Float's rounding
			// (and MRI's Integer#to_f) for every value the inline path represents.
			return float64(n.small)
		}
		f := new(big.Float).SetInt(n.i)
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
		if n.i != nil {
			return n.i.Sign()
		}
		switch {
		case n.small < 0:
			return -1
		case n.small > 0:
			return 1
		default:
			return 0
		}
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
		if n.i != nil {
			return n.i.Sign() == 0
		}
		return n.small == 0
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
		return intNum(r.Num())
	}
	return Num{kind: kindRat, r: new(big.Rat).Set(r)}
}

// numAdd, numSub, numMul, numDiv implement the numeric tower's binary ops with
// Ruby's coercion rule: if either operand is Float the result is Float; otherwise
// it is exact (Integer when both operands are Integer, else Rational).
//
// When both operands are inline machine-word Integers, add/sub/mul take an int64 fast
// path (with overflow detection via math/bits) and never touch math/big — the common
// case. On overflow they promote to an exact big.Int, so precision is never lost: the
// result is byte-identical to the always-big path, matching MRI's Fixnum→Bignum
// promotion. Any non-inline (large-Integer, Rational or Float) operand keeps the
// original exact tower path unchanged.
func numAdd(a, b Num) Num {
	if x, y, ok := int64Pair(a, b); ok {
		if s, ok := addInt64(x, y); ok {
			return Num{kind: kindInt, small: s}
		}
		return intNum(new(big.Int).Add(big.NewInt(x), big.NewInt(y)))
	}
	if a.kind == kindFloat || b.kind == kindFloat {
		return Float(a.Float64() + b.Float64())
	}
	return exactResult(a, b, new(big.Rat).Add(a.asRat(), b.asRat()))
}

func numSub(a, b Num) Num {
	if x, y, ok := int64Pair(a, b); ok {
		if d, ok := subInt64(x, y); ok {
			return Num{kind: kindInt, small: d}
		}
		return intNum(new(big.Int).Sub(big.NewInt(x), big.NewInt(y)))
	}
	if a.kind == kindFloat || b.kind == kindFloat {
		return Float(a.Float64() - b.Float64())
	}
	return exactResult(a, b, new(big.Rat).Sub(a.asRat(), b.asRat()))
}

func numMul(a, b Num) Num {
	if x, y, ok := int64Pair(a, b); ok {
		if p, ok := mulInt64(x, y); ok {
			return Num{kind: kindInt, small: p}
		}
		return intNum(new(big.Int).Mul(big.NewInt(x), big.NewInt(y)))
	}
	if a.kind == kindFloat || b.kind == kindFloat {
		return Float(a.Float64() * b.Float64())
	}
	return exactResult(a, b, new(big.Rat).Mul(a.asRat(), b.asRat()))
}

// int64Pair reports whether both a and b are inline machine-word Integers, returning
// their int64 values for the arithmetic fast path.
func int64Pair(a, b Num) (x, y int64, ok bool) {
	xa, oka := a.int64Val()
	xb, okb := b.int64Val()
	return xa, xb, oka && okb
}

// addInt64 returns x+y and whether it fit an int64 (no signed overflow). The sum is
// formed with bits.Add64; overflow is the classic same-sign-operands, differing-sign
// result condition.
func addInt64(x, y int64) (int64, bool) {
	s, _ := bits.Add64(uint64(x), uint64(y), 0)
	sum := int64(s)
	if (x < 0) == (y < 0) && (sum < 0) != (x < 0) {
		return 0, false
	}
	return sum, true
}

// subInt64 returns x-y and whether it fit an int64. The difference is formed with
// bits.Sub64; overflow occurs only when the operands differ in sign and the result's
// sign differs from x's.
func subInt64(x, y int64) (int64, bool) {
	d, _ := bits.Sub64(uint64(x), uint64(y), 0)
	diff := int64(d)
	if (x < 0) != (y < 0) && (diff < 0) != (x < 0) {
		return 0, false
	}
	return diff, true
}

// mulInt64 returns x*y and whether it fit an int64. It multiplies the magnitudes with
// bits.Mul64 and rejects any 128-bit product whose high word is non-zero or whose low
// word exceeds the signed range for the result's sign (|MinInt64| == 1<<63 is the
// largest representable negative magnitude, MaxInt64 == 1<<63 - 1 the positive one).
func mulInt64(x, y int64) (int64, bool) {
	hi, lo := bits.Mul64(uabs(x), uabs(y))
	if hi != 0 {
		return 0, false
	}
	const maxMag = uint64(1) << 63 // |math.MinInt64|
	if (x < 0) != (y < 0) {
		if lo > maxMag {
			return 0, false
		}
		return int64(-lo), true // lo==maxMag negates (mod 2^64) to MinInt64 exactly
	}
	if lo >= maxMag {
		return 0, false
	}
	return int64(lo), true
}

// uabs returns the magnitude of x as a uint64, correct even for math.MinInt64 (whose
// magnitude 1<<63 is not representable as a positive int64).
func uabs(x int64) uint64 {
	u := uint64(x)
	if x < 0 {
		u = -u
	}
	return u
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
		return intNum(r.Num())
	}
	return Num{kind: kindRat, r: r}
}

// numNeg returns -n.
func numNeg(n Num) Num {
	switch n.kind {
	case kindInt:
		if x, ok := n.int64Val(); ok && x != math.MinInt64 {
			// -x fits an int64 for every inline value except MinInt64 (whose negation
			// overflows the signed range and must promote to big).
			return Num{kind: kindInt, small: -x}
		}
		return intNum(new(big.Int).Neg(n.bigInt()))
	case kindRat:
		return Num{kind: kindRat, r: new(big.Rat).Neg(n.r)}
	default:
		return Float(-n.f)
	}
}

// numEqual reports value equality across the tower (1 == 1.0 == 1/1), matching
// Ruby's `==` on numbers (used by Complex#==).
func numEqual(a, b Num) bool {
	if x, y, ok := int64Pair(a, b); ok {
		return x == y
	}
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
		// Integers are canonical: both inline iff both fit int64, so a cheap int64
		// compare settles the common case; a big operand means unequal magnitudes.
		if x, y, ok := int64Pair(a, b); ok {
			return x == y
		}
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
