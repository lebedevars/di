package di

import (
	"fmt"
	"reflect"
)

type dependencyGraph struct {
	deps map[reflect.Type][]reflect.Type
}

func newDependencyGraph() *dependencyGraph {
	return &dependencyGraph{deps: make(map[reflect.Type][]reflect.Type)}
}

func (graph *dependencyGraph) addDependency(from, to reflect.Type) {
	graph.deps[from] = append(graph.deps[from], to)
}

// detectCyclicDependencies uses DFS to determine if the dependency graph is cyclic
func (graph *dependencyGraph) detectCyclicDependencies() error {
	visited := make(map[reflect.Type]bool)
	recStack := make(map[reflect.Type]bool)
	for t := range graph.deps {
		if cyclic, dep := graph.isCyclic(t, visited, recStack); cyclic {
			return fmt.Errorf("cyclic dependency detected between %s and %s", t, dep)
		}
	}

	return nil
}

func (graph *dependencyGraph) isCyclic(t reflect.Type, visited, recStack map[reflect.Type]bool) (bool, reflect.Type) {
	if recStack[t] {
		return true, t
	}

	if visited[t] {
		return false, nil
	}

	recStack[t] = true
	visited[t] = true

	for _, dep := range graph.deps[t] {
		if cyclic, _ := graph.isCyclic(dep, visited, recStack); cyclic {
			return true, dep
		}
	}

	recStack[t] = false
	return false, nil
}
