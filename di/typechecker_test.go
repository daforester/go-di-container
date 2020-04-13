package di

import (
	"fmt"
	"reflect"
	"testing"
)

type MockTypeChecker struct {
	TypeChecker
	overrideEnabled bool
	override        bool
}

func (T MockTypeChecker) IsTypeCompatible(a reflect.Type, b reflect.Type, strict bool) bool {
	if T.overrideEnabled {
		return T.override
	}

	return T.TypeChecker.IsTypeCompatible(a, b, strict)
}

func (T *MockTypeChecker) Override(override bool) *MockTypeChecker {
	T.override = override
	T.overrideEnabled = true
	return T
}

func (T *MockTypeChecker) DisableOverride() *MockTypeChecker {
	T.overrideEnabled = false
	return T
}

type teststructint interface {
	Compat()
}

type teststructa struct{}

func (t teststructa) Compat() {}

type teststructb struct{}

func TestTypeChecker_IsTypeCompatible(t *testing.T) {
	var b bool

	t1 := new(teststructa)
	t2 := teststructa{}
	t3 := new(teststructb)
	t4 := teststructb{}
	t5 := (*teststructint)(nil)

	aType := reflect.TypeOf(t1)
	bType := reflect.TypeOf(t2)
	cType := reflect.TypeOf(t3)
	dType := reflect.TypeOf(t4)
	eType := reflect.TypeOf(t5).Elem()

	b = TypeChecker{}.IsTypeCompatible(reflect.TypeOf(5), reflect.TypeOf("5"), false)
	if b {
		t.Error(fmt.Sprintf("Expected false comparing int & string"))
	}

	b = TypeChecker{}.IsTypeCompatible(aType, bType, false)
	if !b {
		t.Error(fmt.Sprintf("Expected true comparing ptr to struct of same type in non-strict mode"))
	}

	b = TypeChecker{}.IsTypeCompatible(aType, bType, true)
	if b {
		t.Error(fmt.Sprintf("Expected false comparing ptr to struct of same type in strict mode"))
	}

	b = TypeChecker{}.IsTypeCompatible(aType, cType, false)
	if b {
		t.Error(fmt.Sprintf("Expected false comparing two different ptrs"))
	}

	b = TypeChecker{}.IsTypeCompatible(bType, dType, false)
	if b {
		t.Error(fmt.Sprintf("Expected false comparing two different structs"))
	}

	b = TypeChecker{}.IsTypeCompatible(aType, aType, true)
	if !b {
		t.Error(fmt.Sprintf("Expected true comparing same ptr"))
	}

	b = TypeChecker{}.IsTypeCompatible(bType, bType, true)
	if !b {
		t.Error(fmt.Sprintf("Expected true comparing same struct"))
	}

	b = TypeChecker{}.IsTypeCompatible(eType, aType, true)
	if !b {
		t.Error(fmt.Sprintf("Expected true comparing ptr to interface"))
	}

	b = TypeChecker{}.IsTypeCompatible(eType, bType, true)
	if !b {
		t.Error(fmt.Sprintf("Expected true comparing struct to interface"))
	}
}
