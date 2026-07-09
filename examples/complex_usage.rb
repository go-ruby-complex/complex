# frozen_string_literal: true

# Complex is a core feature of the numeric tower — no `require` is needed.
# It is the `a+bi` number that keeps its parts exact on the numeric tower.

z = Complex(1, 2)                     # build a Complex from real + imaginary
puts z.inspect                        # => (1+2i)
puts z.to_s                           # => 1+2i

# Exact-preserving arithmetic (Integer parts stay Integers).
puts (z + Complex(3, 4)).inspect      # => (4+6i)
puts (z * Complex(1, 2)).inspect      # => (-3+4i)
puts (Complex(2, -2) / 2).inspect     # => (1.0-1.0i)

# Parts, conjugate and magnitude.
puts z.real                           # => 1
puts z.imaginary                      # => 2
puts z.conjugate.inspect              # => (1-2i)
puts z.abs                            # => 2.23606797749979

# Polar / rectangular views and the phase angle.
puts z.arg                            # => 1.1071487177940904
puts z.polar.inspect                  # => [2.23606797749979, 1.1071487177940904]
puts z.rectangular.inspect            # => [1, 2]

# Rational parts stay Rational, so nothing is rounded away.
puts Complex(Rational(1, 2), Rational(3, 4)).inspect # => ((1/2)+(3/4)*i)
