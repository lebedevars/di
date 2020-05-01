package di

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type (
	container struct {
		m            sync.RWMutex
		graph        *dependencyGraph
		constructors map[reflect.Type]innerConstructor
		cache        map[reflect.Type]reflect.Value
	}

	// innerConstructor calls provider with arguments resolved from the container
	innerConstructor func() reflect.Value

	Scope int
)

const (
	Singleton Scope = 1
	Request   Scope = 2
)

var (
	notAFunctionError = errors.New("argument is not a function")
)

func NewContainer() *container {
	return &container{
		m:            sync.RWMutex{},
		graph:        newDependencyGraph(),
		constructors: make(map[reflect.Type]innerConstructor),
		cache:        make(map[reflect.Type]reflect.Value),
	}
}

// Register registers the provider's out argument with the provider's parameters as dependencies
func (c *container) Register(provider interface{}, scope Scope) error {
	providerType := reflect.TypeOf(provider)
	if providerType.Kind() != reflect.Func {
		return notAFunctionError
	}

	c.m.Lock()
	defer c.m.Unlock()

	numOut := providerType.NumOut()
	if numOut != 1 {
		return errors.New("only 1 out parameter is allowed")
	}

	outType := providerType.Out(0)
	_, ok := c.graph.deps[outType]
	if ok {
		return fmt.Errorf("dependency %s was already registered", outType)
	} else {
		c.graph.addDependency(outType, nil)
	}

	numIn := providerType.NumIn()
	argTypes := make([]reflect.Type, numIn)
	for i := 0; i < numIn; i++ {
		argTypes[i] = providerType.In(i)
	}

	// out-parameter depends on all of the in-parameters
	for _, argType := range argTypes {
		c.graph.addDependency(outType, argType)
		if _, ok := c.constructors[argType]; !ok {
			c.constructors[argType] = nil
		}
	}

	providerValue := reflect.ValueOf(provider)
	// resolve each argument and call provider
	innerConstructor := func() reflect.Value {
		args := make([]reflect.Value, numIn)
		for i, argType := range argTypes {
			// if arg exists in cache - retrieve it
			if val, ok := c.cache[argType]; ok {
				args[i] = val
			}
			// otherwise - call constructor for that type
			args[i] = c.constructors[argType]()
		}

		return providerValue.Call(args)[0]
	}

	// create entries in cache for singletons
	if scope == Singleton {
		c.cache[outType] = reflect.Value{}
	}

	c.constructors[outType] = innerConstructor
	return nil
}

// Build checks dependency graph for cyclic dependencies, checks if all dependencies
// were registered and created singletons
func (c *container) Build() error {
	err := c.graph.detectCyclicDependencies()
	if err != nil {
		return err
	}

	errs := make([]string, 0)
	for t, innerConstructor := range c.constructors {
		// check all innerConstructors, if any of them is nil - no provider was registered for that dependency
		if innerConstructor == nil {
			errs = append(errs, fmt.Sprintf("type %s was not registered", t))
		}

		// if there needs to be a cached value (singleton) - create it
		if _, ok := c.cache[t]; ok {
			c.cache[t] = c.constructors[t]()
		}
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

// Invoke calls invoker with resolved arguments
func (c *container) Invoke(invoker interface{}) error {
	invokerType := reflect.TypeOf(invoker)
	if invokerType.Kind() != reflect.Func {
		return notAFunctionError
	}

	// resolve each argument type, if found in cache - return value,
	// otherwise call innerConstructor for that type
	numIn := invokerType.NumIn()
	args := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		argType := invokerType.In(i)
		if cachedValue, ok := c.cache[argType]; ok {
			args[i] = cachedValue
		} else {
			constructor, ok := c.constructors[argType]
			if !ok {
				return fmt.Errorf("dependency %s was not registered", argType)
			}

			args[i] = constructor()
		}
	}

	// call invoker with resolved arguments
	reflect.ValueOf(invoker).Call(args)
	return nil
}
