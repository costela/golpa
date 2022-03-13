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

/*

GoLPA is a library for moddeling and solving linear programming problems.

As an example of the API, the model of the following problem:

    Maximize:
      z = x1 + 2 x2 - 3 x3
    With:
      0 <= x1 <= 40
      5 <= x3 <= 11
    Subject to:
      0 <= - x1 + x2 + 5.3 x3 <= 10
      -inf <= 2 x1 - 5 x2 + 3 x3 <= 20
      x2 - 8 x3 = 0

can be expressed with GoLPA like this:

	package main

	import (
		"fmt"
		"math"

		"github.com/costela/golpa"
	)

	func main() {
		model, _ := golpa.NewModel("some model", golpa.Maximize)
		x1, _ := model.AddVariable("x1")
		x1.SetBounds(0, 40)
		x2, _ := model.AddVariable("x2")
		x2.SetObjectiveCoefficient(2)
		// alternatively, all information pertaining can be given at once:
		x3, _ := model.AddDefinedVariable("x3", golpa.ContinuousVariable, 3, 5, 11)

		model.AddConstraint(0, 10, []*golpa.Variable{x1, x2, x3}, []float64{-1, 1, 5.3})
		model.AddConstraint(math.Inf(-1), 20, []*golpa.Variable{x1, x2, x3}, []float64{2, -5, 3})
		model.AddConstraint(0, 0, []*golpa.Variable{x1, x3}, []float64{1, -8})

		// ⋮
		// The model can than be solved and the resulting values can than be retrieved as follows:
		// ⋮

		result, _ := model.Solve() // you should check for errors

		fmt.Printf("solution optimal? %t\n", result.Status() == golpa.SolutionOptimal)
		fmt.Printf("z = %f\n", result.ObjectiveValue())
		fmt.Printf("x1 = %f\n", result.Value(x1))
		// ⋮
	}

*/
package golpa

// #cgo linux LDFLAGS: -llpsolve55
// #cgo darwin LDFLAGS: -L/usr/local/lib -llpsolve55
// #cgo darwin CFLAGS: -I/usr/local/include
// #include <lp_lib.h>
// #include <stdlib.h>
/*
// https://golang.org/issue/19837
extern int abortCallback(lprec *lp, void *userhandle);
extern void logCallback(lprec *lp, void *userhandle, char *buf);
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"unsafe"
)

/* Types */

type Model struct {
	mu     sync.RWMutex
	prob   *C.lprec
	vars   []*Variable
	logger Logger
}

type direction C.uchar

const (
	Minimize = direction(C.FALSE)
	Maximize = direction(C.TRUE)
)

/* Model related functions */

// NewModel instantiates a new linear programming model, providing a
// name (purely informational) and a optimization direction (either
// Minimize or Maximize)
func NewModel(name string, dir direction, opts ...Option) (*Model, error) {
	prob := C.make_lp(0, 0)

	c_name := C.CString(name)
	defer C.free(unsafe.Pointer(c_name))

	C.set_lp_name(prob, c_name)
	C.set_sense(prob, C.uchar(dir))

	model := &Model{
		prob:   prob,
		logger: noopLogger{},
	}

	for _, opt := range opts {
		if err := opt(model); err != nil {
			return nil, fmt.Errorf("applying model option: %w", err)
		}
	}

	model.finishInitialization()

	return model, nil
}

// finishInitialization performs steps that are common to NewModel() and Clone().
func (model *Model) finishInitialization() {
	// disable stdoud logging and redirect to out internal logger
	C.put_logfunc(model.prob, (*C.lphandlestr_func)(C.logCallback), saveRef(model))
	C.set_outputfile(model.prob, C.CString(""))

	// plug the underlying C library's destructors to the instance of Model,
	// otherwise we get a memory-leak of the underlying struct
	runtime.SetFinalizer(model, finalizeModel)
}

//export logCallback
func logCallback(prob *C.lprec, modelPtr unsafe.Pointer, msg *C.char) {
	model, ok := loadRef(modelPtr).(*Model)
	if !ok {
		return
	}

	model.logger.Print(C.GoString(msg))
}

// finalizeModel is the function registered to be called upon garbage-
// collection of the model value
func finalizeModel(model *Model) {
	C.delete_lp(model.prob)
}

// Clone returns a copy of the model.
func (model *Model) Clone() *Model {
	model.mu.RLock()
	defer model.mu.RUnlock()

	newProb := C.copy_lp(model.prob)
	newVars := make([]*Variable, len(model.vars))
	newModel := &Model{
		prob:   newProb,
		logger: model.logger,
	}

	for i, v := range model.vars {
		newVars[i] = &Variable{
			model: newModel,
			index: v.index,
		}
	}

	newModel.vars = newVars

	newModel.finishInitialization()

	return newModel
}

// Name returns the name provided upon instantiation of a model
func (model *Model) Name() string {
	model.mu.RLock()
	defer model.mu.RUnlock()

	return C.GoString(C.get_lp_name(model.prob))
}

// SetDirection changes the direction of the model's optimization
func (model *Model) SetDirection(dir direction) {
	model.mu.Lock()
	defer model.mu.Unlock()

	C.set_sense(model.prob, C.uchar(dir))
}

// GetDirection returns the model's current optimization direction
func (model *Model) Direction() direction {
	model.mu.RLock()
	defer model.mu.RUnlock()

	if C.is_maxim(model.prob) == C.TRUE {
		return Maximize
	} else {
		return Minimize
	}
}

/* Column-related functions */

func (model *Model) VariableCount() int {
	model.mu.RLock()
	defer model.mu.RUnlock()

	return int(C.get_Ncolumns(model.prob))
}

// Variables returns a new slice with the model's variables. Changes to the slice will not be reflected in the model.
// Changes to the variables
func (model *Model) Variables() []*Variable {
	model.mu.RLock()
	defer model.mu.RUnlock()

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
//
// Empty names will automatically replaced by a unique name.
func (model *Model) AddVariable(name string) (v *Variable, err error) {
	return model.AddDefinedVariable(name, ContinuousVariable, 1, math.Inf(-1), math.Inf(1))
}

// AddBinaryVariable is a convenience function for adding a single
// named binary variable to the model, with a default coefficient of 1.
// Empty names will automatically replaced by a unique name.
func (model *Model) AddBinaryVariable(name string) (v *Variable, err error) {
	return model.AddDefinedVariable(name, BinaryVariable, 1, 0, 1)
}

// AddIntegerVariable is a convenience function for adding a single
// named unbounded integer variable to the model, with a default
// objective coefficient of 1.
// Empty names will automatically replaced by a unique name.
func (model *Model) AddIntegerVariable(name string) (v *Variable, err error) {
	return model.AddDefinedVariable(name, IntegerVariable, 1, math.Inf(-1), math.Inf(1))
}

// AddDefinedVariable add a variable to the linear programming model
// with its attributes passed as arguments.
// If varType is BinaryVariable, the bounds are ignored.
// Empty names will automatically replaced by a unique name.
func (model *Model) AddDefinedVariable(name string, varType VariableType, coefficient, lowerBound, upperBound float64) (v *Variable, err error) {
	size := model.VariableCount()

	func() {
		model.mu.Lock()
		defer model.mu.Unlock()

		v = new(Variable)
		v.index = size
		v.model = model
		model.vars = append(model.vars, v)

		// when adding a variable after some constraints have been defined,
		// we pass an array filled with zeroes to add_column, so the new
		// variable is assumed to not be used in the existing constraints
		C.add_columnex(model.prob, 0, nil, nil)
		// coef_array := make([]C.REAL, model.ConstraintCount()+1)
		// C.add_column(model.prob, &coef_array[0])

		if name == "" {
			name = fmt.Sprintf("V%d", size)
		}

		c_name := C.CString(name)
		defer C.free(unsafe.Pointer(c_name))

		C.set_col_name(model.prob, C.int(v.index+1), c_name)
	}()

	v.SetType(varType)
	v.SetObjectiveCoefficient(coefficient)
	if varType != BinaryVariable {
		v.SetBounds(lowerBound, upperBound)
	}

	return
}

// SetObjectiveFunction defines the objective function for the model as
// a slice of coefficients and a slice of its respective variables.
// E.g.: an objective function of the form 2x+3y is passed as:
//   SetObjectiveFunction([]float64{2,3}, []*Variable{x, y})
// Where x and y are the return values of one of the Add*Variable
// functions.
func (model *Model) SetObjectiveFunction(coefs []float64, vars []*Variable) error {
	for i, v := range vars {
		v.SetObjectiveCoefficient(coefs[i])
	}
	return nil
}

/* Constraint-related functions */

// ConstraintCount returns the number of individual constraints in
// the model
func (model *Model) ConstraintCount() int {
	model.mu.RLock()
	defer model.mu.RUnlock()

	return int(C.get_Nrows(model.prob))
}

// AddConstraint adds a constraint to the model as a lower and an upper
// bounds, a slice of variables and a slice of their respective
// coefficients.
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

// Solve attempts to find an optimal solution to the model.
// Information about the solution can be queried from the returned
// SolveResult value.
func (model *Model) Solve() (res *SolveResult, err error) {
	model.mu.Lock()
	defer model.mu.Unlock()

	res = new(SolveResult)
	res.model = model

	ret := C.solve(model.prob)

	switch ret {
	case C.OPTIMAL, C.SUBOPTIMAL:
		res.status = SolveStatus(ret)
		return res, nil
	case C.INFEASIBLE, C.UNBOUNDED, C.DEGENERATE, C.NUMFAILURE,
		C.USERABORT, C.TIMEOUT, C.PROCFAIL, C.PROCBREAK, C.FEASFOUND,
		C.NOFEASFOUND, C.NOMEMORY:
		return nil, SolveError(ret)
	default:
		panic("unrecognized result")
	}
}

//export abortCallback
func abortCallback(prob *C.lprec, ctxPtr unsafe.Pointer) C.int {
	ctx, ok := loadRef(ctxPtr).(context.Context)
	if ok && ctx.Err() != nil {
		return C.TRUE
	}

	return C.FALSE
}

// SolveWithContext wraps Solve() with a context. If the context is cancelled or times out, the solution search will be
// aborted and the context error will be returned.
// Note that if some solution has already been found, res.Status() will be SolutionSuboptimal.
func (model *Model) SolveWithContext(ctx context.Context) (res *SolveResult, err error) {
	C.put_abortfunc(model.prob, (*C.lphandle_intfunc)(C.abortCallback), saveRef(ctx))
	defer C.put_abortfunc(model.prob, nil, nil)

	ret, err := model.Solve()

	if errors.Is(err, ErrUserAbort) {
		return ret, ctx.Err()
	}

	return ret, err
}
