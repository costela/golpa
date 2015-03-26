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

type SimplexResult struct {
	model *Model
}

type SimplexStatus C.int

const (
	SimplexSolutionOptimal    = C.GLP_OPT
	SimplexSolutionFeasible   = C.GLP_FEAS
	SimplexSolutionInfeasible = C.GLP_INFEAS
	SimplexNoFeasibleSolution = C.GLP_NOFEAS
	SimplexSolutionUnbounded  = C.GLP_UNBND
	SimplexSolutionUndefined  = C.GLP_UNDEF
)

// SolveSimplex solves the linear programming model using the
// primal simplex algorithm.
func (model *Model) SolveSimplex() (res *SimplexResult, err error) {
	return model.solveSimplex(C.GLP_PRIMAL)
}

// SolveSimplex solves the linear programming model using the
// dual simplex algorithm.
func (model *Model) SolveSimplexDual() (res *SimplexResult, err error) {
	return model.solveSimplex(C.GLP_DUALP)
}

func (model *Model) solveSimplex(method C.int) (result *SimplexResult, err error) {
	result = new(SimplexResult)
	result.model = model
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

	if err := glpkError(C.glp_simplex(model.prob, &parm)); err != nil {
		return nil, err
	}
	return result, nil
}

/* Result-related functions */

func (res SimplexResult) GetStatus() SimplexStatus {
	return SimplexStatus(C.glp_get_status(res.model.prob))
}

func (res SimplexResult) GetValue(v *Variable) float64 {
	return res.GetPrimalValue(v)
}

func (res SimplexResult) GetPrimalValue(v *Variable) float64 {
	return float64(C.glp_get_col_prim(res.model.prob, C.int(v.index+1)))
}

func (res SimplexResult) GetDualValue(v *Variable) float64 {
	return float64(C.glp_get_col_dual(res.model.prob, C.int(v.index+1)))
}

func (res SimplexResult) GetObjectiveValue() float64 {
	return float64(C.glp_get_obj_val(res.model.prob))
}
