// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import "math/big"

// Extended is an x87 Extended value: the FPU computes with a 64-bit
// mantissa and rounds to nearest even after every operation (Delphi's
// default control word). Where the original truncates an Extended result,
// the extra rounding can land below an integer that float64 would hit.
type Extended struct {
	val big.Float
}

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
