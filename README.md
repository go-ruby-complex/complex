<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-complex/brand/main/social/go-ruby-complex-complex.png" alt="go-ruby-complex/complex" width="720"></p>

# complex — go-ruby-complex

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-complex.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's
[`Complex`](https://docs.ruby-lang.org/en/master/Complex.html)** — the `a+bi`
number, byte-exact with MRI 4.0.5's `inspect` / `to_s` and arithmetic, **without
any Ruby runtime**.

Unlike Go's built-in `complex128`, Ruby's `Complex` keeps its two parts on the
**numeric tower**: an Integer part stays an arbitrary-precision Integer, a Rational
part stays Rational, and only a Float part becomes Float. So
`Complex(1,2)*Complex(1,2)` is the *exact* `(-3+4i)`,
`Complex(Rational(1,2),Rational(3,4))` inspects as `((1/2)+(3/4)*i)`, and only
operations that intrinsically need a float (`abs`, `arg`, non-integer `**`) produce
floats. This package reproduces that behaviour and MRI's exact byte layout.

It is the `Complex` backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby)'s `rbgo`, but is a
**standalone, reusable** module with no dependency on the Ruby runtime — a sibling
of [go-ruby-rational](https://github.com/go-ruby-rational/rational) and
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp).

## Features

Faithful port of `Complex`, validated against the `ruby` binary on every supported
platform:

- **Exact-preserving arithmetic** — `Add`/`Sub`/`Mul`/`Div`/`Pow` keep Integer and
  Rational parts exact (Integer×Integer → Integer; anything Rational stays
  Rational, so `Rational(1,2)*2` is `(1/1)`; division canonicalises, so
  `Complex(2,-2)/2` is `(1-1i)` but `Complex(10,2)/4` is `((5/2)+(1/2)i)`).
- **Integer powers stay exact** — `Complex(2,3)**5 == (122-597i)`,
  `Complex(2,3)**-2 == ((-5/169)-(12/169)i)`; non-integer / Float exponents use the
  floating polar form.
- **Magnitude & phase** — `Abs` (`5.0` for `Complex(3,4)`, via `math.Hypot`),
  `Abs2` (exact `25`), `Arg`/`Angle`/`Phase` (`math.Atan2`), `Conjugate`,
  `Rectangular`, `PolarParts`.
- **Constructors** — `New`/`Rect`, `Polar`, and `Parse` for MRI's `Complex(string)`
  grammar (`"1+2i"`, `"-i"`, `"1/2+3/4i"`, `"2.5-1.5i"`, `"1.5e3+2i"`, `"1@2"`
  polar, `j` as the imaginary unit, `_` digit separators).
- **MRI-exact rendering** — `Inspect` (`(1+2i)`, `((1/2)+(3/4)*i)`, `(Infinity+1i)`)
  and `ToS` (`1+2i`, `1/2+3/4i`), including Ruby's `Float#to_s` formatting and the
  sign / `*i` rules.
- **Conversions** — `ToF`/`ToI`/`ToR`/`ToC`, `Numerator`/`Denominator` (common
  denominator), `Eql`(`eql?`)/`Equal`(`==`), `FiniteQ`/`InfiniteQ`.

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) and three OSes (Linux, macOS, Windows).

## Install

```sh
go get github.com/go-ruby-complex/complex
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-complex/complex"
)

func main() {
	a := complex.New(complex.IntFromInt64(1), complex.IntFromInt64(2))
	b := complex.New(complex.IntFromInt64(3), complex.IntFromInt64(4))

	fmt.Println(a.Inspect())            // (1+2i)
	fmt.Println(a.Add(b).Inspect())     // (4+6i)
	fmt.Println(a.Mul(a).Inspect())     // (-3+4i)   — exact
	fmt.Println(b.Abs())                // 5

	// Exact Rational parts.
	r := complex.New(complex.RatFromInt64(1, 2), complex.RatFromInt64(3, 4))
	fmt.Println(r.Inspect())            // ((1/2)+(3/4)*i)
	fmt.Println(r.ToS())                // 1/2+3/4i

	// Parse MRI's Complex(string) forms.
	c, _ := complex.Parse("1+2i")
	fmt.Println(c.Inspect())            // (1+2i)
}
```

## The numeric tower (`Num`)

Each part is a `Num` — a small union of Integer (`*big.Int`), Rational
(`*big.Rat`) and Float (`float64`) — so the package preserves exactness exactly as
MRI does. Build one with `IntFromInt64` / `Int`, `RatFromInt64` / `RatFrom` / `Rat`,
or `Float`; inspect it with `IsInt` / `IsRat` / `IsFloat` / `Float64`.

## API

```go
// Construction
func New(re, im Num) *Complex            // Complex.rect
func Rect(re, im Num) *Complex           // alias
func Polar(abs, arg Num) *Complex        // Complex.polar (Float parts)
func Parse(s string) (*Complex, error)   // Complex(string)

// Arithmetic (exact-preserving)
func (c *Complex) Add(o *Complex) *Complex
func (c *Complex) Sub(o *Complex) *Complex
func (c *Complex) Mul(o *Complex) *Complex
func (c *Complex) Div(o *Complex) *Complex
func (c *Complex) Pow(exp Num) *Complex
func (c *Complex) Neg() *Complex
func (c *Complex) Conjugate() *Complex   // conjugate / conj

// Magnitude & phase
func (c *Complex) Abs() float64          // abs / magnitude
func (c *Complex) Abs2() Num             // abs2 (exact)
func (c *Complex) Arg() float64          // arg / angle / phase
func (c *Complex) Rectangular() (Num, Num)
func (c *Complex) PolarParts() (float64, float64)

// Parts, equality, predicates
func (c *Complex) Real() Num
func (c *Complex) Imaginary() Num
func (c *Complex) Eql(o *Complex) bool   // eql?
func (c *Complex) Equal(o *Complex) bool // ==
func (c *Complex) FiniteQ() bool         // finite?
func (c *Complex) InfiniteQ() (int, bool)// infinite?

// Conversions & rendering
func (c *Complex) ToF() (float64, error) // to_f (RangeError if imag != 0)
func (c *Complex) ToI() (*big.Int, error)// to_i
func (c *Complex) ToR() (Num, error)     // to_r
func (c *Complex) ToC() *Complex         // to_c
func (c *Complex) Numerator() *Complex
func (c *Complex) Denominator() Num
func (c *Complex) Inspect() string       // "(1+2i)"
func (c *Complex) ToS() string           // "1+2i"
```

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
100%, so the qemu cross-arch and Windows lanes pass the gate) with a **differential
MRI oracle**: a wide corpus of constructors, arithmetic, conversions and string
parses is built here and compared byte-for-byte against the system `ruby`'s
`inspect` / `to_s`. The oracle gates itself on `RUBY_VERSION >= "4.0"`,
`$stdout.binmode`s so Windows text-mode never pollutes the bytes, and skips itself
where a 4.x `ruby` is absent.

> **Note.** Exact arithmetic is byte-identical with MRI. The handful of results
> that are intrinsically floating (`Polar`, `Parse("a@b")`, non-integer `Pow`) can
> differ from MRI by a unit in the last place, because Go's pure-Go `math.Sin` /
> `Cos` / `Pow` are not the platform `libm` MRI links; `Abs` (`Hypot`) and `Arg`
> (`Atan2`) do match. These transcendental cases are verified under a tolerance.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-complex/complex authors.
