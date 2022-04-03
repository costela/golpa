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

/* Types */

type SolveResult struct {
	model  *Model
	status SolveStatus
}

type SolveStatus C.int

const (
	SolutionOptimal    = SolveStatus(C.OPTIMAL)
	SolutionSuboptimal = SolveStatus(C.SUBOPTIMAL)
)

type SolveError C.int

const (
	ErrBranchCutBreak   = SolveError(C.PROCBREAK)
	ErrBranchCutFail    = SolveError(C.PROCFAIL)
	ErrFeasibleFound    = SolveError(C.FEASFOUND)
	ErrModelDegenerate  = SolveError(C.DEGENERATE)
	ErrModelInfeasible  = SolveError(C.INFEASIBLE)
	ErrModelUnbounded   = SolveError(C.UNBOUNDED)
	ErrNoFeasibleFound  = SolveError(C.NOFEASFOUND)
	ErrNoMemory         = SolveError(C.NOMEMORY)
	ErrNumericalFailure = SolveError(C.NUMFAILURE)
	ErrPresolved        = SolveError(C.PRESOLVED) // should not be seen: we can't use C.set_presolve because it might remove variables behind our backs
	ErrTimeout          = SolveError(C.TIMEOUT)
	ErrUserAbort        = SolveError(C.USERABORT)
)

// Error returns a string representation of the given error value.
func (e SolveError) Error() string {
	switch e {
	case ErrBranchCutBreak:
		return "branch-and-cut stopped at beakpoint"
	case ErrBranchCutFail:
		return "branch-and-cut failure"
	case ErrFeasibleFound:
		return "feasible but non-integer solution found"
	case ErrModelDegenerate:
		return "model is degenerate"
	case ErrModelInfeasible:
		return "model is infeasible"
	case ErrModelUnbounded:
		return "model is unbounded"
	case ErrNoFeasibleFound:
		return "no feasible solution found"
	case ErrNoMemory:
		return "ran out of memory while solving"
	case ErrNumericalFailure:
		return "numerical failure while solving"
	case ErrPresolved:
		return "model was presolved"
	case ErrTimeout:
		return "timeout occurred before any integer solution could be found"
	case ErrUserAbort:
		return "aborted by user abort function "
	default:
		panic("unrecognized error")
	}
}

// Status reports if the solution is optimal (SolutionOptimal) or
// not (SolutionSuboptimal)
func (res SolveResult) Status() SolveStatus {
	return res.status
}

// Value returns the computed value of the given variable for this
// optimization result.
// This is a shorthand for PrimalValue.
func (res SolveResult) Value(v *Variable) float64 {
	return res.PrimalValue(v)
}

// PrimalValue returns the computed value of the given variable for
// this optimization result.
func (res SolveResult) PrimalValue(v *Variable) float64 {
	res.model.mu.RLock()
	defer res.model.mu.RUnlock()

	// get_var_*result uses funny indexing: 0=objective,1 to Nrows=constraint,Nrows to Nrows+Ncols=variable
	return float64(C.get_var_primalresult(res.model.prob, C.int(v.index+v.model.ConstraintCount()+1)))
}

// DualValue returns the dual value of the given variable in this
// optimization result.
func (res SolveResult) DualValue(v *Variable) float64 {
	res.model.mu.RLock()
	defer res.model.mu.RUnlock()

	// get_var_*result uses funny indexing: 0=objective,1 to Nrows=constraint,Nrows to Nrows+Ncols=variable
	return float64(C.get_var_dualresult(res.model.prob, C.int(v.index+v.model.ConstraintCount()+1)))
}

// ObjectiveValue returns the value of the objective function for
// this optimization result. This value is only optimal if Status
// also returns SolutionOptimal.
func (res SolveResult) ObjectiveValue() float64 {
	res.model.mu.RLock()
	defer res.model.mu.RUnlock()

	return float64(C.get_objective(res.model.prob))
}
