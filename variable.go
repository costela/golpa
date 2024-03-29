/*
Copyright © 2015-2022 Leo Antunes <leo@costela.net>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package golpa

// #cgo CFLAGS: -I/usr/include/lpsolve/
// #cgo LDFLAGS: -llpsolve55 -lm -ldl -lcolamd
// #include <lp_lib.h>
// #include <stdlib.h>
import "C"

import (
	"math"
)

type Variable struct {
	model *Model
	index int
}

type VariableType int

const (
	ContinuousVariable VariableType = iota
	IntegerVariable
	BinaryVariable
)

/* variable-related functions (model variables, as opposed to Go variables) */

// Name returns the name of a variable
func (v *Variable) Name() string {
	v.model.mu.RLock()
	defer v.model.mu.RUnlock()

	return C.GoString(C.get_col_name(v.model.prob, C.int(v.index+1)))
}

// SetType sets the type of a variable to either:
//    - ContinuousVariable
//    - Integervariable
//    - BinaryVariable
func (v *Variable) SetType(vartype VariableType) {
	v.model.mu.Lock()
	defer v.model.mu.Unlock()

	switch vartype {
	case ContinuousVariable:
		C.set_int(v.model.prob, C.int(v.index+1), C.FALSE)
	case IntegerVariable:
		C.set_int(v.model.prob, C.int(v.index+1), C.TRUE)
	case BinaryVariable:
		C.set_binary(v.model.prob, C.int(v.index+1), C.TRUE)
	default:
		panic("unrecognized variable type!")
	}
}

// Type returns this variable's type
func (v *Variable) Type() VariableType {
	v.model.mu.RLock()
	defer v.model.mu.RUnlock()

	if C.is_binary(v.model.prob, C.int(v.index+1)) == C.TRUE {
		return BinaryVariable
	} else if C.is_int(v.model.prob, C.int(v.index+1)) == C.TRUE {
		return IntegerVariable
	} else {
		return ContinuousVariable
	}
}

// SetBounds sets the boundaries for the given variable.
// To set a bound to infinity, pass math.Inf(1) or math.Inf(-1). The
// signal of the infinity is ignored, as the lower and upper bounds are
// always assumed to be the negative and positive infinities,
// respectively.
func (v *Variable) SetBounds(lower, upper float64) {
	v.model.mu.Lock()
	defer v.model.mu.Unlock()

	switch {
	case math.IsInf(lower, 0) && math.IsInf(upper, 0):
		C.set_unbounded(v.model.prob, C.int(v.index+1))
	case math.IsInf(lower, 0):
		C.set_unbounded(v.model.prob, C.int(v.index+1))
		C.set_upbo(v.model.prob, C.int(v.index+1), C.double(upper))
	case math.IsInf(upper, 0):
		C.set_unbounded(v.model.prob, C.int(v.index+1))
		C.set_lowbo(v.model.prob, C.int(v.index+1), C.double(lower))
	default:
		C.set_bounds(v.model.prob, C.int(v.index+1), C.double(lower), C.double(upper))
	}
}

// Bounds returns the bounds currently set for this variable.
func (v *Variable) Bounds() (lower, upper float64) {
	v.model.mu.RLock()
	defer v.model.mu.RUnlock()

	lower = float64(C.get_lowbo(v.model.prob, C.int(v.index+1)))
	upper = float64(C.get_upbo(v.model.prob, C.int(v.index+1)))

	inf := float64(C.get_infinite(v.model.prob))

	if lower == -inf {
		lower = math.Inf(-1)
	}
	if upper == inf {
		upper = math.Inf(1)
	}
	return
}

// SetObjectiveCoefficient sets the coefficient for this variable in
// the objective function.
func (v *Variable) SetObjectiveCoefficient(coef float64) {
	v.model.mu.Lock()
	defer v.model.mu.Unlock()

	C.set_mat(v.model.prob, C.int(0), C.int(v.index+1), C.REAL(coef))
}

// Coefficient returns this variable's coefficient in the objective
// function.
func (v *Variable) Coefficient() float64 {
	v.model.mu.RLock()
	defer v.model.mu.RUnlock()

	return float64(C.get_mat(v.model.prob, C.int(0), C.int(v.index+1)))
}
