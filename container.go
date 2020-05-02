package di

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type (
	Container struct {
		m             sync.RWMutex
		graph         *dependencyGraph
		constructors  map[reflect.Type]innerConstructor
		cache         map[reflect.Type]reflect.Value
		contextParams ContextParams
	}

	// innerConstructor calls provider with arguments resolved from the Container
	innerConstructor func(*Container) reflect.Value

	Scope int

	// ContextParams represents container parameters. When
	ContextParams map[string]interface{}
)

const (
	Singleton Scope = 1
	Request   Scope = 2
)

var (
	notAFunctionError = errors.New("argument is not a function")
	contextParamsType = reflect.TypeOf(ContextParams{})
)

// NewContainer creates a new container
func NewContainer() *Container {
	return &Container{
		m:             sync.RWMutex{},
		graph:         newDependencyGraph(),
		constructors:  make(map[reflect.Type]innerConstructor),
		cache:         make(map[reflect.Type]reflect.Value),
		contextParams: make(map[string]interface{}),
	}
}

// WithContext returns container with added contextParams values without changing the original one.
// Context allows to change how dependencies are instantiated.
// Context values can be retrieved in provider functions:
//  err := c.Register(func(params di.ContextParams) *example {
//		return newExample(params.GetValue("key").(string))
//	}, Request)
func (c *Container) WithContext(key string, value interface{}) *Container {
	newContext := make(map[string]interface{})
	for k, v := range c.contextParams {
		newContext[k] = v
	}

	newContext[key] = value
	newContainer := &Container{
		graph:         c.graph,
		constructors:  c.constructors,
		cache:         c.cache,
		contextParams: newContext,
	}

	return newContainer
}

// GetValues returns values from context params
func (contextParams ContextParams) GetValue(key string) interface{} {
	return contextParams[key]
}

// Register teaches the container how to resolve dependencies: provider's out-parameter
// needs all of its inner parameters to be instantiated.
// If ContextParams type is passed as an argument, it will give access to container's
// context parameters.
func (c *Container) Register(provider interface{}, scope Scope) error {
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
		// skip ContextParams
		if argType == contextParamsType {
			continue
		}

		c.graph.addDependency(outType, argType)
		if _, ok := c.constructors[argType]; !ok {
			c.constructors[argType] = nil
		}
	}

	providerValue := reflect.ValueOf(provider)
	// resolve each argument and call provider
	innerConstructor := getConstructor(numIn, argTypes, providerValue)

	// create entries in cache for singletons
	if scope == Singleton {
		c.cache[outType] = reflect.Value{}
	}

	c.constructors[outType] = innerConstructor
	return nil
}

func getConstructor(numIn int, argTypes []reflect.Type, providerValue reflect.Value) func(con *Container) reflect.Value {
	return func(con *Container) reflect.Value {
		args := make([]reflect.Value, numIn)
		for i, argType := range argTypes {
			// get value of ContextParams
			if argType == contextParamsType {
				args[i] = reflect.ValueOf(con.contextParams)
				continue
			}

			// if arg exists in cache - retrieve it
			if val, ok := con.cache[argType]; ok {
				args[i] = val
			}
			// otherwise - call constructor for that type
			args[i] = con.constructors[argType](con)
		}

		return providerValue.Call(args)[0]
	}
}

// Build checks dependency graph for cyclic dependencies, checks if all dependencies
// were registered and created singletons
func (c *Container) Build() error {
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
			c.cache[t] = c.constructors[t](c)
		}
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

// Invoke calls invoker with resolved arguments
func (c *Container) Invoke(invoker interface{}) error {
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

		// get ContextParams
		if argType == contextParamsType {
			args[i] = reflect.ValueOf(c.contextParams)
			continue
		}

		if cachedValue, ok := c.cache[argType]; ok {
			args[i] = cachedValue
		} else {
			constructor, ok := c.constructors[argType]
			if !ok {
				return fmt.Errorf("dependency %s was not registered", argType)
			}

			args[i] = constructor(c)
		}
	}

	// call invoker with resolved arguments
	reflect.ValueOf(invoker).Call(args)
	return nil
}

// Get returns dependency of type t
func (c *Container) Get(t reflect.Type) (interface{}, error) {
	if cachedValue, ok := c.cache[t]; ok {
		return cachedValue.Interface(), nil
	} else {
		constructor, ok := c.constructors[t]
		if !ok {
			return nil, fmt.Errorf("dependency %s was not registered", t)
		}

		return constructor(c).Interface(), nil
	}
}
