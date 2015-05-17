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

// #cgo CFLAGS: -I/usr/include/lpsolve/
// #cgo LDFLAGS: -llpsolve55 -lm -ldl -lcolamd
// #include <lp_lib.h>
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
	prob *C.lprec
	vars []*Variable
}

type direction C.uchar

const (
	Minimize = direction(C.FALSE)
	Maximize = direction(C.TRUE)
)

/* Model related functions */

func NewModel(name string, dir direction) *Model {
	prob := C.make_lp(0, 0)
	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))
	C.set_lp_name(prob, c_name)
	C.set_sense(prob, C.uchar(dir))

	model := &Model{prob: prob}

	C.set_verbose(prob, C.FALSE) // FIXME: use put_logfunc to *really* silence the lib

	// plug the underlying C library's destructors to the instance of Model,
	// otherwise we get a memory-leak of the underlying struct
	runtime.SetFinalizer(model, finalizeModel)

	return model
}

func finalizeModel(model *Model) {
	C.delete_lp(model.prob)
}

func NewMaximizeModel(name string) *Model {
	return NewModel(name, Maximize)
}

func NewMinimizeModel(name string) *Model {
	return NewModel(name, Minimize)
}

func (model *Model) GetName() string {
	return C.GoString(C.get_lp_name(model.prob))
}

func (model Model) SetDirection(dir direction) {
	C.set_sense(model.prob, C.uchar(dir))
}

func (model *Model) GetDirection() direction {
	if C.is_maxim(model.prob) == C.TRUE {
		return Maximize
	} else {
		return Minimize
	}
}

/* Column-related functions */

func (model *Model) GetVariableCount() int {
	return int(C.get_Ncolumns(model.prob))
}

func (model *Model) GetVariables() []*Variable {
	return model.vars
}

// AddVariable adds a variable to the linear programming model and
// returns a reference to it.
// A freshly instantiated variable has the default type of
// ContinuousVariable, no bounds and an objective coefficient of 1.
//
// A variable is bound to its model. Attempting to use a variable
// created in one model for fetching solutions from a different model
// results in undefined behaviour.
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
// objective coefficient of 1.
func (model *Model) AddIntegerVariable(name string) (v *Variable, err error) {
	return model.AddDefinedVariable(name, IntegerVariable, 1, math.Inf(-1), math.Inf(1))
}

// AddDefinedVariable add a variable to the linear programming model
// with its attributes passed as arguments.
// If varType is BinaryVariable, the bounds are ignored.
func (model *Model) AddDefinedVariable(name string, varType variableType, coefficient, lowerBound, upperBound float64) (v *Variable, err error) {
	size := model.GetVariableCount()
	v = new(Variable)
	v.index = size
	v.model = model
	model.vars = append(model.vars, v)

	// when adding a variable after some constraints have been defined,
	// we pass an array filled with zeroes to add_column, so the new
	// variable is assumed to not be used in the existing constraints
	C.add_columnex(model.prob, 0, nil, nil)
	//coef_array := make([]C.REAL, model.GetConstraintCount()+1)
	//C.add_column(model.prob, &coef_array[0])

	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))
	C.set_col_name(model.prob, C.int(v.index+1), c_name)
	v.SetType(varType)
	v.SetObjectiveCoefficient(coefficient)
	if varType != BinaryVariable {
		v.SetBounds(lowerBound, upperBound)
	}

	return
}

/* Constraint-related functions */

func (model *Model) GetConstraintCount() int {
	return int(C.get_Nrows(model.prob))
}

func (model *Model) AddConstraint(lower, upper float64, vars []*Variable, coefs []float64) error {
	if len(vars) != len(coefs) {
		return fmt.Errorf("inconsistent number of variables and coefficients: %d != %d", len(vars), len(coefs))
	}

	row := make([]C.REAL, len(vars))
	colno := make([]C.int, len(vars))
	for i, v := range vars {
		colno[i] = C.int(v.index + 1)
		row[i] = C.REAL(coefs[i])
	}

	switch {
	case math.IsInf(lower, 0) && math.IsInf(upper, 0):
		// no constraints
	case math.IsInf(lower, 0):
		C.add_constraintex(model.prob, C.int(len(vars)), &row[0], &colno[0], C.LE, C.double(upper))
	case math.IsInf(upper, 0):
		C.add_constraintex(model.prob, C.int(len(vars)), &row[0], &colno[0], C.GE, C.double(lower))
	case upper == lower:
		C.add_constraintex(model.prob, C.int(len(vars)), &row[0], &colno[0], C.EQ, C.double(upper))
	default:
		C.add_constraintex(model.prob, C.int(len(vars)), &row[0], &colno[0], C.LE, C.double(upper))
		C.add_constraintex(model.prob, C.int(len(vars)), &row[0], &colno[0], C.GE, C.double(lower))
	}

	return nil
}

type solveResult struct {
	model  *Model
	status solveStatus
}

type solveStatus C.int

const (
	SolutionOptimal    = solveStatus(C.OPTIMAL)
	SolutionSuboptimal = solveStatus(C.SUBOPTIMAL)
)

type solveError C.int

const (
	ErrorModelInfeasible  = solveError(C.INFEASIBLE)
	ErrorModelUnbounded   = solveError(C.UNBOUNDED)
	ErrorModelDegenerate  = solveError(C.DEGENERATE)
	ErrorNumericalFailure = solveError(C.NUMFAILURE)
	ErrorUserAbort        = solveError(C.USERABORT) // we don't use C.put_abortfunc
	ErrorTimeout          = solveError(C.TIMEOUT)   // FIXME: support C.set_timeout
	//ErrorPresolved        = solveError(C.PRESOLVED) // we can't use C.set_presolve because it might remove Variables
	ErrorBranchCutFail   = solveError(C.PROCFAIL)
	ErrorBranchCutBreak  = solveError(C.PROCBREAK) // we don't use set_break_at_first/set_break_at_value
	ErrorFeasibleFound   = solveError(C.FEASFOUND)
	ErrorNoFeasibleFound = solveError(C.NOFEASFOUND)
	ErrorNoMemory        = solveError(C.NOMEMORY)
)

func (e solveError) Error() string {
	switch e {
	case ErrorModelInfeasible:
		return "model is infeasible"
	case ErrorModelUnbounded:
		return "model is unbounded"
	case ErrorModelDegenerate:
		return "model is degenerate"
	case ErrorNumericalFailure:
		return "numerical failure while solving"
	case ErrorUserAbort:
		return "aborted by user abort function "
	case ErrorTimeout:
		return "timeout occurred before any integer solution could be found"
	//case ErrorPresolved:
	case ErrorBranchCutFail:
		return "branch-and-cut failure"
	case ErrorBranchCutBreak:
		return "branch-and-cut stopped at beakpoint"
	case ErrorFeasibleFound:
		return "feasible but non-integer solution found"
	case ErrorNoFeasibleFound:
		return "no feasible solution found"
	case ErrorNoMemory:
		return "ran out of memory while solving"
	default:
		panic("unrecognized error")
	}
}

// Solve attempts to find an optimal solution to the model.
func (model *Model) Solve() (res *solveResult, err error) {
	res = new(solveResult)
	res.model = model

	ret := C.solve(model.prob)

	switch ret {
	case C.OPTIMAL, C.SUBOPTIMAL:
		res.status = solveStatus(ret)
		return res, nil
	case C.INFEASIBLE, C.UNBOUNDED, C.DEGENERATE, C.NUMFAILURE,
		C.USERABORT, C.TIMEOUT, C.PROCFAIL, C.PROCBREAK, C.FEASFOUND,
		C.NOFEASFOUND, C.NOMEMORY:
		return nil, solveError(ret)
	default:
		panic("unrecognized result")
	}
}

/* Result-related functions */

// GetStatus reports if the solution is optimal (SolutionOptimal) or
// not (SolutionSuboptimal)
func (res solveResult) GetStatus() solveStatus {
	return res.status
}

func (res solveResult) GetValue(v *Variable) float64 {
	return res.GetPrimalValue(v)
}

func (res solveResult) GetPrimalValue(v *Variable) float64 {
	// get_var_*result uses funny indexing: 0=objective,1 to Nrows=constraint,Nrows to Nrows+Ncols=variable
	return float64(C.get_var_primalresult(res.model.prob, C.int(v.index+v.model.GetConstraintCount()+1)))
}

func (res solveResult) GetDualValue(v *Variable) float64 {
	// get_var_*result uses funny indexing: 0=objective,1 to Nrows=constraint,Nrows to Nrows+Ncols=variable
	return float64(C.get_var_dualresult(res.model.prob, C.int(v.index+v.model.GetConstraintCount()+1)))
}

func (res solveResult) GetObjectiveValue() float64 {
	return float64(C.get_objective(res.model.prob))
}
