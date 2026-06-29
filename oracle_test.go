// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"os/exec"
	"strings"
	"testing"
)

// rubyBin locates a usable `ruby` whose RUBY_VERSION is >= "4.0" (the byte layout
// this package targets is MRI 4.0.5's). The oracle tests skip themselves when no
// such interpreter is present — the Windows lane, the qemu cross-arch lanes, and
// any host with an older Ruby — so the deterministic, ruby-free suite alone drives
// the 100% coverage gate there.
func rubyBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI oracle")
	}
	out, err := exec.Command(path, "-e", "print RUBY_VERSION").Output()
	if err != nil {
		t.Skipf("cannot query ruby version: %v", err)
	}
	if v := string(out); v < "4.0" {
		t.Skipf("ruby %s < 4.0; skipping MRI 4.x oracle", v)
	}
	return path
}

// rubyComplex evaluates `Complex(<expr>)` style scripts and returns the trimmed
// stdout. The script binmodes stdin and stdout so Windows text-mode never rewrites
// the bytes (the go-ruby-erb lesson); the no-ruby/Windows lanes skip via rubyBin.
func rubyComplex(t *testing.T, bin, script string) string {
	t.Helper()
	full := "$stdout.binmode\n$stdin.binmode\n" + script
	out, err := exec.Command(bin, "-e", full).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return strings.TrimRight(string(out), "\r\n")
}

// TestOracleInspectToS checks our inspect / to_s bytes against MRI for a corpus
// spanning every component shape (Integer, Rational, Float, signs, zeros, the
// non-finite floats). The corpus is built here as Go Complex values and as the
// equivalent Ruby Complex(...) expression, and the two renderings are compared
// byte for byte.
func TestOracleInspectToS(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct {
		c    *Complex
		ruby string // Ruby expression evaluating to the same Complex
	}{
		{ci(1, 2), "Complex(1,2)"},
		{ci(1, -2), "Complex(1,-2)"},
		{ci(-1, 2), "Complex(-1,2)"},
		{ci(-1, -2), "Complex(-1,-2)"},
		{ci(0, 1), "Complex(0,1)"},
		{ci(0, -1), "Complex(0,-1)"},
		{ci(1, 0), "Complex(1,0)"},
		{ci(0, 0), "Complex(0,0)"},
		{New(Float(1.5), Float(2.5)), "Complex(1.5,2.5)"},
		{New(Float(1.0), Float(2.0)), "Complex(1.0,2.0)"},
		{New(Float(-1.5), Float(-2.5)), "Complex(-1.5,-2.5)"},
		{New(ratNum(1, 2), ratNum(3, 4)), "Complex(Rational(1,2),Rational(3,4))"},
		{New(ratNum(-1, 2), ratNum(3, 4)), "Complex(Rational(-1,2),Rational(3,4))"},
		{New(ratNum(1, 2), ratNum(-3, 4)), "Complex(Rational(1,2),Rational(-3,4))"},
		{New(IntFromInt64(0), ratNum(3, 4)), "Complex(0,Rational(3,4))"},
		{New(IntFromInt64(1), ratNum(-3, 4)), "Complex(1,Rational(-3,4))"},
		{New(Int(bigStr(t, "123456789012345678901234567890")), IntFromInt64(2)),
			"Complex(123456789012345678901234567890,2)"},
		{New(Float(infP()), IntFromInt64(1)), "Complex(Float::INFINITY,1)"},
		{New(IntFromInt64(1), Float(nan())), "Complex(1,Float::NAN)"},
		{New(Float(infN()), IntFromInt64(2)), "Complex(-Float::INFINITY,2)"},
		{New(IntFromInt64(1), Float(infN())), "Complex(1,-Float::INFINITY)"},
	}
	for _, c := range cases {
		want := rubyComplex(t, bin, "c="+c.ruby+"\nprint c.inspect, \"\\n\", c.to_s")
		got := c.c.Inspect() + "\n" + c.c.ToS()
		if got != want {
			t.Errorf("%s: got %q, want %q", c.ruby, got, want)
		}
	}
}

// TestOracleArithmetic checks exact-preserving arithmetic against MRI: every result
// here must inspect byte-identically, proving the parts stay on the numeric tower.
func TestOracleArithmetic(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct {
		got  *Complex
		ruby string
	}{
		{ci(1, 2).Add(ci(3, 4)), "Complex(1,2)+Complex(3,4)"},
		{ci(1, 2).Sub(ci(3, 4)), "Complex(1,2)-Complex(3,4)"},
		{ci(1, 2).Mul(ci(1, 2)), "Complex(1,2)*Complex(1,2)"},
		{ci(1, 2).Div(ci(3, 4)), "Complex(1,2)/Complex(3,4)"},
		{ci(1, 2).Neg(), "-Complex(1,2)"},
		{ci(1, 2).Conjugate(), "Complex(1,2).conjugate"},
		{ci(0, 1).Pow(IntFromInt64(2)), "Complex(0,1)**2"},
		{ci(2, 3).Pow(IntFromInt64(5)), "Complex(2,3)**5"},
		{ci(2, 3).Pow(IntFromInt64(-2)), "Complex(2,3)**-2"},
		{ci(1, 2).Pow(IntFromInt64(10)), "Complex(1,2)**10"},
		{New(ratNum(1, 2), ratNum(3, 4)).Mul(ci(2, 0)), "Complex(Rational(1,2),Rational(3,4))*Complex(2,0)"},
	}
	for _, c := range cases {
		want := rubyComplex(t, bin, "print ("+c.ruby+").inspect")
		if got := c.got.Inspect(); got != want {
			t.Errorf("%s: got %q, want %q", c.ruby, got, want)
		}
	}
}

// TestOracleAbsArg checks magnitude/phase (which MRI computes via libm hypot/atan2,
// the same routines Go's math package uses) match exactly.
func TestOracleAbsArg(t *testing.T) {
	bin := rubyBin(t)
	cases := []struct {
		got  float64
		ruby string
	}{
		{ci(3, 4).Abs(), "Complex(3,4).abs"},
		{ci(1, 1).Arg(), "Complex(1,1).arg"},
		{ci(-1, 0).Arg(), "Complex(-1,0).arg"},
		{ci(0, 0).Arg(), "Complex(0,0).arg"},
	}
	for _, c := range cases {
		want := rubyComplex(t, bin, "printf \"%.17g\", ("+c.ruby+")")
		if got := formatG17(c.got); got != want {
			t.Errorf("%s: got %s, want %s", c.ruby, got, want)
		}
	}
}

// TestOracleAbs2Numerator checks the exact abs2 and numerator/denominator.
func TestOracleAbs2Numerator(t *testing.T) {
	bin := rubyBin(t)
	if got, want := numToRuby(t, ci(3, 4).Abs2()), rubyComplex(t, bin, "print Complex(3,4).abs2"); got != want {
		t.Errorf("abs2 = %s, want %s", got, want)
	}
	c := New(ratNum(1, 2), ratNum(3, 4))
	if got, want := c.Numerator().Inspect(), rubyComplex(t, bin, "print Complex(Rational(1,2),Rational(3,4)).numerator.inspect"); got != want {
		t.Errorf("numerator = %s, want %s", got, want)
	}
	if got, want := numToRuby(t, c.Denominator()), rubyComplex(t, bin, "print Complex(Rational(1,2),Rational(3,4)).denominator"); got != want {
		t.Errorf("denominator = %s, want %s", got, want)
	}
}

// TestOracleParse checks Parse against MRI's Complex(string) for the accepted forms
// (exact ones compared byte-for-byte; the polar form is transcendental and verified
// in the deterministic suite under a tolerance).
func TestOracleParse(t *testing.T) {
	bin := rubyBin(t)
	inputs := []string{
		"1+2i", "1-2i", "-1+2i", "i", "-i", "3i", "1/2+3/4i", "2.5-1.5i",
		"1.5e3+2i", "2+i", "1+2j", "5+0i", ".5+.5i", "2e-3i", "1_000+2i",
	}
	for _, in := range inputs {
		c, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		want := rubyComplex(t, bin, "print Complex("+rubyStr(in)+").inspect")
		if got := c.Inspect(); got != want {
			t.Errorf("Parse(%q) = %s, want %s", in, got, want)
		}
	}
}

// numToRuby renders an exact Num the way Ruby's Integer/Rational inspect would, for
// oracle comparison (Integers bare, Rationals as "n/d").
func numToRuby(t *testing.T, n Num) string {
	t.Helper()
	switch {
	case n.IsInt():
		return n.bigInt().String()
	case n.IsRat():
		return n.r.Num().String() + "/" + n.r.Denom().String()
	default:
		t.Fatalf("numToRuby on non-exact %v", n)
		return ""
	}
}

// rubyStr quotes a Go string as a Ruby double-quoted literal for the oracle script.
func rubyStr(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
