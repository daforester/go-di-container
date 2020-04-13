package di

import (
	"fmt"
	"testing"
)

type MockObject struct {
	Object
	overrideEnabled bool
	override        ObjectInterface
}

func (o MockObject) New(v interface{}, k ...Kind) ObjectInterface {
	if o.overrideEnabled {
		return o.override
	}

	return o.Object.New(v, k...)
}

func (o *MockObject) OverrideNew(override ObjectInterface) *MockObject {
	o.override = override
	o.overrideEnabled = true
	return o
}

func (o *MockObject) DisableOverride() *MockObject {
	o.overrideEnabled = false
	return o
}

type teststruct struct {
	X int    `inject:"123"`
	Y string `inject:"string"`
}

func TestObject_New(t *testing.T) {

}

func TestObject_New_ValidStruct(t *testing.T) {
	var o ObjectInterface

	o = Object{}.New(&struct {
		A string
		B int
	}{
		A: "A",
		B: 1,
	}, Struct)

	if o == nil {
		t.Error(fmt.Sprintf("Object expected in return to New"))
	}

	o = Object{}.New("foo")

	if o == nil {
		t.Error(fmt.Sprintf("Object expected in return to New"))
	}

	o = Object{}.New(&struct {
		A string
		B int
	}{
		A: "A",
		B: 1,
	})

	if o == nil {
		t.Error(fmt.Sprintf("Object expected in return to New"))
	}
}

func TestObject_New_ValidFunc(t *testing.T) {
	var z BindFunc = func(A *App) interface{} {
		return new(teststruct)
	}

	o := Object{}.New(z)

	if o == nil {
		t.Error("Object expected in response to New")
	}

	o = Object{}.New(func(A *App) interface{} {
		return A
	})

	if o == nil {
		t.Error("Object expected in response to New")
	}
}

func TestObject_New_InvalidFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Panic expected in response to invalid func")
		}
	}()

	Object{}.New(func() bool {
		return true
	})
}

func TestObject_New_InvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Panic expected in response to invalid type")
		}
	}()

	Object{}.New(1)
}

func TestObject_Singleton(t *testing.T) {
	o := Object{}.New(struct {
		A string
	}{
		A: "A",
	}).(*Object)

	if o.IsSingleton() != false {
		t.Error("Expected false from IsSingleton when object not set to singleton")
	}
	o.Singleton()
	if o.singleton != true {
		t.Error("Singleton method did not set object singleton property to true")
	}
	if o.IsSingleton() != true {
		t.Error("Expected true from IsSingleton when object is set to singleton")
	}
}

func TestObject_String(t *testing.T) {
	o := Object{}.New(struct {
		A string
	}{
		A: "A",
	}).(*Object)

	if o.String() != o.Name {
		t.Error("String method should return object name")
	}
}
