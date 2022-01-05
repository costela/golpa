# Go Linear Programming Abstraction
[![Build Status](https://travis-ci.org/costela/golpa.svg)](https://travis-ci.org/costela/golpa)
[![GoDoc](https://godoc.org/github.com/costela/golpa/golpa?status.svg)](https://godoc.org/github.com/costela/golpa/golpa)

GoLPA is a library for modeling and solving linear programming problems. It uses [lp\_solve](http://lpsolve.sourceforge.net) for the actual number crunching, but offers a slightly higher-level interface for problem modeling relative to the underlying library.
Since the intention is providing a simpler interface, the underlying API is not completely exposed. If there are features of the low-level library you'd like to see exposed by GoLPA, please open an issue.

**Warning**: the API is currently not stable.

# Dependencies

GoLPA requires the lp\_solve libraries to be accessible. On Linux systems, this means the liblpsolve55-dev (Debian, etc) or lpsolve-devel (Red Hat, etc) package must be installed.

# Installing

```bash
$ go get github.com/costela/golpa
```

# Example usage

The model of the following problem:

```
Maximize:
  z = x1 + 2 x2 - 3 x3
With:
  0 <= x1 <= 40
  5 <= x3 <= 11
Subject to:
  0 <= - x1 + x2 + 5.3 x3 <= 10
  -inf <= 2 x1 - 5 x2 + 3 x3 <= 20
  x2 - 8 x3 = 0
```

can be expressed with GoLPA like this:

```go
package main

import (
    "github.com/costela/golpa/golpa"
    "math"
    "fmt"
)

func main() {
  model := golpa.NewModel("some model", golpa.Maximize)
  x1, _ := model.AddVariable("x1")
  x1.SetBounds(0, 40)
  x2, _ := model.AddVariable("x2")
  x2.SetObjectiveCoefficient(2)
  // alternatively, all information pertaining can be given at once:
  x3, _ := model.AddDefinedVariable("x3", golpa.ContinuousVariable, 3, 5, 11)

  model.AddConstraint(0, 10, []*golpa.Variable{x1, x2, x3}, []float64{-1, 1, 5.3})
  model.AddConstraint(math.Inf(-1), 20, []*golpa.Variable{x1, x2, x3}, []float64{2, -5, 3})
  model.AddConstraint(0, 0, []*golpa.Variable{x1, x3}, []float64{1, -8})
  ⋮
```

The model can than be solved and the resulting values can than be retrieved as follows:

```go
  ⋮
  result, _ := model.Solve() // you should check for errors

  fmt.Printf("solution optimal? %t", result.Status() == golpa.SolutionOptimal)
  fmt.Printf("z = %f\n", result.ObjectiveValue())
  fmt.Printf("x1 = %f\n", result.Value(x1))
  ⋮
}

```
