package di

import (
	"fmt"
	"reflect"
)

type whenLink struct {
	a    *App
	when interface{}
}

func (w *whenLink) Needs(a interface{}) *needLink {
	return &needLink{
		w.a,
		w,
		a,
	}
}

type needLink struct {
	a    *App
	when *whenLink
	need interface{}
}

func (n *needLink) Give(b interface{}) ObjectInterface {
	A := n.a
	w := n.when.when
	a := n.need

	reflectW := reflect.TypeOf(w)
	reflectA := reflect.TypeOf(a)

	if b == nil {
		// Unset binding
		object := A.injectRegistry[A.typeFullName(reflectW)][A.typeFullName(reflectA)]
		delete(A.injectRegistry[A.typeFullName(reflectW)], A.typeFullName(reflectA))
		return object
	}

	if !A.validBindCombination(a, b) && !A.validSingletonCombination(a, b) {
		panic(fmt.Sprintf("Can not assign %s to %s for %s", reflect.TypeOf(b), reflectA, reflectW))
	}

	if A.injectRegistry[A.typeFullName(reflectW)] == nil {
		A.injectRegistry[A.typeFullName(reflectW)] = make(map[string]ObjectInterface)
	}

	var object ObjectInterface

	object = A.objectBuilder.New(b)

	A.injectRegistry[A.typeFullName(reflectW)][A.typeFullName(reflectA)] = object

	return object
}
