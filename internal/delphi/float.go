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
	exponent, _ := strconv.Atoi(exp)
	digits := strings.TrimRight(strings.Replace(mant, ".", "", 1), "0")
	// Fixed notation for values of at least 0.00001 with at most 15 digits
	// before the point; scientific otherwise ("1.5E-7", "1E20").
	if x >= 0.00001 && exponent < 15 {
		var txt string

		switch {
		case exponent < 0:
			txt = "0." + strings.Repeat("0", -exponent-1) + digits
		case len(digits) <= exponent+1:
			txt = digits + strings.Repeat("0", exponent+1-len(digits))
		default:
			txt = digits[:exponent+1] + "." + digits[exponent+1:]
		}

		return sign + txt
	}

	txt := digits[:1]
	if len(digits) > 1 {
		txt += "." + digits[1:]
	}

	return sign + txt + "E" + strconv.Itoa(exponent)
}

// Extended is an x87 Extended value: the FPU computes with a 64-bit
// mantissa and rounds to nearest even after every operation (Delphi's
// default control word). Where the original truncates an Extended result,
// the extra rounding can land below an integer that float64 would hit.
type Extended struct{ val big.Float }

// NewExtended returns x as an Extended (exact: a float64 always fits).
func NewExtended(x float64) *Extended {
	e := &Extended{}
	e.val.SetPrec(64).SetMode(big.ToNearestEven).SetFloat64(x)

	return e
}

// Quo returns e / y, rounded to 64 bits.
func (e *Extended) Quo(y *Extended) *Extended {
	r := NewExtended(0)
	r.val.Quo(&e.val, &y.val)

	return r
}

// Mul returns e × y, rounded to 64 bits.
func (e *Extended) Mul(y *Extended) *Extended {
	r := NewExtended(0)
	r.val.Mul(&e.val, &y.val)

	return r
}

// Int truncates toward zero, as Int does.
func (e *Extended) Int() *Extended {
	i, _ := e.val.Int(nil)
	r := NewExtended(0)
	r.val.SetInt(i)

	return r
}

// Float64 rounds e to a Double, as storing it in a Double variable does.
func (e *Extended) Float64() float64 {
	x, _ := e.val.Float64()

	return x
}
