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

type VariableType C.int

const (
	ContinuousVariable = VariableType(C.GLP_CV)
	IntegerVariable    = VariableType(C.GLP_IV)
	BinaryVariable     = VariableType(C.GLP_BV)
)

type BoundType C.int

const (
	NoBound     = BoundType(C.GLP_FR)
	UpperBound  = BoundType(C.GLP_UP)
	LowerBound  = BoundType(C.GLP_LO)
	DoubleBound = BoundType(C.GLP_DB)
	FixedBound  = BoundType(C.GLP_FX)
)

type Variable struct {
	model *Model
	index int
}

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

/* Solver-related functions */

func (model *Model) loadMatrix() {
	C.glp_load_matrix(model.prob, C.int(len(model.ia)-1), &model.ia[0], &model.ja[0], &model.ar[0])
}

// SolveSimplex solves the linear programming model using the
// general simplex algorithm.
func (model *Model) SolveSimplex() error {
	model.loadMatrix()
	var parm C.glp_smcp
	C.glp_init_smcp(&parm)

	if model.Verbose {
		parm.msg_lev = C.GLP_MSG_ON
	} else {
		parm.msg_lev = C.GLP_MSG_OFF
	}

	if model.Presolve {
		parm.presolve = C.GLP_ON
	} else {
		parm.presolve = C.GLP_OFF
	}

	err := C.glp_simplex(model.prob, &parm)
	switch err {
	case 0:
		return nil
	default:
		return fmt.Errorf("glpk error: %d", err)
		//FIXME: add actual error messages from doc
	}
}

// SolveBranchCut solves the linear programming model using the
// branch-and-cut algorithm. Better suited for mixed-integer linear
// programs, i.e.: problems with integer and/or binary variables.
func (model *Model) SolveBranchCut() error {
	model.loadMatrix()
	var parm C.glp_iocp
	C.glp_init_iocp(&parm)

	if model.Verbose {
		parm.msg_lev = C.GLP_MSG_ON
	} else {
		parm.msg_lev = C.GLP_MSG_OFF
	}

	if model.Presolve {
		parm.presolve = C.GLP_ON
	} else {
		parm.presolve = C.GLP_OFF
	}

	err := C.glp_intopt(model.prob, &parm)
	switch err {
	case 0:
		return nil
	default:
		return fmt.Errorf("glpk error: %d", err)
		//FIXME: add actual error messages from doc
	}
}

/* Result-related functions */

func (v *Variable) GetValue() float64 {
	return float64(C.glp_mip_col_val(v.model.prob, C.int(v.index+1)))
}

func (model *Model) GetObjectiveValue() float64 {
	return float64(C.glp_mip_obj_val(model.prob))
}
