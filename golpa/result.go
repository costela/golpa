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
	ErrorModelInfeasible  = SolveError(C.INFEASIBLE)
	ErrorModelUnbounded   = SolveError(C.UNBOUNDED)
	ErrorModelDegenerate  = SolveError(C.DEGENERATE)
	ErrorNumericalFailure = SolveError(C.NUMFAILURE)
	ErrorUserAbort        = SolveError(C.USERABORT) // we don't use C.put_abortfunc
	ErrorTimeout          = SolveError(C.TIMEOUT)   // FIXME: support C.set_timeout
	//ErrorPresolved        = SolveError(C.PRESOLVED) // we can't use C.set_presolve because it might remove variables
	ErrorBranchCutFail    = SolveError(C.PROCFAIL)
	ErrorBranchCutBreak   = SolveError(C.PROCBREAK) // we don't use set_break_at_first/set_break_at_value
	ErrorFeasibleFound    = SolveError(C.FEASFOUND)
	ErrorNoFeasibleFound  = SolveError(C.NOFEASFOUND)
	ErrorNoMemory         = SolveError(C.NOMEMORY)
)

// Error returns a string representation of the given error value.
func (e SolveError) Error() string {
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

// GetStatus reports if the solution is optimal (SolutionOptimal) or
// not (SolutionSuboptimal)
func (res SolveResult) GetStatus() SolveStatus {
	return res.status
}

// GetValue returns the computed value of the given variable for this
// optimization result.
// This is a shorthand for GetPrimalValue.
func (res SolveResult) GetValue(v *Variable) float64 {
	return res.GetPrimalValue(v)
}

// GetPrimalValue returns the computed value of the given variable for
// this optimization result.
func (res SolveResult) GetPrimalValue(v *Variable) float64 {
	// get_var_*result uses funny indexing: 0=objective,1 to Nrows=constraint,Nrows to Nrows+Ncols=variable
	return float64(C.get_var_primalresult(res.model.prob, C.int(v.index+v.model.GetConstraintCount()+1)))
}

// GetDualValue returns the dual value of the given variable in this
// optimization result.
func (res SolveResult) GetDualValue(v *Variable) float64 {
	// get_var_*result uses funny indexing: 0=objective,1 to Nrows=constraint,Nrows to Nrows+Ncols=variable
	return float64(C.get_var_dualresult(res.model.prob, C.int(v.index+v.model.GetConstraintCount()+1)))
}

// GetObjectiveValue returns the value of the objective function for
// this optimization result. This value is only optimal if GetStatus
// also returns SolutionOptimal.
func (res SolveResult) GetObjectiveValue() float64 {
	return float64(C.get_objective(res.model.prob))
}
