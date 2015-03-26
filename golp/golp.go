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
	"runtime"
	"unsafe"
)

/* Types */

type Model struct {
	prob     *C.glp_prob
	vars     []*Variable
	ia       []C.int
	ja       []C.int
	ar       []C.double
	Verbose  bool
	Presolve bool
}

type Direction C.int

const (
	Minimize = Direction(C.GLP_MIN)
	Maximize = Direction(C.GLP_MAX)
)

type BoundType C.int

const (
	NoBound     = BoundType(C.GLP_FR)
	UpperBound  = BoundType(C.GLP_UP)
	LowerBound  = BoundType(C.GLP_LO)
	DoubleBound = BoundType(C.GLP_DB)
	FixedBound  = BoundType(C.GLP_FX)
)

/* Model related functions */

func NewModel(name string, dir Direction) Model {
	prob := C.glp_create_prob()
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))
	C.glp_set_prob_name(prob, c_name)
	C.glp_set_obj_dir(prob, C.int(dir))

	model := Model{prob: prob}
	// glpk indices start at 1; index 0 is reserved
	model.ia = append(model.ia, 0)
	model.ja = append(model.ja, 0)
	model.ar = append(model.ar, 0.0)

	model.Verbose = false
	model.Presolve = true

	// plug the underlying C library's destructors to the instance of Model,
	// otherwise we get a memory-leak of the underlying struct
	runtime.SetFinalizer(&model, finalizeModel)

	return model
}

func finalizeModel(model *Model) {
	C.glp_delete_prob(model.prob)
}

func NewMaximizeModel(name string) Model {
	return NewModel(name, Maximize)
}

func NewMinimizeModel(name string) Model {
	return NewModel(name, Minimize)
}

func (model *Model) GetName() string {
	return C.GoString(C.glp_get_prob_name(model.prob))
}

func (model Model) SetDirection(dir Direction) {
	C.glp_set_obj_dir(model.prob, C.int(dir))
}

func (model *Model) GetDirection() Direction {
	return Direction(C.glp_get_obj_dir(model.prob))
}

/* Column-related functions */

func (model *Model) GetVariableCount() int {
	return int(C.glp_get_num_cols(model.prob))
}

func (model *Model) setColumnCount(n int) (err error) {
	current_columns := model.GetVariableCount()
	if current_columns < n {
		if ret := C.glp_add_cols(model.prob, C.int(n-current_columns)); ret < 1 {
			return fmt.Errorf("could not scale columns for model")
		}
	} else if current_columns > n {
		// TODO: we could probably handle this more elegantly
		return fmt.Errorf("reducing column-count not supported")
	} // current_columns == n → noop
	return
}

func (model *Model) GetVariables() []*Variable {
	return model.vars
}

// AddVariable adds a variable to the linear programming model.
// A freshly instantiated variable has the default type of
// ContinuousVariable, no bounds and a coefficient of 1.
func (model *Model) AddVariable(name string) (v *Variable, err error) {
	return model.AddDefinedVariable(name, ContinuousVariable, 1, math.Inf(-1), math.Inf(1))
}

// AddBinaryVariable is a convenience function for adding a single
// named binary variable to the model, with a default coefficient of 1.
func (model *Model) AddBinaryVariable(name string) (v *Variable, err error) {
	return model.AddDefinedVariable(name, BinaryVariable, 1, 0, 1)
}

// AddIntegerVariable is a convenience function for adding a single
// named unbounded integer variable to the model, with a default
// coefficient of 1.
func (model *Model) AddIntegerVariable(name string) (v *Variable, err error) {
	return model.AddDefinedVariable(name, IntegerVariable, 1, math.Inf(-1), math.Inf(1))
}

// AddDefinedVariable add a variable to the linear programming model
// with its attributes passed as arguments.
// If varType is BinaryVariable, the bounds are ignored.
func (model *Model) AddDefinedVariable(name string, varType VariableType, coefficient, lowerBound, upperBound float64) (v *Variable, err error) {
	size := model.GetVariableCount()
	if err = model.setColumnCount(size + 1); err != nil {
		return
	}
	v = new(Variable)
	v.index = size
	v.model = model
	model.vars = append(model.vars, v)

	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))
	C.glp_set_col_name(model.prob, C.int(v.index+1), c_name)
	v.SetType(varType)
	v.SetCoefficient(coefficient)
	if varType != BinaryVariable {
		v.SetBounds(lowerBound, upperBound)
	}

	return
}

/* Constraint-related functions */

func (model *Model) GetConstraintCount() int {
	return int(C.glp_get_num_rows(model.prob))
}

func (model *Model) setRowCount(n int) (err error) {
	current_rows := model.GetConstraintCount()
	if current_rows < n {
		if ret := C.glp_add_rows(model.prob, C.int(n-current_rows)); ret < 1 {
			return fmt.Errorf("could not scale rows for model")
		}
	} else if current_rows > n {
		// TODO: we could probably handle this more elegantly
		return fmt.Errorf("reducing row-count not supported")
	} // current_rows == n → noop
	return
}

func (model *Model) AddConstraint(lower, upper float64, vars []*Variable, coefs []float64) error {
	if len(vars) != len(coefs) {
		return fmt.Errorf("inconsistent number of variables and coefficients: %d != %d", len(vars), len(coefs))
	}

	size := model.GetConstraintCount()
	if err := model.setRowCount(size + 1); err != nil {
		return err
	}
	switch {
	case math.IsInf(lower, 0) && math.IsInf(upper, 0):
		C.glp_set_row_bnds(model.prob, C.int(size+1), C.GLP_FR, C.double(0), C.double(0))
	case math.IsInf(lower, 0):
		C.glp_set_row_bnds(model.prob, C.int(size+1), C.GLP_UP, C.double(0), C.double(upper))
	case math.IsInf(upper, 0):
		C.glp_set_row_bnds(model.prob, C.int(size+1), C.GLP_LO, C.double(lower), C.double(0))
	case upper == lower:
		C.glp_set_row_bnds(model.prob, C.int(size+1), C.GLP_FX, C.double(lower), C.double(upper))
	default:
		C.glp_set_row_bnds(model.prob, C.int(size+1), C.GLP_DB, C.double(lower), C.double(upper))
	}

	for i, v := range vars {
		model.ia = append(model.ia, C.int(size+1))
		model.ja = append(model.ja, C.int(v.index+1))
		model.ar = append(model.ar, C.double(coefs[i]))
	}
	return nil
}

func (model *Model) loadMatrix() {
	C.glp_load_matrix(model.prob, C.int(len(model.ia)-1), &model.ia[0], &model.ja[0], &model.ar[0])
}

func glpkError(err C.int) error {
	switch err {
	case 0:
		return nil
	case C.GLP_EBADB:
		return fmt.Errorf("initial basis invalid")
	case C.GLP_ESING:
		return fmt.Errorf("initial basis is exactly singular")
	case C.GLP_EBOUND:
		return fmt.Errorf("double-bounded (auxiliary or structural) variables has incorrect bounds")
	case C.GLP_EFAIL:
		return fmt.Errorf("problem instance has no rows/columns")
	case C.GLP_EITLIM:
		return fmt.Errorf("simplex iteration limit exceeded")
	case C.GLP_ETMLIM:
		return fmt.Errorf("time limit exceeded")
	case C.GLP_EROOT:
		return fmt.Errorf("optimal basis for initial LP relaxation not provided and presolver not used")
	case C.GLP_ENOPFS:
		return fmt.Errorf("LP relaxation of MIP problem has no primal feasible solution")
	case C.GLP_ENODFS:
		return fmt.Errorf("LP relaxation of MIP problem has no dual feasible solution")
	case C.GLP_EMIPGAP:
		return fmt.Errorf("MIP gap tolerance exceeded")
	default:
		return fmt.Errorf("unknown glpk error: %d", err)
	}
}
