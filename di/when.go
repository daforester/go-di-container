package di

import (
	"fmt"
	"reflect"
)

// whenLink is the first step of a contextual binding chain:
// app.When(&RequestingType{}).Needs((*Dependency)(nil)).Give(&Impl{})
type whenLink struct {
	a    *App
	when interface{}
}

// Needs specifies which dependency type to override for the requesting type.
func (w *whenLink) Needs(a interface{}) *needLink {
	return &needLink{
		w.a,
		w,
		a,
	}
}

// needLink is the second step of a contextual binding chain.
type needLink struct {
	a    *App
	when *whenLink
	need interface{}
}

// Give completes the contextual binding: when the requesting type needs the
// dependency, give it b instead of the default binding. Pass nil to remove.
func (n *needLink) Give(b interface{}) ObjectInterface {
	A := n.a
	A.lock()
	defer A.unlock()

	w := n.when.when
	a := n.need

	if w == nil {
		panic("When() requires a non-nil requesting type")
	}
	if a == nil {
		panic("Needs() requires a non-nil dependency type")
	}

	reflectW := reflect.TypeOf(w)
	reflectA := reflect.TypeOf(a)
	wKey := A.typeFullName(reflectW)
	aKey := A.typeFullName(reflectA)

	if b == nil {
		object := A.injectRegistry[wKey][aKey]
		delete(A.injectRegistry[wKey], aKey)
		return object
	}

	if !A.validBindCombination(a, b) && !A.validSingletonCombination(a, b) {
		panic(fmt.Sprintf("Can not assign %s to %s for %s", reflect.TypeOf(b), reflectA, reflectW))
	}

	if A.injectRegistry[wKey] == nil {
		A.injectRegistry[wKey] = make(map[string]ObjectInterface)
	}

	object := A.objectBuilder.New(b)

	A.injectRegistry[wKey][aKey] = object

	return object
}
