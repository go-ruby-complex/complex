// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math"
	"testing"
)

// ratNum builds a Rational Num from num/den for tests.
func ratNum(num, den int64) Num { return RatFromInt64(num, den) }

// ci builds an integer-part Complex.
func ci(re, im int64) *Complex { return New(IntFromInt64(re), IntFromInt64(im)) }

func TestConstructorsAndAccessors(t *testing.T) {
	c := New(IntFromInt64(1), IntFromInt64(2))
	if c.Inspect() != "(1+2i)" || c.ToS() != "1+2i" {
		t.Fatalf("New = %s / %s", c.Inspect(), c.ToS())
	}
	if r := c.Real(); !r.IsInt() || r.Float64() != 1 {
		t.Errorf("Real = %v", r)
	}
	if im := c.Imaginary(); im.Float64() != 2 {
		t.Errorf("Imaginary = %v", im)
	}
	re, im := c.Rectangular()
	if re.Float64() != 1 || im.Float64() != 2 {
		t.Errorf("Rectangular = %v,%v", re, im)
	}
	// Rect is the alias of New.
	if Rect(IntFromInt64(3), IntFromInt64(4)).Inspect() != "(3+4i)" {
		t.Error("Rect mismatch")
	}
	// String() == Inspect()
	if c.String() != c.Inspect() {
		t.Error("String != Inspect")
	}
	// ToC returns self.
	if c.ToC() != c {
		t.Error("ToC not self")
	}
}

func TestArithmetic(t *testing.T) {
	if got := ci(1, 2).Add(ci(3, 4)).Inspect(); got != "(4+6i)" {
		t.Errorf("Add = %s", got)
	}
	if got := ci(1, 2).Sub(ci(3, 4)).Inspect(); got != "(-2-2i)" {
		t.Errorf("Sub = %s", got)
	}
	if got := ci(1, 2).Mul(ci(1, 2)).Inspect(); got != "(-3+4i)" {
		t.Errorf("Mul = %s", got)
	}
	if got := ci(1, 2).Div(ci(3, 4)).Inspect(); got != "((11/25)+(2/25)*i)" {
		t.Errorf("Div = %s", got)
	}
	// A quotient that divides evenly canonicalises to Integer parts (MRI Complex#/).
	if got := ci(2, 0).Div(ci(1, 1)).Inspect(); got != "(1-1i)" {
		t.Errorf("Div even = %s", got)
	}
	if got := ci(1, 2).Neg().Inspect(); got != "(-1-2i)" {
		t.Errorf("Neg = %s", got)
	}
	if got := ci(1, 2).Conjugate().Inspect(); got != "(1-2i)" {
		t.Errorf("Conjugate = %s", got)
	}
	// Float operand makes the result float.
	mixed := New(Float(1.0), IntFromInt64(2)).Add(ci(0, 0))
	if got := mixed.Inspect(); got != "(1.0+2i)" {
		t.Errorf("mixed Add = %s", got)
	}
	// Division by zero yields a Float (Infinity/NaN) part, not a panic.
	dz := ci(1, 2).Div(ci(0, 0))
	if dz.FiniteQ() {
		t.Errorf("div by zero finite? %s", dz.Inspect())
	}
}

func TestMagnitudeAndPhase(t *testing.T) {
	if got := ci(3, 4).Abs(); got != 5.0 {
		t.Errorf("Abs = %v", got)
	}
	if got := ci(3, 4).Abs2(); !got.IsInt() || got.Float64() != 25 {
		t.Errorf("Abs2 = %v", got)
	}
	if got := ci(1, 1).Arg(); math.Abs(got-math.Pi/4) > 1e-15 {
		t.Errorf("Arg = %v", got)
	}
	abs, arg := ci(3, 4).PolarParts()
	if abs != 5.0 || math.Abs(arg-math.Atan2(4, 3)) > 1e-15 {
		t.Errorf("PolarParts = %v,%v", abs, arg)
	}
}

func TestPolar(t *testing.T) {
	c := Polar(IntFromInt64(1), Float(0))
	re, im := c.Rectangular()
	if math.Abs(re.Float64()-1) > 1e-15 || math.Abs(im.Float64()) > 1e-15 {
		t.Errorf("Polar(1,0) = %s", c.Inspect())
	}
}

func TestPow(t *testing.T) {
	cases := []struct {
		c    *Complex
		exp  Num
		want string
	}{
		{ci(0, 1), IntFromInt64(2), "(-1+0i)"},
		{ci(2, 3), IntFromInt64(0), "(1+0i)"},
		{ci(2, 3), IntFromInt64(1), "(2+3i)"},
		{ci(2, 3), IntFromInt64(5), "(122-597i)"},
		{ci(2, 3), IntFromInt64(-1), "((2/13)-(3/13)*i)"},
		{ci(2, 3), IntFromInt64(-2), "((-5/169)-(12/169)*i)"},
		{ci(0, 0), IntFromInt64(0), "(1+0i)"},
		{ci(1, 2), IntFromInt64(10), "(237-3116i)"},
	}
	for _, c := range cases {
		if got := c.c.Pow(c.exp).Inspect(); got != c.want {
			t.Errorf("%s ** %v = %s, want %s", c.c.Inspect(), c.exp, got, c.want)
		}
	}
	// Float exponent path produces Float parts (value checked loosely; the exact
	// libm-vs-Go ULPs are an oracle concern).
	f := ci(2, 3).Pow(Float(2.0))
	if !f.Real().IsFloat() || math.Abs(f.Real().Float64()+5) > 1e-9 {
		t.Errorf("float pow = %s", f.Inspect())
	}
}

func TestEquality(t *testing.T) {
	if !ci(1, 2).Eql(ci(1, 2)) {
		t.Error("Eql same ints")
	}
	if ci(1, 2).Eql(New(Float(1), Float(2))) {
		t.Error("Eql int vs float should be false")
	}
	if !ci(1, 2).Equal(New(Float(1), Float(2))) {
		t.Error("Equal int vs float should be true")
	}
	if ci(1, 2).Equal(ci(1, 3)) {
		t.Error("Equal differing")
	}
	if New(ratNum(1, 2), IntFromInt64(0)).Eql(New(ratNum(1, 3), IntFromInt64(0))) {
		t.Error("Eql differing rats")
	}
}

func TestFiniteInfinite(t *testing.T) {
	if !ci(1, 2).FiniteQ() {
		t.Error("ints finite")
	}
	inf := New(Float(math.Inf(1)), IntFromInt64(1))
	if inf.FiniteQ() {
		t.Error("inf finite?")
	}
	if v, ok := inf.InfiniteQ(); v != 1 || !ok {
		t.Errorf("InfiniteQ inf = %v,%v", v, ok)
	}
	if v, ok := ci(1, 2).InfiniteQ(); v != 0 || ok {
		t.Errorf("InfiniteQ finite = %v,%v", v, ok)
	}
	// imaginary infinite also counts.
	if _, ok := New(IntFromInt64(1), Float(math.Inf(-1))).InfiniteQ(); !ok {
		t.Error("imag inf not infinite?")
	}
}

func TestConversions(t *testing.T) {
	// ToF
	if f, err := ci(5, 0).ToF(); err != nil || f != 5.0 {
		t.Errorf("ToF = %v,%v", f, err)
	}
	if _, err := ci(1, 2).ToF(); err == nil {
		t.Error("ToF non-real should error")
	}
	// ToI: integer, rational (truncate), float (truncate)
	if i, err := ci(6, 0).ToI(); err != nil || i.Int64() != 6 {
		t.Errorf("ToI int = %v,%v", i, err)
	}
	if i, err := New(ratNum(7, 2), IntFromInt64(0)).ToI(); err != nil || i.Int64() != 3 {
		t.Errorf("ToI rat = %v,%v", i, err)
	}
	if i, err := New(Float(3.9), IntFromInt64(0)).ToI(); err != nil || i.Int64() != 3 {
		t.Errorf("ToI float = %v,%v", i, err)
	}
	if _, err := ci(1, 2).ToI(); err == nil {
		t.Error("ToI non-real should error")
	}
	// ToR: integer, rational, float, and reject non-real
	if r, err := New(ratNum(3, 2), IntFromInt64(0)).ToR(); err != nil || !r.IsRat() || r.Float64() != 1.5 {
		t.Errorf("ToR rat = %v,%v", r, err)
	}
	if r, err := New(Float(2.0), IntFromInt64(0)).ToR(); err != nil || !r.IsExact() || r.Float64() != 2 {
		t.Errorf("ToR float = %v,%v", r, err)
	}
	if _, err := ci(2, 3).ToR(); err == nil {
		t.Error("ToR non-real should error")
	}
	// ToR of a non-finite float real has no rational value.
	if _, err := New(Float(math.Inf(1)), IntFromInt64(0)).ToR(); err == nil {
		t.Error("ToR inf should error")
	}
}

func TestNumeratorDenominator(t *testing.T) {
	// Integer parts: denominator 1, numerator itself.
	if d := ci(2, 3).Denominator(); !d.IsInt() || d.Float64() != 1 {
		t.Errorf("int Denominator = %v", d)
	}
	if n := ci(2, 3).Numerator().Inspect(); n != "(2+3i)" {
		t.Errorf("int Numerator = %s", n)
	}
	// Rational parts: common denominator.
	c := New(ratNum(1, 2), ratNum(3, 4))
	if d := c.Denominator(); d.Float64() != 4 {
		t.Errorf("Denominator = %v", d)
	}
	if n := c.Numerator().Inspect(); n != "(2+3i)" {
		t.Errorf("Numerator = %s", n)
	}
	c2 := New(ratNum(1, 3), ratNum(1, 6))
	if d := c2.Denominator(); d.Float64() != 6 {
		t.Errorf("Denominator2 = %v", d)
	}
	if n := c2.Numerator().Inspect(); n != "(2+1i)" {
		t.Errorf("Numerator2 = %s", n)
	}
}

func TestRangeErrorMessage(t *testing.T) {
	_, err := ci(1, 2).ToF()
	if err == nil || err.Error() != "can't convert Complex into Float" {
		t.Errorf("RangeError msg = %v", err)
	}
}
