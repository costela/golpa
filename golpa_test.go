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

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	delta = 0.0000001 // acceptable numerical deviation for test results
)

var (
	bigModel     *Model
	bigModelOnce sync.Once
)

func getBigModelCopy(t *testing.T) *Model {
	t.Helper()

	bigModelOnce.Do(func() {
		num_vars := 10000
		model, err := NewModel("testBig", Maximize)
		require.NoError(t, err)

		vars := make([]*Variable, num_vars)
		coefs := make([]float64, num_vars)
		for i := 0; i < num_vars; i++ {
			v, _ := model.AddIntegerVariable(fmt.Sprintf("x%d", i))
			vars[i] = v
			coefs[i] = 1
			err := model.AddConstraint(-float64(i), float64(i), []*Variable{v}, []float64{1})
			require.NoError(t, err)
		}

		bigModel = model
	})

	return bigModel.Clone()
}

func TestInstantiation(t *testing.T) {
	name := "test model 1"
	model, err := NewModel(name, Maximize)
	require.NoError(t, err)

	assert.Equal(t, name, model.Name())
	assert.Equal(t, Maximize, model.Direction())
}

func TestClone(t *testing.T) {
	name := "test model 1"
	model, err := NewModel(name, Maximize)
	require.NoError(t, err)

	v, err := model.AddDefinedVariable("x", ContinuousVariable, 1, 2, 3)
	require.NoError(t, err)

	err = model.AddConstraint(0, 1, []*Variable{v}, []float64{1})
	require.NoError(t, err)

	modelClone := model.Clone()

	assert.Equal(t, model.Name(), modelClone.Name())
	assert.Equal(t, model.Direction(), modelClone.Direction())
	assert.Equal(t, model.VariableCount(), modelClone.VariableCount())
	assert.Equal(t, model.ConstraintCount(), modelClone.ConstraintCount())
}

func TestAddVariableWithDetails(t *testing.T) {
	model, err := NewModel("test", Maximize)
	require.NoError(t, err)

	v1, err := model.AddDefinedVariable("x", BinaryVariable, 3.1416, 0, 1)
	require.NoError(t, err)

	assert.Equal(t, "x", v1.Name())
	assert.Equal(t, BinaryVariable, v1.Type())
	assert.Equal(t, 3.1416, v1.Coefficient())
	l, h := v1.Bounds()
	assert.Equal(t, 0.0, l)
	assert.Equal(t, 1.0, h)

	v2, err := model.AddDefinedVariable("y", ContinuousVariable, -1, math.Inf(-1), 5)
	require.NoError(t, err)

	assert.Equal(t, "y", v2.Name())
	assert.Equal(t, ContinuousVariable, v2.Type())
	assert.Equal(t, -1.0, v2.Coefficient())
	l, h = v2.Bounds()
	assert.Equal(t, math.Inf(-1), l)
	assert.Equal(t, 5.0, h)
}

func TestSetObjectiveFunction(t *testing.T) {
	model, err := NewModel("test", Maximize)
	require.NoError(t, err)

	v1, _ := model.AddVariable("x")
	v2, _ := model.AddVariable("y")
	v2.SetType(IntegerVariable)
	v3, _ := model.AddVariable("y")
	v3.SetType(BinaryVariable)

	vars := []*Variable{v1, v2, v3}
	coefs := []float64{1.3, 2.7182, 3.1416}
	model.SetObjectiveFunction(coefs, vars)
	for i, coef := range coefs {
		assert.Equal(t, coef, vars[i].Coefficient())
	}
}

func TestSolveMIP(t *testing.T) {
	model, err := NewModel("test", Maximize)
	require.NoError(t, err)

	x1, _ := model.AddDefinedVariable("x1", ContinuousVariable, 1, 0, 40)
	x2, _ := model.AddDefinedVariable("x2", ContinuousVariable, 2, 0, math.Inf(1))
	x3, _ := model.AddDefinedVariable("x3", ContinuousVariable, 3, 0, math.Inf(1))
	x4, _ := model.AddDefinedVariable("x3", IntegerVariable, 1, 2, 3)

	model.AddConstraint(0, 20, []*Variable{x1, x2, x3, x4}, []float64{-1, 1, 1, 10})
	model.AddConstraint(0, 30, []*Variable{x1, x2, x3}, []float64{1, -3, 1})
	model.AddConstraint(0, 0, []*Variable{x2, x4}, []float64{1, -3.5})

	res, err := model.Solve()
	require.NoError(t, err)

	expected_xs := []float64{40, 10.5, 19.5, 3}
	expected_obj := 122.5

	assert.Equal(t, SolutionOptimal, res.Status())

	// ignore numerical inaccuracies
	assert.InDelta(t, expected_obj, res.ObjectiveValue(), delta)

	for i, x := range []*Variable{x1, x2, x3, x4} {
		assert.InDelta(t, expected_xs[i], res.Value(x), delta)
	}
}

func TestSolveLP(t *testing.T) {
	model, err := NewModel("test", Maximize)
	require.NoError(t, err)

	x1, _ := model.AddDefinedVariable("x1", ContinuousVariable, 1, 0, math.Inf(1))
	x2, _ := model.AddDefinedVariable("x2", ContinuousVariable, 2, 0, math.Inf(1))
	x3, _ := model.AddDefinedVariable("x3", ContinuousVariable, -1, 0, math.Inf(1))

	model.AddConstraint(0, 14, []*Variable{x1, x2, x3}, []float64{2, 1, 1})
	model.AddConstraint(0, 28, []*Variable{x1, x2, x3}, []float64{4, 2, 3})
	model.AddConstraint(0, 30, []*Variable{x1, x2, x3}, []float64{2, 5, 5})

	res, err := model.Solve()
	require.NoError(t, err)

	expected_xs := []float64{5, 4, 0}
	expected_obj := 13.0

	assert.Equal(t, SolutionOptimal, res.Status())

	// ignore numerical inaccuracies
	assert.InDelta(t, expected_obj, res.ObjectiveValue(), delta)

	for i, x := range []*Variable{x1, x2, x3} {
		assert.InDelta(t, expected_xs[i], res.Value(x), delta)
	}
}

func TestBig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	model := getBigModelCopy(t)

	res, err := model.Solve()
	require.NoError(t, err)

	expected := 49995000.0
	assert.Equal(t, expected, res.ObjectiveValue())
}

func TestContext(t *testing.T) {
	model := getBigModelCopy(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := model.SolveWithContext(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// Try to detect non-reentrant code in underlying lib
func TestParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	model := getBigModelCopy(t)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		model.Solve()
	}()
	go func() {
		defer wg.Done()
		model.Solve()
	}()
	wg.Wait()
}

/* Benchmarks */

/*
 * BenchmarkMemoryLeaks is a hack to check if the GC really gets rid of
 * unreferenced model values.
 */
func BenchmarkMemoryLeaks(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}
	b.ReportAllocs()
	const n = 100000
	for i := 0; i < n; i++ {
		NewModel(strconv.Itoa(i), Minimize)
	}
	runtime.GC()
	time.Sleep(10 * time.Second)
}
