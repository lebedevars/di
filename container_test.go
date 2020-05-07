package di

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type example struct {
	text string
}

func newExample(text string) *example {
	return &example{text: text}
}

type example2 struct {
	Example *example
}

func newExample2(ex *example) *example2 {
	return &example2{
		Example: ex,
	}
}

type example3 struct {
}

func newExample3() *example3 {
	return &example3{}
}

func TestSimple(t *testing.T) {
	wasInjected := "I was injected"
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func(ex *example) *example2 {
		return newExample2(ex)
	}, Transient)
	as.NoError(err)

	err = c.Register(func() *example {
		return newExample(wasInjected)
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	err = c.Invoke(func(ex *example, ex2 *example2) {
		as.Equal(wasInjected, ex.text)
		as.Equal(wasInjected, ex2.Example.text)
	})
	as.NoError(err)
}

func TestSingletonMainScope(t *testing.T) {
	singleton := newExample(time.Now().String())
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return singleton
	}, Singleton)
	as.NoError(err)

	err = c.Register(func(ex *example) *example2 {
		return newExample2(ex)
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	err = c.Invoke(func(ex *example, ex2 *example2) {
		as.Equal(singleton, ex)
		as.Equal(singleton, ex2.Example)
	})
	as.NoError(err)
}

func TestSingletonRequestScope(t *testing.T) {
	singleton := newExample(time.Now().String())
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return singleton
	}, Singleton)
	as.NoError(err)

	err = c.Register(func(ex *example) *example2 {
		return newExample2(ex)
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	c = c.Scoped()
	err = c.Invoke(func(ex *example, ex2 *example2) {
		as.Equal(singleton, ex)
		as.Equal(singleton, ex2.Example)
	})
	as.NoError(err)
}

func TestScopedMainScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return &example{text: time.Now().String()}
	}, Scoped)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	var firstRetrieve *example
	err = c.Invoke(func(ex *example) {
		firstRetrieve = ex
	})
	as.NoError(err)

	var secondRetrieve *example
	err = c.Invoke(func(ex *example) {
		secondRetrieve = ex
	})
	as.NoError(err)
	as.NotEqual(firstRetrieve, secondRetrieve)
}

func TestScopedRequestScope(t *testing.T) {
	type dependsOnScoped struct {
		Example2 *example2
	}

	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return &example{text: time.Now().String()}
	}, Scoped)
	as.NoError(err)
	err = c.Register(func(ex *example) *example2 {
		return newExample2(ex)
	}, Scoped)
	as.NoError(err)
	err = c.Register(func(ex2 *example2) *dependsOnScoped {
		return &dependsOnScoped{Example2: ex2}
	}, Scoped)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	c = c.Scoped()
	var firstRetrieve *example2
	err = c.Invoke(func(ex2 *example2) {
		firstRetrieve = ex2
	})
	as.NoError(err)

	var secondRetrieve *example2
	err = c.Invoke(func(ex2 *example2) {
		secondRetrieve = ex2
	})
	as.NoError(err)
	as.Equal(firstRetrieve, secondRetrieve)

	err = c.Invoke(func(dep *dependsOnScoped) {
		as.Equal(secondRetrieve, dep.Example2)
	})
}

func TestTransientMainScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return &example{text: time.Now().String()}
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	var firstRetrieve *example
	err = c.Invoke(func(ex *example) {
		firstRetrieve = ex
	})
	as.NoError(err)

	var secondRetrieve *example
	err = c.Invoke(func(ex *example) {
		secondRetrieve = ex
	})
	as.NoError(err)
	as.NotEqual(firstRetrieve, secondRetrieve)
}

func TestTransientRequestScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return &example{text: time.Now().String()}
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	c = c.Scoped()
	var firstRetrieve *example
	err = c.Invoke(func(ex *example) {
		firstRetrieve = ex
	})
	as.NoError(err)

	var secondRetrieve *example
	err = c.Invoke(func(ex *example) {
		secondRetrieve = ex
	})
	as.NoError(err)
	as.NotEqual(firstRetrieve, secondRetrieve)
}

func TestGetError(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Build()
	as.NoError(err)

	_, err = c.Get(reflect.TypeOf(&example{}))
	as.Error(err)
}

func TestGetSingletonMainScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample(time.Now().String())
	}, Singleton)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	firstRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.IsType(&example{}, firstRetrieve)
	secondRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.Equal(firstRetrieve.(*example), secondRetrieve.(*example))
}

func TestGetSingletonRequestScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample(time.Now().String())
	}, Singleton)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	c = c.Scoped()
	firstRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.IsType(&example{}, firstRetrieve)
	secondRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.Equal(firstRetrieve.(*example), secondRetrieve.(*example))
}

func TestGetScopedMainScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample(time.Now().String())
	}, Scoped)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	firstRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.IsType(&example{}, firstRetrieve)
	secondRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.NotEqual(firstRetrieve.(*example), secondRetrieve.(*example))
}

func TestGetScopedRequestScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample(time.Now().String())
	}, Scoped)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	c = c.Scoped()
	firstRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.IsType(&example{}, firstRetrieve)
	secondRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.Equal(firstRetrieve.(*example), secondRetrieve.(*example))
}

func TestGetTransientMainScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample(time.Now().String())
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	firstRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.IsType(&example{}, firstRetrieve)
	secondRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.NotEqual(firstRetrieve.(*example), secondRetrieve.(*example))
}

func TestGetTransientRequestScope(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample(time.Now().String())
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	c = c.Scoped()
	firstRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.IsType(&example{}, firstRetrieve)
	secondRetrieve, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.NotEqual(firstRetrieve.(*example), secondRetrieve.(*example))
}

func TestNoCachedSingleton(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample(time.Now().String())
	}, Singleton)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	// corrupt container
	c.singletonsCache = make(map[reflect.Type]reflect.Value)

	err = c.Invoke(func(ex *example) {})
	as.Error(err)
}

func TestNoLifetime(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample(time.Now().String())
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	// corrupt container
	c.lifetimes = make(map[reflect.Type]Lifetime)

	err = c.Invoke(func(ex *example) {})
	as.Error(err)
}

func TestWithContext(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()
	value := "I was injected from container's contextParams"
	err := c.Register(func(params ContextParams) *example {
		return newExample(params.GetValue("text").(string))
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	c = c.WithContext("text", value)
	err = c.Invoke(func(ex *example) {
		as.Equal(value, ex.text)
	})
	as.NoError(err)
}

func TestWithContextInheritContext(t *testing.T) {
	type InheritedContextExample struct {
		Inherited string
		New       string
	}

	as := assert.New(t)
	c := NewContainer()
	inherited := "I was injected from container's contextParams"
	newVal := "I, too, was injected from container's contextParams"
	err := c.Register(func(params ContextParams) *InheritedContextExample {
		return &InheritedContextExample{
			Inherited: params.GetValue("inherited").(string),
			New:       params.GetValue("new").(string),
		}
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	c = c.WithContext("inherited", inherited)
	c = c.WithContext("new", newVal)
	err = c.Invoke(func(ex *InheritedContextExample) {
		as.Equal(inherited, ex.Inherited)
		as.Equal(newVal, ex.New)
	})
	as.NoError(err)
}

func TestDoubleRegister(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample("")
	}, Transient)
	as.NoError(err)

	err = c.Register(func() *example {
		return newExample("")
	}, Transient)
	as.NotNil(err)
	as.True(strings.HasSuffix(err.Error(), "was already registered"))
}

func TestCyclicDependency(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func(ex3 *example3) *example {
		return newExample("")
	}, Transient)
	as.NoError(err)

	err = c.Register(func(ex *example) *example2 {
		return newExample2(ex)
	}, Transient)
	as.NoError(err)

	err = c.Register(func(ex *example2) *example3 {
		return newExample3()
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NotNil(err)
	as.True(strings.HasPrefix(err.Error(), "cyclic dependency detected"))
}

func TestUnregisteredDependency(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func(ex *example) *example2 {
		return newExample2(ex)
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NotNil(err)
	as.True(strings.HasSuffix(err.Error(), "was not registered"))
}

func TestInvokeUnregisteredDependency(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample("")
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	err = c.Invoke(func(ex *example, ex2 *example2) {
	})
	as.NotNil(err)
	as.True(strings.HasSuffix(err.Error(), "was not registered"))
}

func TestRegisterNotFunc(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(struct{}{}, Transient)
	as.Errorf(err, errNotAFunction.Error())
}

func TestRegisterInvalidOutParameterCount(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() (*example, string) {
		return newExample(""), "some-string"
	}, Transient)
	as.EqualError(err, errOnlyOneOutParam.Error())

	err = c.Register(func() {}, Transient)
	as.EqualError(err, errOnlyOneOutParam.Error())
}

func TestInvokeNotFunc(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample("")
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	err = c.Invoke(struct{}{})
	as.Errorf(err, errNotAFunction.Error())
}

func TestNonBuildContainer(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	err := c.Register(func() *example {
		return newExample("")
	}, Transient)
	as.NoError(err)

	err = c.Invoke(func(ex *example) {})
	as.EqualError(err, errMustBuildContainer.Error())

	_, err = c.Get(reflect.TypeOf(&example{}))
	as.EqualError(err, errMustBuildContainer.Error())
}

func BenchmarkResolve(b *testing.B) {
	as := assert.New(b)
	b.ReportAllocs()

	c := NewContainer()
	err := c.Register(func() *example {
		return newExample("I was injected")
	}, Transient)
	as.NoError(err)

	err = c.Register(func(ex *example) *example2 {
		return newExample2(ex)
	}, Transient)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	for i := 0; i < b.N; i++ {
		_ = c.Invoke(func(ex2 *example2) {
		})
	}
}
