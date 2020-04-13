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

func (n *needLink) Give(b interface{}) *App {
	A := n.a
	w := n.when.when
	a := n.need

	reflectW := reflect.TypeOf(w)
	reflectA := reflect.TypeOf(a)

	if b == nil {
		// Unset binding
		delete(A.injectRegistry[A.typeFullName(reflectW)], A.typeFullName(reflectA))
		return A
	}

	if !A.validBindCombination(a, b) && !A.validSingletonCombination(a, b) {
		panic(fmt.Sprintf("Can not assign %s to %s for %s", reflect.TypeOf(b), reflectA, reflectW))
	}

	if A.injectRegistry[A.typeFullName(reflectW)] == nil {
		A.injectRegistry[A.typeFullName(reflectW)] = make(map[string]ObjectInterface)
	}

	if A.typeFullName(reflectA)[0] == '/' && A.typeFullName(reflectA) == A.typeFullName(reflect.TypeOf(b)) {
		// If a primitive is expected, then inject a primitive
		A.injectRegistry[A.typeFullName(reflectW)][A.typeFullName(reflectA)] = A.objectBuilder.New(b, Primitive)
	} else {
		A.injectRegistry[A.typeFullName(reflectW)][A.typeFullName(reflectA)] = A.objectBuilder.New(b)
	}

	return A
}
