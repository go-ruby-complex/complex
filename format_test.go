// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math"
	"testing"
)

// TestFormatCorpus pins the inspect/to_s byte layout for every component shape
// against the values captured from MRI 4.0.5 (see oracle_test.go for the live
// differential check).
func TestFormatCorpus(t *testing.T) {
	cases := []struct {
		c            *Complex
		inspect, toS string
	}{
		{ci(1, 2), "(1+2i)", "1+2i"},
		{ci(1, -2), "(1-2i)", "1-2i"},
		{ci(-1, 2), "(-1+2i)", "-1+2i"},
		{ci(-1, -2), "(-1-2i)", "-1-2i"},
		{ci(0, 1), "(0+1i)", "0+1i"},
		{ci(0, -1), "(0-1i)", "0-1i"},
		{ci(1, 0), "(1+0i)", "1+0i"},
		{New(Float(1.5), Float(2.5)), "(1.5+2.5i)", "1.5+2.5i"},
		{New(Float(1.0), Float(2.0)), "(1.0+2.0i)", "1.0+2.0i"},
		{New(Float(-1.5), Float(-2.5)), "(-1.5-2.5i)", "-1.5-2.5i"},
		{New(ratNum(1, 2), ratNum(3, 4)), "((1/2)+(3/4)*i)", "1/2+3/4i"},
		{New(ratNum(-1, 2), ratNum(3, 4)), "((-1/2)+(3/4)*i)", "-1/2+3/4i"},
		{New(ratNum(1, 2), ratNum(-3, 4)), "((1/2)-(3/4)*i)", "1/2-3/4i"},
		{New(IntFromInt64(0), ratNum(3, 4)), "(0+(3/4)*i)", "0+3/4i"},
		{New(IntFromInt64(1), ratNum(-3, 4)), "(1-(3/4)*i)", "1-3/4i"},
		{New(Float(math.Inf(1)), IntFromInt64(1)), "(Infinity+1i)", "Infinity+1i"},
		{New(IntFromInt64(1), Float(math.NaN())), "(1+NaN*i)", "1+NaN*i"},
		{New(Float(math.Inf(-1)), IntFromInt64(2)), "(-Infinity+2i)", "-Infinity+2i"},
		{New(IntFromInt64(1), Float(math.Inf(-1))), "(1-Infinity*i)", "1-Infinity*i"},
		{New(IntFromInt64(2), Float(math.Inf(1))), "(2+Infinity*i)", "2+Infinity*i"},
		// signed-zero float imaginary keeps its minus.
		{New(IntFromInt64(1), Float(math.Copysign(0, -1))), "(1-0.0i)", "1-0.0i"},
		{New(Float(math.Copysign(0, -1)), IntFromInt64(1)), "(-0.0+1i)", "-0.0+1i"},
	}
	for _, c := range cases {
		if got := c.c.Inspect(); got != c.inspect {
			t.Errorf("Inspect = %q, want %q", got, c.inspect)
		}
		if got := c.c.ToS(); got != c.toS {
			t.Errorf("ToS = %q, want %q", got, c.toS)
		}
	}
}

// TestFormatFloat pins Ruby Float#to_s formatting (the part renderer) across the
// fixed/scientific threshold and exponent-padding boundaries.
func TestFormatFloat(t *testing.T) {
	cases := map[float64]string{
		2.0:                  "2.0",
		2.5:                  "2.5",
		0.1:                  "0.1",
		100.0:                "100.0",
		0.0001:               "0.0001",
		1e-5:                 "1.0e-05",
		1e14:                 "100000000000000.0",
		1e15:                 "1.0e+15",
		1e16:                 "1.0e+16",
		1e20:                 "1.0e+20",
		1e-7:                 "1.0e-07",
		1e100:                "1.0e+100",
		1.5e-10:              "1.5e-10",
		3.141592653589793:    "3.141592653589793",
		123456789012345.0:    "123456789012345.0",
		1234567890123456.0:   "1.234567890123456e+15",
		0.0:                  "0.0",
		math.Copysign(0, -1): "-0.0",
		-2.5:                 "-2.5",
		math.Inf(1):          "Infinity",
		math.Inf(-1):         "-Infinity",
		math.NaN():           "NaN",
	}
	for f, want := range cases {
		if got := formatFloat(f); got != want {
			t.Errorf("formatFloat(%g) = %q, want %q", f, got, want)
		}
	}
}
