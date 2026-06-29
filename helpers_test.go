// Copyright (c) the go-ruby-complex/complex authors
//
// SPDX-License-Identifier: BSD-3-Clause

package complex

import (
	"math"
	"math/big"
	"strconv"
	"testing"
)

func bigStr(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("bad bigStr %q", s)
	}
	return v
}

func infP() float64 { return math.Inf(1) }
func infN() float64 { return math.Inf(-1) }
func nan() float64  { return math.NaN() }

// formatG17 mirrors Ruby's printf("%.17g") for comparing transcendental floats.
func formatG17(f float64) string { return strconv.FormatFloat(f, 'g', 17, 64) }
