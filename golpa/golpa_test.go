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

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	epsilon = 0.0000001 // acceptable numerical deviation for test results
)

func TestInstantiation(t *testing.T) {
	name := "test model 1"
	model := NewModel(name, Maximize)
	if model.GetName() != name {
		t.Fatal("model name did not survive instantiation")
	}
	if model.GetDirection() != Maximize {
		t.Fatal("optimization direction did not survive instantiation")
	}
}

func TestAddVariableWithDetails(t *testing.T) {
	model := NewModel("test", Maximize)
	v1, _ := model.AddDefinedVariable("x", BinaryVariable, 3.1416, 0, 1)
	if v1.GetName() != "x" {
		t.Fatal("variable name did not survive instantiation")
	}
	if v1.GetType() != BinaryVariable {
		t.Fatal("variable type did not survive instantiation")
	}
	if v1.GetCoefficient() != 3.1416 {
		t.Fatal("variable coefficient did not survive instantiation")
	}
	if l, h := v1.GetBounds(); l != 0 || h != 1 {
		t.Fatal("variable bounds did not survive instantiation")
	}

	v2, _ := model.AddDefinedVariable("y", ContinuousVariable, -1, math.Inf(-1), 5)
	if v2.GetName() != "y" {
		t.Fatal("variable name did not survive instantiation")
	}
	if v2.GetType() != ContinuousVariable {
		t.Fatal("variable type did not survive instantiation")
	}
	if v2.GetCoefficient() != -1 {
		t.Fatal("variable coefficient did not survive instantiation")
	}
	if l, h := v2.GetBounds(); l != math.Inf(-1) || h != 5 {
		t.Fatal("variable bounds did not survive instantiation")
	}
}

func TestSetObjectiveFunction(t *testing.T) {
	model := NewModel("test", Maximize)
	v1, _ := model.AddVariable("x")
	v2, _ := model.AddVariable("y")
	v2.SetType(IntegerVariable)
	v3, _ := model.AddVariable("y")
	v3.SetType(BinaryVariable)

	vars := []*Variable{v1, v2, v3}
	coefs := []float64{1.3, 2.7182, 3.1416}
	model.SetObjectiveFunction(coefs, vars)
	for i, coef := range coefs {
		if vars[i].GetCoefficient() != coef {
			t.Fatalf("%s coefficient not set correctly while defining objective function", v1)
		}
	}
}

func TestSolveMIP(t *testing.T) {
	model := NewModel("test", Maximize)
	x1, _ := model.AddDefinedVariable("x1", ContinuousVariable, 1, 0, 40)
	x2, _ := model.AddDefinedVariable("x2", ContinuousVariable, 2, 0, math.Inf(1))
	x3, _ := model.AddDefinedVariable("x3", ContinuousVariable, 3, 0, math.Inf(1))
	x4, _ := model.AddDefinedVariable("x3", IntegerVariable, 1, 2, 3)

	model.AddConstraint(0, 20, []*Variable{x1, x2, x3, x4}, []float64{-1, 1, 1, 10})
	model.AddConstraint(0, 30, []*Variable{x1, x2, x3}, []float64{1, -3, 1})
	model.AddConstraint(0, 0, []*Variable{x2, x4}, []float64{1, -3.5})

	if res, err := model.Solve(); err != nil {
		t.Fatalf("model solving failed: %s", err)
	} else {
		expected_xs := []float64{40, 10.5, 19.5, 3}
		expected_obj := 122.5

		if res.GetStatus() != SolutionOptimal {
			t.Errorf("solution should have been optimal")
		}

		// ignore numerical inaccuracies
		if math.Abs(res.GetObjectiveValue()-expected_obj) > epsilon {
			t.Errorf("objective function value did not match expectation: %v != %v", res.GetObjectiveValue(), expected_obj)
		}
		for i, x := range []*Variable{x1, x2, x3, x4} {
			if math.Abs(res.GetValue(x)-expected_xs[i]) > epsilon {
				t.Errorf("result of %s did not match expectation: %f != %f", x.GetName(), res.GetValue(x), expected_xs[i])
			}
		}
	}
}

func TestSolveLP(t *testing.T) {
	model := NewModel("test", Maximize)
	x1, _ := model.AddDefinedVariable("x1", ContinuousVariable, 1, 0, math.Inf(1))
	x2, _ := model.AddDefinedVariable("x2", ContinuousVariable, 2, 0, math.Inf(1))
	x3, _ := model.AddDefinedVariable("x3", ContinuousVariable, -1, 0, math.Inf(1))

	model.AddConstraint(0, 14, []*Variable{x1, x2, x3}, []float64{2, 1, 1})
	model.AddConstraint(0, 28, []*Variable{x1, x2, x3}, []float64{4, 2, 3})
	model.AddConstraint(0, 30, []*Variable{x1, x2, x3}, []float64{2, 5, 5})

	if res, err := model.Solve(); err != nil {
		t.Fatalf("model solving failed: %s", err)
	} else {
		expected_xs := []float64{5, 4, 0}
		expected_obj := 13.0

		if res.GetStatus() != SolutionOptimal {
			t.Errorf("solution should have been optimal")
		}

		// ignore numerical inaccuracies
		if math.Abs(res.GetObjectiveValue()-expected_obj) > epsilon {
			t.Errorf("objective function value did not match expectation: %f != %f", res.GetObjectiveValue(), expected_obj)
		}
		for i, x := range []*Variable{x1, x2, x3} {
			if math.Abs(res.GetValue(x)-expected_xs[i]) > epsilon {
				t.Errorf("result of %s did not match expectation: %f != %f", x.GetName(), res.GetValue(x), expected_xs[i])
			}
		}
	}
}

func TestBig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	num_vars := 100000
	model := NewModel("test", Maximize)
	vars := make([]*Variable, num_vars)
	coefs := make([]float64, num_vars)
	for i := 0; i < num_vars; i++ {
		v, _ := model.AddDefinedVariable(fmt.Sprintf("x%d", i), ContinuousVariable, 1, 0, math.Inf(1))
		vars[i] = v
		coefs[i] = 1
		model.AddConstraint(0, float64(i), []*Variable{v}, []float64{1})
	}
	model.AddConstraint(-100, 100, vars, coefs)
	res, err := model.Solve()
	if err != nil {
		t.Fatal("error solving model: %s", err)
	}
	if val := res.GetObjectiveValue(); val != 100 {
		t.Fatal("model did not maximize to 100")
	}
}

// Try to detect non-reentrant code in underlying lib
func TestParallel(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		TestBig(t)
	}()
	go func() {
		defer wg.Done()
		TestBig(t)
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
