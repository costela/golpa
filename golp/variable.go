/*
Copyright © 2015 Leo Antunes <leo@costela.net>

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

package golp

// #cgo LDFLAGS: -lglpk
// #include <glpk.h>
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"math"
)

type Variable struct {
	model *Model
	index int
}

type VariableType C.int

const (
	ContinuousVariable = VariableType(C.GLP_CV)
	IntegerVariable    = VariableType(C.GLP_IV)
	BinaryVariable     = VariableType(C.GLP_BV)
)

/* Variable-related functions (model variables, as opposed to Go variables) */

func (v *Variable) GetName() string {
	return C.GoString(C.glp_get_col_name(v.model.prob, C.int(v.index+1)))
}

func (v *Variable) SetType(vartype VariableType) {
	C.glp_set_col_kind(v.model.prob, C.int(v.index+1), C.int(vartype))
}

func (v *Variable) GetType() VariableType {
	return VariableType(C.glp_get_col_kind(v.model.prob, C.int(v.index+1)))
}

// SetBounds sets the boundaries for the given variable.
// To set a bound to infinity, pass math.Inf(1) or math.Inf(-1). The
// signal of the infinity is ignored, as the lower and upper bounds are
// always assumed to be the negative and positive infinities,
// respectively.
func (v *Variable) SetBounds(lower, upper float64) {
	switch {
	case math.IsInf(lower, 0) && math.IsInf(upper, 0):
		C.glp_set_col_bnds(v.model.prob, C.int(v.index+1), C.GLP_FR, C.double(0), C.double(0))
	case math.IsInf(lower, 0):
		C.glp_set_col_bnds(v.model.prob, C.int(v.index+1), C.GLP_UP, C.double(0), C.double(upper))
	case math.IsInf(upper, 0):
		C.glp_set_col_bnds(v.model.prob, C.int(v.index+1), C.GLP_LO, C.double(lower), C.double(0))
	case upper == lower:
		C.glp_set_col_bnds(v.model.prob, C.int(v.index+1), C.GLP_FX, C.double(lower), C.double(upper))
	default:
		C.glp_set_col_bnds(v.model.prob, C.int(v.index+1), C.GLP_DB, C.double(lower), C.double(upper))
	}
}

func (v *Variable) GetBounds() (lower, upper float64) {
	bound_type := C.glp_get_col_type(v.model.prob, C.int(v.index+1))

	lower = math.Inf(-1)
	upper = math.Inf(1)

	switch bound_type {
	case C.GLP_FR:
		return
	case C.GLP_UP:
		upper = float64(C.glp_get_col_ub(v.model.prob, C.int(v.index+1)))
		return
	case C.GLP_LO:
		lower = float64(C.glp_get_col_lb(v.model.prob, C.int(v.index+1)))
		return
	case C.GLP_FX:
		// according to the glpk docs, only lb is used for fixed bounds
		lower = float64(C.glp_get_col_lb(v.model.prob, C.int(v.index+1)))
		upper = float64(C.glp_get_col_lb(v.model.prob, C.int(v.index+1)))
		return
	case C.GLP_DB:
		lower = float64(C.glp_get_col_lb(v.model.prob, C.int(v.index+1)))
		upper = float64(C.glp_get_col_ub(v.model.prob, C.int(v.index+1)))
		return
	default:
		panic(fmt.Sprintf("unsupported bound type %v", bound_type))
	}
}

func (v *Variable) SetCoefficient(coef float64) {
	C.glp_set_obj_coef(v.model.prob, C.int(v.index+1), C.double(coef))
}

func (v *Variable) GetCoefficient() float64 {
	return float64(C.glp_get_obj_coef(v.model.prob, C.int(v.index+1)))
}
