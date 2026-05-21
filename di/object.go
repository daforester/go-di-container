package di

import (
	"fmt"
	"reflect"
)

// Kind classifies what a bound Object wraps.
type Kind uint

const (
	Unknown   Kind = iota
	Func           // BindFunc factory
	Ptr            // pointer to a struct
	Redirect       // string alias to another binding
	Struct         // concrete struct value
	Primitive      // primitive value (int, string, etc.)
)

// ObjectInterface wraps a bound value with metadata about its kind and
// singleton status.
type ObjectInterface interface {
	New(interface{}, ...Kind) ObjectInterface
	Singleton() ObjectInterface
	IsSingleton() bool
}

// Object is the default ObjectInterface implementation.
type Object struct {
	Value     interface{}
	Name      string
	Kind      Kind
	singleton bool
}

func (o Object) New(v interface{}, k ...Kind) ObjectInterface {
	obj := new(Object)

	obj.Value = v

	vType := reflect.TypeOf(v)

	if len(k) > 0 {
		obj.Kind = k[0]
	} else {
		t := vType.Kind()
		switch t {
		case reflect.Func:
			obj.Kind = Func
		case reflect.Ptr:
			obj.Kind = Ptr
		case reflect.Struct:
			obj.Kind = Struct
		case reflect.String:
			obj.Kind = Redirect
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			obj.Kind = Primitive
		default:
			panic(fmt.Sprintf("Unsupported type: %s", t))
		}
	}

	if obj.Kind == Func {
		if !vType.ConvertibleTo(bindFuncType) {
			panic("Unsupported function type, must be compatible with BindFunc")
		}
	}

	obj.Name = typeFullName(vType)

	return obj
}

func (o *Object) Singleton() ObjectInterface {
	o.singleton = true
	return o
}

func (o *Object) IsSingleton() bool {
	return o.singleton
}

func (o *Object) String() string {
	return o.Name
}
