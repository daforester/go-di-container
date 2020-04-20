package di

import "reflect"

type TypeCheckerInterface interface {
	IsTypeCompatible(reflect.Type, reflect.Type, bool) bool
}

type TypeChecker struct{}

// Checks that b can be used as a
func (T TypeChecker) IsTypeCompatible(a reflect.Type, b reflect.Type, strict bool) bool {
	if a.Kind() == reflect.Interface {
		// Does b implement interface a
		return b.Implements(a)
	} else if a == b {
		// They're the same thing!
		return true
	}

	// If strict checking is disabled a pointer of type will match a concrete type
	if !strict {
		if a.Kind() == reflect.Ptr && b.Kind() == reflect.Struct && a.Elem().Kind() == b.Kind() {
			return true
		}
		if a.Kind() == reflect.Struct && b.Kind() == reflect.Ptr && a.Kind() == b.Elem().Kind() {
			return true
		}
	}

	return false
}
