// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// wantInt asserts that got is an Integer Num equal to want, and that its
// representation is canonical: inline (i == nil) exactly when want fits an int64,
// spilled to big.Int otherwise. This proves the int64 fast path and the
// overflow→big promotion agree with the arbitrary-precision reference *exactly*
// (never wrapping, never losing precision) while keeping the disjoint
// Fixnum/Bignum representation.
func wantInt(t *testing.T, ctx string, got Num, want *big.Int) {
	t.Helper()
	if !got.IsInt() {
		t.Fatalf("%s: kind = non-Integer %v, want Integer", ctx, got)
	}
	if got.bigInt().Cmp(want) != 0 {
		t.Fatalf("%s: value = %s, want %s", ctx, got.bigInt(), want)
	}
	inline := got.i == nil
	if inline != want.IsInt64() {
		t.Fatalf("%s: inline=%v but want.IsInt64()=%v (representation not canonical)", ctx, inline, want.IsInt64())
	}
}

// TestIntFromInt64IsInline confirms the machine-word constructor is allocation-free
// (inline representation, i == nil) yet reads back its value everywhere.
func TestIntFromInt64IsInline(t *testing.T) {
	n := IntFromInt64(42)
	if x, ok := n.int64Val(); !ok || x != 42 || n.i != nil {
		t.Fatalf("IntFromInt64 not inline: val=%d ok=%v i=%v", x, ok, n.i)
	}
	// The zero Num is inline integer 0.
	var z Num
	if x, ok := z.int64Val(); !ok || x != 0 {
		t.Fatalf("zero Num int64Val = %d,%v", x, ok)
	}
}

// TestIntConstructorCanonicalises verifies Int() stores a machine-word value inline
// and a larger one as big.Int, and that Int(nil) is inline zero.
func TestIntConstructorCanonicalises(t *testing.T) {
	if n := Int(big.NewInt(7)); n.i != nil {
		t.Errorf("Int(7) should be inline, got big")
	}
	big20 := bigStr(t, "100000000000000000000") // 10^20 > int64
	if n := Int(big20); n.i == nil {
		t.Errorf("Int(10^20) should spill to big.Int")
	}
	if n := Int(nil); n.i != nil || !n.IsZero() {
		t.Errorf("Int(nil) = %v, want inline zero", n)
	}
	// A big.Int that happens to fit int64 must canonicalise to inline.
	if n := Int(big.NewInt(math.MaxInt64)); n.i != nil {
		t.Errorf("Int(MaxInt64) should be inline")
	}
}

// TestBigIntAccessors exercises the big.Int (non-inline) branch of every accessor,
// deterministically (no Ruby oracle needed), so coverage of those branches holds on
// the no-ruby CI lanes.
func TestBigIntAccessors(t *testing.T) {
	pos := Int(bigStr(t, "100000000000000000000"))  // +10^20
	neg := Int(bigStr(t, "-100000000000000000000")) // -10^20
	if pos.i == nil || neg.i == nil {
		t.Fatal("test values must be big")
	}
	if pos.Sign() != 1 || neg.Sign() != -1 {
		t.Errorf("big Sign = %d,%d", pos.Sign(), neg.Sign())
	}
	if pos.IsZero() || neg.IsZero() {
		t.Error("big IsZero should be false")
	}
	if got := pos.Float64(); got != 1e20 {
		t.Errorf("big Float64 = %v, want 1e20", got)
	}
	if got := pos.bigInt().String(); got != "100000000000000000000" {
		t.Errorf("big bigInt = %s", got)
	}
	// numEql big branch (differing magnitudes / equal magnitudes).
	if !numEql(pos, Int(bigStr(t, "100000000000000000000"))) {
		t.Error("numEql equal bigs should be true")
	}
	if numEql(pos, neg) {
		t.Error("numEql differing bigs should be false")
	}
	// A big vs an inline int: unequal, and int64Pair reports not-both-inline.
	if numEql(pos, IntFromInt64(1)) {
		t.Error("numEql big vs inline should be false")
	}
}

// TestArithmeticFastPathMatchesBig cross-checks the int64 fast path against the
// arbitrary-precision reference over a wide range of operands, including the exact
// overflow boundaries, so any wrap or precision loss would be caught.
func TestArithmeticFastPathMatchesBig(t *testing.T) {
	const maxI, minI = int64(math.MaxInt64), int64(math.MinInt64)
	vals := []int64{
		0, 1, -1, 2, -2, 3, -4, 12, -12,
		maxI, minI, maxI - 1, minI + 1,
		1 << 31, -(1 << 31),
		3037000499, -3037000499, // floor(sqrt(2^63)); square fits
		3037000500, -3037000500, // square just overflows
		4000000000, -3000000000, // product magnitude in (2^63, 2^64): hi==0, lo>maxMag
		1 << 62, -2, // product == -(2^63) == MinInt64 exactly (fits, negative)
	}
	for _, x := range vals {
		for _, y := range vals {
			bx, by := big.NewInt(x), big.NewInt(y)
			wantInt(t, "add", numAdd(IntFromInt64(x), IntFromInt64(y)), new(big.Int).Add(bx, by))
			wantInt(t, "sub", numSub(IntFromInt64(x), IntFromInt64(y)), new(big.Int).Sub(bx, by))
			wantInt(t, "mul", numMul(IntFromInt64(x), IntFromInt64(y)), new(big.Int).Mul(bx, by))
		}
	}
}

// TestArithmeticFastPathRandom fuzzes the fast path against big for random operands.
func TestArithmeticFastPathRandom(t *testing.T) {
	r := rand.New(rand.NewSource(0xC0FFEE))
	for i := 0; i < 20000; i++ {
		x, y := int64(r.Uint64()), int64(r.Uint64())
		bx, by := big.NewInt(x), big.NewInt(y)
		wantInt(t, "add", numAdd(IntFromInt64(x), IntFromInt64(y)), new(big.Int).Add(bx, by))
		wantInt(t, "sub", numSub(IntFromInt64(x), IntFromInt64(y)), new(big.Int).Sub(bx, by))
		wantInt(t, "mul", numMul(IntFromInt64(x), IntFromInt64(y)), new(big.Int).Mul(bx, by))
	}
}

// TestOverflowBoundaryExact pins the exact-promotion boundary values MRI would hit:
// each overflowing op yields the precise (big) result, never a wrapped int64.
func TestOverflowBoundaryExact(t *testing.T) {
	cases := []struct {
		op   string
		x, y int64
		want string // exact decimal of the true result
	}{
		{"add", math.MaxInt64, 1, "9223372036854775808"},   // 2^63
		{"add", math.MinInt64, -1, "-9223372036854775809"}, // -(2^63+1)
		{"sub", math.MinInt64, 1, "-9223372036854775809"},
		{"sub", math.MaxInt64, math.MinInt64, "18446744073709551615"}, // 2^64-1
		{"mul", math.MaxInt64, math.MaxInt64, "85070591730234615847396907784232501249"},
		{"mul", 3037000500, 3037000500, "9223372037000250000"},    // pos, hi==0, lo>=maxMag
		{"mul", 4000000000, -3000000000, "-12000000000000000000"}, // neg, hi==0, lo>maxMag
	}
	for _, c := range cases {
		var got Num
		switch c.op {
		case "add":
			got = numAdd(IntFromInt64(c.x), IntFromInt64(c.y))
		case "sub":
			got = numSub(IntFromInt64(c.x), IntFromInt64(c.y))
		case "mul":
			got = numMul(IntFromInt64(c.x), IntFromInt64(c.y))
		}
		want := bigStr(t, c.want)
		wantInt(t, c.op, got, want)
		if got.i == nil {
			t.Errorf("%s(%d,%d): expected big promotion, stayed inline", c.op, c.x, c.y)
		}
	}
}

// TestMulMinInt64ExactlyFits checks the delicate case where a product equals
// math.MinInt64 exactly: it must stay inline (it fits) and equal -(2^63).
func TestMulMinInt64ExactlyFits(t *testing.T) {
	got := numMul(IntFromInt64(1<<62), IntFromInt64(-2)) // 2^62 * -2 == -(2^63)
	if got.i != nil {
		t.Fatalf("MinInt64 product should fit inline, got big")
	}
	if x, _ := got.int64Val(); x != math.MinInt64 {
		t.Fatalf("product = %d, want MinInt64", x)
	}
}

// TestNumNegBoundary covers negation across inline, MinInt64 promotion, and big.
func TestNumNegBoundary(t *testing.T) {
	if got := numNeg(IntFromInt64(5)); got.i != nil || got.bigInt().Int64() != -5 {
		t.Errorf("neg inline = %v", got)
	}
	// -MinInt64 == 2^63 overflows the signed range → promotes to big.
	got := numNeg(IntFromInt64(math.MinInt64))
	if got.i == nil {
		t.Fatalf("neg(MinInt64) should promote to big")
	}
	if got.bigInt().String() != "9223372036854775808" {
		t.Errorf("neg(MinInt64) = %s", got.bigInt())
	}
	// Negating a big value stays big and negates exactly.
	nb := numNeg(Int(bigStr(t, "100000000000000000000")))
	if nb.i == nil || nb.bigInt().String() != "-100000000000000000000" {
		t.Errorf("neg(big) = %v", nb)
	}
}

// TestMixedInlineBigArithmetic exercises the path where one operand is a big Integer:
// the int64 fast path declines (int64Pair not-ok) and the exact big.Rat tower path
// takes over, still yielding a canonical Integer.
func TestMixedInlineBigArithmetic(t *testing.T) {
	big20 := Int(bigStr(t, "100000000000000000000"))
	sum := numAdd(big20, IntFromInt64(1))
	wantInt(t, "big+inline", sum, bigStr(t, "100000000000000000001"))
	prod := numMul(big20, IntFromInt64(3))
	wantInt(t, "big*inline", prod, bigStr(t, "300000000000000000000"))
	// A difference that collapses back into int64 range must canonicalise to inline.
	diff := numSub(big20, Int(bigStr(t, "99999999999999999999")))
	if diff.i != nil {
		t.Errorf("big-big collapsing to small should be inline, got big %v", diff)
	}
	wantInt(t, "big-big", diff, big.NewInt(1))
}

// TestComplexArithmeticExactWithLargeParts confirms Complex ops stay MRI-exact even
// when a partial product overflows int64 mid-computation (promoting to big), matching
// the always-big result byte-for-byte in inspect form.
func TestComplexArithmeticExactWithLargeParts(t *testing.T) {
	// (3037000500 + i)² has real part 3037000500² - 1 = 9223372037000249999 which
	// overflows int64 during Mul; the result must be the exact big value.
	a := New(IntFromInt64(3037000500), IntFromInt64(1))
	got := a.Mul(a).Inspect()
	want := "(9223372037000249999+6074001000i)"
	if got != want {
		t.Errorf("large Mul = %s, want %s", got, want)
	}
}

// TestNumEqualFastPath covers the both-inline int64 equality shortcut and the
// fallback to the rational compare for a mixed int/rational pair.
func TestNumEqualFastPath(t *testing.T) {
	if !numEqual(IntFromInt64(7), IntFromInt64(7)) || numEqual(IntFromInt64(7), IntFromInt64(8)) {
		t.Error("inline == shortcut")
	}
	if !numEqual(IntFromInt64(2), ratNum(4, 2)) {
		t.Error("int == rational fallback")
	}
	// A big Integer vs an equal big Integer (non-inline compare via asRat).
	if !numEqual(Int(bigStr(t, "100000000000000000000")), Int(bigStr(t, "100000000000000000000"))) {
		t.Error("big == big")
	}
}
