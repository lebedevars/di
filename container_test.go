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

func TestSingleton(t *testing.T) {
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

	c = c.RequestScoped()
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
	as.Equal(firstRetrieve, secondRetrieve)
}

func TestGet(t *testing.T) {
	as := assert.New(t)
	c := NewContainer()

	text := "I was resolved"
	err := c.Register(func() *example {
		return newExample(text)
	}, Singleton)
	as.NoError(err)

	err = c.Build()
	as.NoError(err)

	ex, err := c.Get(reflect.TypeOf(&example{}))
	as.NoError(err)
	as.IsType(&example{}, ex)
	newEx := ex.(*example)
	as.Equal(text, newEx.text)
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
	as.Errorf(err, notAFunctionError.Error())
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
	as.Errorf(err, notAFunctionError.Error())
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
