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

type BranchCutResult struct {
	model *Model
}

type BranchCutStatus C.int

const (
	BranchCutSolutionOptimal    = C.GLP_OPT
	BranchCutSolutionFeasible   = C.GLP_FEAS
	BranchCutNoFeasibleSolution = C.GLP_NOFEAS
	BranchCutSolutionUndefined  = C.GLP_UNDEF
)

// SolveBranchCut solves the linear programming model using the
// branch-and-cut algorithm. Better suited for mixed-integer linear
// programs, i.e.: problems with integer and/or binary variables.
func (model *Model) SolveBranchCut() (result *BranchCutResult, err error) {
	result = new(BranchCutResult)
	result.model = model
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

	if err := glpkError(C.glp_intopt(model.prob, &parm)); err != nil {
		return nil, err
	}
	return result, nil
}

/* Result-related functions */

func (res BranchCutResult) GetStatus() SimplexStatus {
	return SimplexStatus(C.glp_mip_status(res.model.prob))
}

func (res BranchCutResult) GetValue(v *Variable) float64 {
	return float64(C.glp_mip_col_val(res.model.prob, C.int(v.index+1)))
}

func (res BranchCutResult) GetObjectiveValue() float64 {
	return float64(C.glp_mip_obj_val(res.model.prob))
}
