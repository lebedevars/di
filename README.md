![build](https://github.com/lebedevars/di/workflows/build/badge.svg)
[![codecov](https://codecov.io/gh/lebedevars/di/branch/master/graph/badge.svg)](https://codecov.io/gh/lebedevars/di)
[![Go Report Card](https://goreportcard.com/badge/lebedevars/di)](https://goreportcard.com/report/lebedevars/di)
[![codebeat badge](https://codebeat.co/badges/9ebc2040-753c-4184-bd9c-7f4abb7a7a3a)](https://codebeat.co/projects/github-com-lebedevars-di-master)

# About
- Heavily uses reflection
- Capable of scoped resolution of dependencies

# Installation
```$ go get -u github.com/lebedevars/di```

# Usage
## Basics
Pass a provider function and lifetime value to Register to teach the container how to build dependencies.
Provider function must have 1 out-parameter. All of provider's arguments need to be registered as well.
```go
c := di.NewContainer()
// *SomeDep has no dependencies
err := c.Register(func() *SomeDep {
  return NewSomeDep()
}, di.Singleton)

// *SomeOtherDep depends on *SomeDep
err = c.Register(func(someDep *SomeDep) *SomeOtherDep {
  return NewSomeOtherDep(someDep)
}, di.Singleton)
```
Build the container after setting it up. Build checks for errors in configuration, such as cyclic or missing dependencies. If everything is correct, it instantiates singletons and the container is ready for use:
```go
err = c.Build()
```
Now you can use it to resolve dependencies. Call Invoke to call a function with resolved arguments or call Get to get a dependency instance:
```go
err = c.Invoke(func(someDep *SomeDep, someOtherDep *SomeOtherDep) {
  // do stuff
})
  
val, err := c.Get(reflect.TypeOf(&SomeOtherDep{}))
typedVal := val.(*SomeOtherDep)
```
## Scopes and lifetimes
Container supports the following dependency lifetimes:
* Singleton - instantiated once per main container
* Scoped - instantiated once per request
* Transient - instantiated once per Invoke or Get call

To take advantage of Scoped resolution, create a container in request scope:
```go
c = c.Scoped()
```
Such a container will cache Scoped dependencies and reuse them on Invoke and Get calls.

## Container context
Container allows parameterized instantiation of depencencies. To use container's context, call WithContext:
```go
c = c.WithContext("key", value)
```
To retrieve context parameters, pass a special type ContextParams as an argument in provider:
```go
err := c.Register(func(params di.ContextParams) *Logger {
	return logger.With("id", params.GetValue("id"))
}, di.Scoped)
```
