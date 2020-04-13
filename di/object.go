package di

import (
	"fmt"
	"reflect"
)

type Kind uint

const (
	Unknown = iota
	Func
	Ptr
	Redirect
	Struct
	Primitive
)

type ObjectInterface interface {
	New(interface{}, ...Kind) ObjectInterface
	Singleton() ObjectInterface
	IsSingleton() bool
}

type Object struct {
	Value        interface{}
	Name         string
	Kind         Kind
	singleton    bool
	reflectType  reflect.Type
	reflectValue reflect.Value
}

func (o Object) New(v interface{}, k ...Kind) ObjectInterface {
	obj := new(Object)

	obj.Value = v

	obj.reflectType = reflect.TypeOf(v)
	obj.reflectValue = reflect.ValueOf(v)

	if len(k) > 0 {
		obj.Kind = k[0]
	} else {
		t := obj.reflectType.Kind()
		switch t {
		case reflect.Func:
			obj.Kind = Func
		case reflect.Ptr:
			obj.Kind = Ptr
		case reflect.Struct:
			obj.Kind = Struct
		case reflect.String:
			obj.Kind = Redirect
		default:
			panic(fmt.Sprintf("Unsupported type: %s", t))
		}
	}

	if obj.Kind == Func {
		var x BindFunc
		if !reflect.TypeOf(obj.Value).ConvertibleTo(reflect.TypeOf(x)) {
			panic("Unsupported function type, must be compatible with BindFunc")
		}
	}

	obj.Name = obj.typeFullName(obj.reflectType)

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

func (o *Object) typeFullName(t reflect.Type) string {
	pt := o.resolveTypePtr(t)
	return pt.PkgPath() + "/" + t.String()
}

func (o *Object) resolveTypePtr(t reflect.Type) reflect.Type {
	for k := t.Kind(); k == reflect.Ptr; {
		t = t.Elem()
		k = t.Kind()
	}
	return t
}
