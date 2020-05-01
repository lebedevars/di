package di

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraph(t *testing.T) {
	g := newDependencyGraph()
	g.addDependency(reflect.TypeOf(&example{}), reflect.TypeOf(&example2{}))
	g.addDependency(reflect.TypeOf(&example2{}), reflect.TypeOf(&example{}))
	g.addDependency(reflect.TypeOf(&example3{}), reflect.TypeOf(&example{}))
	err := g.detectCyclicDependencies()
	assert.Error(t, err)
}
