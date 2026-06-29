// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math"
	"testing"
)

func TestParseValid(t *testing.T) {
	cases := map[string]string{
		"1+2i":     "(1+2i)",
		"1-2i":     "(1-2i)",
		"-1+2i":    "(-1+2i)",
		"1":        "(1+0i)",
		"i":        "(0+1i)",
		"-i":       "(0-1i)",
		"+i":       "(0+1i)",
		"3i":       "(0+3i)",
		"-3i":      "(0-3i)",
		"+2i":      "(0+2i)",
		"2i":       "(0+2i)",
		"1/2+3/4i": "((1/2)+(3/4)*i)",
		"2.5-1.5i": "(2.5-1.5i)",
		"1.5e3+2i": "(1500.0+2i)",
		"2+i":      "(2+1i)",
		"2-i":      "(2-1i)",
		"1+2j":     "(1+2i)",
		"1.0":      "(1.0+0i)",
		"0":        "(0+0i)",
		".5+.5i":   "(0.5+0.5i)",
		"1+2.5i":   "(1+2.5i)",
		"3.14":     "(3.14+0i)",
		"1e5":      "(100000.0+0i)",
		"5+0i":     "(5+0i)",
		"1_000+2i": "(1000+2i)",
		"2e-3i":    "(0+0.002i)",
		"3.0i":     "(0+3.0i)",
		"  1+2i  ": "(1+2i)",
		"1+2i\n":   "(1+2i)",
		"\t1+2i":   "(1+2i)",
	}
	for in, want := range cases {
		c, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", in, err)
			continue
		}
		if got := c.Inspect(); got != want {
			t.Errorf("Parse(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestParsePolar(t *testing.T) {
	// Polar parse produces Float parts; value compared loosely (libm vs Go ULPs).
	c, err := Parse("1@2")
	if err != nil {
		t.Fatalf("Parse polar: %v", err)
	}
	re, im := c.Rectangular()
	if math.Abs(re.Float64()-math.Cos(2)) > 1e-9 || math.Abs(im.Float64()-math.Sin(2)) > 1e-9 {
		t.Errorf("polar = %s", c.Inspect())
	}
	// float@float as well.
	if _, err := Parse("1.0@0.5"); err != nil {
		t.Errorf("Parse(1.0@0.5): %v", err)
	}
}

func TestParseInvalid(t *testing.T) {
	for _, in := range []string{
		" 1 + 2i ", "hello", "", "1+", "1+2", "abc+2i", "0xff", "--1",
		"1+-2i", "i+1", "1+i2", "1+0xi", "Infinity", "1 +2i", "1/0+2i",
		"@", "1@", "/2", "1.", "+", "-", "1@x", "3/", "1e", "1ee",
		"1_+2i", "1__0+2i", "1/0",
	} {
		if c, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) should fail, got %s", in, c.Inspect())
		}
	}
}

func TestParseErrorMessage(t *testing.T) {
	_, err := Parse("nope")
	pe, ok := err.(*ParseError)
	if !ok || pe.Error() != `invalid value for convert(): "nope"` {
		t.Errorf("ParseError = %v", err)
	}
}
