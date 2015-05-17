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

/* Types */

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
	//ErrorPresolved        = solveError(C.PRESOLVED) // we can't use C.set_presolve because it might remove variables
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

// GetStatus reports if the solution is optimal (SolutionOptimal) or
// not (SolutionSuboptimal)
func (res solveResult) GetStatus() solveStatus {
	return res.status
}

func (res solveResult) GetValue(v *variable) float64 {
	return res.GetPrimalValue(v)
}

func (res solveResult) GetPrimalValue(v *variable) float64 {
	// get_var_*result uses funny indexing: 0=objective,1 to Nrows=constraint,Nrows to Nrows+Ncols=variable
	return float64(C.get_var_primalresult(res.model.prob, C.int(v.index+v.model.GetConstraintCount()+1)))
}

func (res solveResult) GetDualValue(v *variable) float64 {
	// get_var_*result uses funny indexing: 0=objective,1 to Nrows=constraint,Nrows to Nrows+Ncols=variable
	return float64(C.get_var_dualresult(res.model.prob, C.int(v.index+v.model.GetConstraintCount()+1)))
}

func (res solveResult) GetObjectiveValue() float64 {
	return float64(C.get_objective(res.model.prob))
}
