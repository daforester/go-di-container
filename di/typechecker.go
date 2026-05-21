package di

import "reflect"

// TypeCheckerInterface validates type compatibility for the container.
type TypeCheckerInterface interface {
	IsTypeCompatible(reflect.Type, reflect.Type, bool) bool
}

// TypeChecker is the default implementation of TypeCheckerInterface.
type TypeChecker struct{}

// IsTypeCompatible checks whether b can be used as a. In non-strict mode,
// a pointer and its underlying struct type are considered compatible.
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
		if a.Kind() == reflect.Ptr && b.Kind() == reflect.Struct && a.Elem() == b {
			return true
		}
		if a.Kind() == reflect.Struct && b.Kind() == reflect.Ptr && a == b.Elem() {
			return true
		}
	}

	return false
}
