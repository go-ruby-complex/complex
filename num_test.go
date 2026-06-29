// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math"
	"math/big"
	"testing"
)

func TestNumConstructorsNil(t *testing.T) {
	// Int(nil) and Rat(nil) treat the nil pointer as zero.
	if n := Int(nil); !n.IsInt() || !n.IsZero() {
		t.Errorf("Int(nil) = %v", n)
	}
	if n := Rat(nil); !n.IsZero() {
		t.Errorf("Rat(nil) = %v", n)
	}
	// Rat keeps the Rational kind even when the value is whole — Ruby's Rational
	// does not collapse N/1 to an Integer (Rational(6,3) is (2/1), a Rational).
	if n := Rat(big.NewRat(6, 3)); !n.IsRat() || n.Float64() != 2 {
		t.Errorf("Rat(6/3) = %v (want Rat 2/1)", n)
	}
	// A Num with a nil big.Int (the zero struct) behaves as integer 0.
	var z Num
	if !z.IsInt() || z.Sign() != 0 || z.Float64() != 0 || !z.IsZero() {
		t.Errorf("zero Num = %v", z)
	}
}

func TestNumKindsAndPredicates(t *testing.T) {
	i := IntFromInt64(-3)
	r := ratNum(1, 2)
	f := Float(2.5)
	if !i.IsInt() || i.IsRat() || i.IsFloat() || !i.IsExact() {
		t.Error("int predicates")
	}
	if r.IsInt() || !r.IsRat() || r.IsFloat() || !r.IsExact() {
		t.Error("rat predicates")
	}
	if f.IsInt() || f.IsRat() || !f.IsFloat() || f.IsExact() {
		t.Error("float predicates")
	}
}

func TestNumSign(t *testing.T) {
	if IntFromInt64(-3).Sign() != -1 || IntFromInt64(0).Sign() != 0 || IntFromInt64(3).Sign() != 1 {
		t.Error("int sign")
	}
	if ratNum(-1, 2).Sign() != -1 || ratNum(1, 2).Sign() != 1 {
		t.Error("rat sign")
	}
	if Float(-1).Sign() != -1 || Float(1).Sign() != 1 || Float(0).Sign() != 0 {
		t.Error("float sign")
	}
	if Float(math.NaN()).Sign() != 0 {
		t.Error("nan sign")
	}
}

func TestNumIsZeroAndFloat64(t *testing.T) {
	if !ratNum(0, 5).IsZero() || ratNum(1, 5).IsZero() {
		t.Error("rat IsZero")
	}
	if !Float(0).IsZero() || Float(0.1).IsZero() {
		t.Error("float IsZero")
	}
	if ratNum(1, 2).Float64() != 0.5 {
		t.Error("rat Float64")
	}
	if IntFromInt64(7).Float64() != 7 {
		t.Error("int Float64")
	}
}

func TestNumFiniteInfinite(t *testing.T) {
	if !IntFromInt64(1).IsFinite() || IntFromInt64(1).IsInfinite() {
		t.Error("int finite")
	}
	if !ratNum(1, 2).IsFinite() {
		t.Error("rat finite")
	}
	if Float(math.Inf(1)).IsFinite() || !Float(math.Inf(1)).IsInfinite() {
		t.Error("inf")
	}
	if Float(math.NaN()).IsFinite() {
		t.Error("nan finite")
	}
	if Float(1).IsInfinite() {
		t.Error("finite float infinite?")
	}
}

func TestNumArithmeticTower(t *testing.T) {
	// Exact stays exact; any float promotes.
	if got := numAdd(IntFromInt64(1), ratNum(1, 2)); !got.IsRat() || got.Float64() != 1.5 {
		t.Errorf("Add exact = %v", got)
	}
	if got := numAdd(IntFromInt64(1), Float(0.5)); !got.IsFloat() || got.Float64() != 1.5 {
		t.Errorf("Add float = %v", got)
	}
	if got := numSub(Float(1), IntFromInt64(2)); !got.IsFloat() || got.Float64() != -1 {
		t.Errorf("Sub float = %v", got)
	}
	if got := numSub(IntFromInt64(5), IntFromInt64(2)); got.Float64() != 3 {
		t.Errorf("Sub exact = %v", got)
	}
	if got := numMul(Float(2), IntFromInt64(3)); !got.IsFloat() || got.Float64() != 6 {
		t.Errorf("Mul float = %v", got)
	}
	if got := numMul(IntFromInt64(2), IntFromInt64(3)); got.Float64() != 6 {
		t.Errorf("Mul exact = %v", got)
	}
	if got := numDiv(IntFromInt64(1), IntFromInt64(2)); !got.IsRat() || got.Float64() != 0.5 {
		t.Errorf("Div exact = %v", got)
	}
	if got := numDiv(Float(1), IntFromInt64(2)); !got.IsFloat() {
		t.Errorf("Div float = %v", got)
	}
	if got := numDiv(IntFromInt64(1), IntFromInt64(0)); !got.IsFloat() || !math.IsInf(got.f, 1) {
		t.Errorf("Div by zero = %v", got)
	}
}

func TestNumNeg(t *testing.T) {
	if numNeg(IntFromInt64(3)).Float64() != -3 {
		t.Error("neg int")
	}
	if numNeg(ratNum(1, 2)).Float64() != -0.5 {
		t.Error("neg rat")
	}
	if numNeg(Float(2.5)).Float64() != -2.5 {
		t.Error("neg float")
	}
}

func TestNumEqualityAcrossKinds(t *testing.T) {
	if !numEqual(IntFromInt64(1), Float(1.0)) {
		t.Error("== int/float")
	}
	if !numEqual(IntFromInt64(1), ratNum(2, 2)) {
		t.Error("== int/rat")
	}
	if numEqual(IntFromInt64(1), IntFromInt64(2)) {
		t.Error("== differing")
	}
	if numEql(IntFromInt64(1), Float(1.0)) {
		t.Error("eql int/float should be false")
	}
	if !numEql(Float(1.0), Float(1.0)) {
		t.Error("eql float/float")
	}
	if !numEql(ratNum(1, 2), ratNum(1, 2)) {
		t.Error("eql rat/rat")
	}
	if numEql(ratNum(1, 2), ratNum(1, 3)) {
		t.Error("eql differing rat")
	}
}
