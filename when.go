package di

import (
	"fmt"
	"reflect"
)

type whenlink struct {
	a    *App
	when interface{}
}

func (w *whenlink) Needs(a interface{}) *needlink {
	return &needlink{
		w.a,
		w,
		a,
	}
}

type needlink struct {
	a    *App
	when *whenlink
	need interface{}
}

func (n *needlink) Give(b interface{}) *App {
	A := n.a
	w := n.when.when
	a := n.need

	reflectW := reflect.TypeOf(w)
	reflectA := reflect.TypeOf(a)

	if b == nil {
		// Unset binding
		delete(A.injectregistry[A.typeFullName(reflectW)], A.typeFullName(reflectA))
		return A
	}

	if !A.validBindCombination(a, b) && !A.validSingletonCombination(a, b) {
		panic(fmt.Sprintf("Can not assign %s to %s for %s", reflect.TypeOf(b), reflectA, reflectW))
	}

	if A.injectregistry[A.typeFullName(reflectW)] == nil {
		A.injectregistry[A.typeFullName(reflectW)] = make(map[string]ObjectInterface)
	}
	A.injectregistry[A.typeFullName(reflectW)][A.typeFullName(reflectA)] = A.objectBuilder.New(b)
	return A
}
