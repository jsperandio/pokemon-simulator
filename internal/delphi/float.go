// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// FloatToStr formats x as Delphi's FloatToStr does (general format, 15
// significant digits), except that the decimal separator is always "."
// where the original followed the Windows locale.
func FloatToStr(x float64) string {
	switch {
	case math.IsNaN(x):
		return "NAN"
	case math.IsInf(x, 1):
		return "INF"
	case math.IsInf(x, -1):
		return "-INF"
	case x == 0:
		return "0"
	}

	sign := ""
	if x < 0 {
		sign, x = "-", -x
	}

	// 15 significant digits: d.dddddddddddddde±x
	mant, exp, _ := strings.Cut(strconv.FormatFloat(x, 'e', 14, 64), "e")
	e, _ := strconv.Atoi(exp)
	digits := strings.TrimRight(strings.Replace(mant, ".", "", 1), "0")
	// Fixed notation for values of at least 0.00001 with at most 15 digits
	// before the point; scientific otherwise ("1.5E-7", "1E20").
	if x >= 0.00001 && e < 15 {
		var s string

		switch {
		case e < 0:
			s = "0." + strings.Repeat("0", -e-1) + digits
		case len(digits) <= e+1:
			s = digits + strings.Repeat("0", e+1-len(digits))
		default:
			s = digits[:e+1] + "." + digits[e+1:]
		}

		return sign + s
	}

	s := digits[:1]
	if len(digits) > 1 {
		s += "." + digits[1:]
	}

	return sign + s + "E" + strconv.Itoa(e)
}

// Extended is an x87 Extended value: the FPU computes with a 64-bit
// mantissa and rounds to nearest even after every operation (Delphi's
// default control word). Where the original truncates an Extended result,
// the extra rounding can land below an integer that float64 would hit.
type Extended struct{ f big.Float }

// NewExtended returns x as an Extended (exact: a float64 always fits).
func NewExtended(x float64) *Extended {
	e := &Extended{}
	e.f.SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(x)

	return e
}

// Quo returns e / y, rounded to 64 bits.
func (e *Extended) Quo(y *Extended) *Extended {
	r := NewExtended(0)
	r.f.Quo(&e.f, &y.f)

	return r
}

// Mul returns e × y, rounded to 64 bits.
func (e *Extended) Mul(y *Extended) *Extended {
	r := NewExtended(0)
	r.f.Mul(&e.f, &y.f)

	return r
}

// Int truncates toward zero, as Int does.
func (e *Extended) Int() *Extended {
	i, _ := e.f.Int(nil)
	r := NewExtended(0)
	r.f.SetInt(i)

	return r
}

// Float64 rounds e to a Double, as storing it in a Double variable does.
func (e *Extended) Float64() float64 {
	x, _ := e.f.Float64()

	return x
}
