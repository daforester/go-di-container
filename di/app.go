// Package di provides a dependency injection / Inversion of Control (IoC)
// container for Go. It uses reflection to resolve dependencies via struct
// tags, constructor methods, and explicit bindings.
//
// All public methods on App are safe for concurrent use from multiple goroutines.
// Errors are reported via panic, not returned errors.
package di

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
)

// AppInterface defines the public contract for a DI container.
type AppInterface interface {
	New(...AppConfig) AppInterface
	Bind(interface{}, interface{}) AppInterface
	Singleton(interface{}, ...interface{}) AppInterface
	Make(interface{}) interface{}
	MakeWith(interface{}, map[string]interface{}) interface{}
	When(a interface{}) *whenLink
}

// BindFunc is a factory function that receives the container and returns
// an instance. Used with Bind and Singleton to construct dependencies.
type BindFunc func(*App) interface{}

// Package-level globals for the default and named container instances.
var (
	defaultApp AppInterface
	instances  map[string]AppInterface
	mu         sync.RWMutex
)

// App is the main DI container. It holds a binding registry, a contextual
// injection registry (When/Needs/Give), and uses a reentrant mutex so
// that BindFunc callbacks can safely call Make on the same container.
type App struct {
	objectBuilder  ObjectInterface
	typeChecker    TypeCheckerInterface
	registry       map[string]ObjectInterface
	injectRegistry map[string]map[string]ObjectInterface
	resolving      map[string]bool // circular dependency detection during resolution
	appMu          sync.Mutex
	lockOwner      int64 // goroutine ID of current lock holder (atomic)
	lockDepth      int32 // reentrant lock depth
}

// AppConfig provides options when creating a new container via New().
type AppConfig struct {
	Name          string
	ObjectBuilder ObjectInterface      // override for testing
	TypeChecker   TypeCheckerInterface // override for testing
	Default       bool
}

// goroutineID extracts the current goroutine's ID from runtime.Stack output.
func goroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	var id int64
	for i := len("goroutine "); i < n && buf[i] != ' '; i++ {
		id = id*10 + int64(buf[i]-'0')
	}
	return id
}

// lock acquires the container mutex. Reentrant: if the same goroutine already
// holds it (e.g. a BindFunc calling Make), the depth counter increments instead
// of deadlocking.
func (A *App) lock() {
	gid := goroutineID()
	if atomic.LoadInt64(&A.lockOwner) == gid {
		A.lockDepth++
		return
	}
	A.appMu.Lock()
	atomic.StoreInt64(&A.lockOwner, gid)
	A.lockDepth = 1
}

func (A *App) unlock() {
	A.lockDepth--
	if A.lockDepth == 0 {
		atomic.StoreInt64(&A.lockOwner, 0)
		A.appMu.Unlock()
	}
}

// New creates a new App container instance with optional config.
func New(config ...AppConfig) *App {
	return (&App{}).New(config...).(*App)
}

// Default returns the default or a named app instance, creating one if it doesn't exist.
func Default(name ...string) AppInterface {
	if len(name) > 0 && len(name[0]) > 0 {
		mu.RLock()
		m := instances[name[0]]
		mu.RUnlock()
		if m == nil {
			mu.Lock()
			defer mu.Unlock()
			if instances[name[0]] != nil {
				return instances[name[0]]
			}
			if instances == nil {
				instances = make(map[string]AppInterface)
			}
			instances[name[0]] = newAppInstance()
			return instances[name[0]]
		}

		return m
	}

	mu.RLock()
	if defaultApp != nil {
		a := defaultApp
		mu.RUnlock()
		return a
	}
	mu.RUnlock()

	mu.Lock()
	defer mu.Unlock()
	if defaultApp != nil {
		return defaultApp
	}
	defaultApp = newAppInstance()
	return defaultApp
}

func newAppInstance() *App {
	a := new(App)
	a.objectBuilder = new(Object)
	a.typeChecker = new(TypeChecker)
	a.registry = make(map[string]ObjectInterface)
	a.injectRegistry = make(map[string]map[string]ObjectInterface)
	a.resolving = make(map[string]bool)
	return a
}

// New creates a new container instance, optionally configured via AppConfig.
func (A *App) New(config ...AppConfig) AppInterface {
	a := newAppInstance()

	mu.Lock()
	if defaultApp == nil {
		defaultApp = a
	}

	// Process config options - allows providing mocked objectBuilder & typeChecker for example
	if len(config) > 0 {
		c := config[0]
		if len(c.Name) > 0 {
			if instances == nil {
				instances = make(map[string]AppInterface)
			}

			instances[c.Name] = a
		}
		if c.Default {
			defaultApp = a
		}
		if c.ObjectBuilder != nil {
			a.objectBuilder = c.ObjectBuilder
		}
		if c.TypeChecker != nil {
			a.typeChecker = c.TypeChecker
		}
	}
	mu.Unlock()

	return a
}

/*
All the combinations of input that can be accepted

Bind
String, String - Alias
Interface, String - Alias
Struct, String - Alias
Pointer, String - Alias

String, Struct
String, Pointer
String, Func
String, Interface
String, Primitive (int, bool, float, etc.)

Interface, Struct
Interface, Pointer
Interface, Func

Struct, Func
Pointer, Func

Singleton
Interface, Pointer - Singleton
Interface, Func - Singleton
Pointer, Func - Singleton
Pointer, Pointer - Singleton (same type)
Pointer - Singleton
*/

// Bind registers implementation b for type a. Pass nil as b to remove a binding.
func (A *App) Bind(a interface{}, b interface{}) AppInterface {
	A.lock()
	defer A.unlock()

	var o ObjectInterface
	var label string
	var aType reflect.Type
	var bType reflect.Type

	if b == nil {
		// Unset binding
		A.deleteRegistryEntry(a)
		return A
	}

	// Check that a & b are compatible binding
	if !A.validBindCombination(a, b) {
		if a != nil {
			aType = reflect.TypeOf(a)
		}

		bType = reflect.TypeOf(b)

		panic(fmt.Sprintf("Unsupported input, cannot bind %s to %s", bType, aType))
	}

	aType = reflect.TypeOf(a)
	bType = reflect.TypeOf(b)

	realAType := A.resolveTypePtr(aType)

	if bType.Kind() == reflect.String {
		// Create redirect
		if aType.Kind() == reflect.String {
			label = a.(string)
		} else {
			label = A.typeFullName(aType)
		}

		o = A.objectBuilder.New(b, Redirect)
	} else if realAType.Kind() == reflect.String {
		// Custom binding
		label = a.(string)
		if A.resolveTypePtr(bType).Kind() == reflect.Interface {
			o = A.objectBuilder.New(A.typeFullName(bType))
		} else {
			o = A.objectBuilder.New(b)
		}
	} else if realAType.Kind() == reflect.Interface || bType.Kind() == reflect.Func {
		// Automatic naming
		label = A.typeFullName(aType)
		o = A.objectBuilder.New(b)
	}

	if o == nil {
		// Should never get here, above checks should catch everything
		panic(fmt.Sprintf("Unexpected error occurred, object not defined, inputs valid but didn't create object. Asked to bind %s to %s", bType, aType))
	}

	// Bind label to object
	A.registry[label] = o

	return A
}

func (A *App) deleteRegistryEntry(a interface{}) bool {
	if a != nil {
		// Unset binding
		var label string
		aType := reflect.TypeOf(a)

		if aType.Kind() == reflect.String {
			label = a.(string)
		} else {
			label = A.typeFullName(aType)
		}

		if _, e := A.registry[label]; e {
			delete(A.registry, label)
			return true
		}
	}

	return false
}

// Defines all the valid bind combinations
func (A *App) validBindCombination(a interface{}, b interface{}) (result bool) {
	// Catch panics generated anywhere as indication type is not valid
	defer func() {
		if r := recover(); r != nil {
			result = false
		}
	}()

	if a == nil {
		// Must bind to something
		panic(fmt.Sprintf("Cannot bind to nil value, use format \"(*Interface)(nil)\" for Interfaces"))
	}

	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)
	realAType := A.resolveTypePtr(aType)

	if realAType.Kind() == reflect.String || bType.Kind() == reflect.String {
		return true
	}
	if realAType.Kind() == reflect.Interface {
		if bType.Kind() == reflect.Func {
			// Interface binding to Func, must be a BindFunc and return a compatible type with a
			if A.isFuncSignatureCompatible(b, realAType) {
				return true
			}
		}
		if bType.Kind() == reflect.Ptr && bType.Implements(realAType) {
			// Interface binding to Ptr
			return true
		}
		if bType.Kind() == reflect.Struct && bType.Implements(realAType) {
			// Interface binding to Struct
			return true
		}
	}
	if (aType.Kind() == reflect.Ptr || aType.Kind() == reflect.Struct) && bType.Kind() == reflect.Func {
		// Bind Ptr & Struct to a BindFunc to run as a constructor
		if A.isFuncSignatureCompatible(b, aType) {
			return true
		}
	}

	return false
}

// Singleton registers a shared instance for type a. Like Bind, but always returns the same instance.
func (A *App) Singleton(a interface{}, c ...interface{}) AppInterface {
	A.lock()
	defer A.unlock()

	var o ObjectInterface
	var aType reflect.Type
	var bType reflect.Type

	if len(c) == 1 && c[0] == nil {
		// Unset binding
		A.deleteRegistryEntry(a)
		return A
	}

	// Check that a & optional b are compatible binding
	if !A.validSingletonCombination(a, c...) {
		if a != nil {
			aType = reflect.TypeOf(a)
		}
		if len(c) >= 1 {
			bType = reflect.TypeOf(c[0])
		}
		panic(fmt.Sprintf("Unsupported input, cannot bind singleton %s to %s", bType, aType))
	}

	aType = reflect.TypeOf(a)
	label := A.typeFullName(aType)

	if len(c) == 0 {
		// Must be a struct or ptr to struct
		if A.resolveTypePtr(aType).Kind() != reflect.Struct {
			panic(fmt.Sprintf("Unsupported input to singleton %s", aType))
		}

		if reflect.ValueOf(a).IsNil() {
			a = A.makeInternal(a)
		}

		o = A.objectBuilder.New(a)
	} else if len(c) > 1 {
		panic("Too many parameters passed to singleton expected 1 or 2")
	} else {
		b := c[0]
		bType = reflect.TypeOf(b)

		// Obtain result of BindFunc and bind to that
		if bType.Kind() == reflect.Func {
			bf := A.interfaceToBindFunc(b)
			b = bf(A)
			realAType := A.resolveTypePtr(aType)
			if !A.typeChecker.IsTypeCompatible(realAType, reflect.TypeOf(b), false) {
				panic(fmt.Sprintf("Singleton BindFunc returned %s which is not compatible with %s", reflect.TypeOf(b), aType))
			}
		} else if isNilableKind(bType.Kind()) && reflect.ValueOf(b).IsNil() {
			b = A.makeInternal(b)
		}

		o = A.objectBuilder.New(b)
	}

	if o == nil {
		panic(fmt.Sprintf("Unexpected error occurred, object not defined, inputs valid but didn't create object. Asked to bind %s to %s", bType, aType))
	}

	o.Singleton()

	A.registry[label] = o

	return A
}

// Defines valid singleton combinations
func (A *App) validSingletonCombination(a interface{}, c ...interface{}) (result bool) {
	// Any panics mean type isn't valid
	defer func() {
		if r := recover(); r != nil {
			result = false
		}
	}()

	aType := reflect.TypeOf(a)
	realAType := A.resolveTypePtr(aType)

	if len(c) == 0 {
		// Defined instance
		return aType.Kind() == reflect.Ptr
	}

	b := c[0]
	bType := reflect.TypeOf(b)

	if realAType.Kind() == reflect.Interface {
		if bType.Kind() == reflect.Func {
			// Interface to func with valid return type
			if A.isFuncSignatureCompatible(b, realAType) {
				return true
			}
		}
		if bType.Kind() == reflect.Ptr && bType.Implements(realAType) {
			// Interface to ptr
			return true
		}
	}
	if aType.Kind() == reflect.Ptr && bType.Kind() == reflect.Func {
		// Pointer uses function for construction
		if A.isFuncSignatureCompatible(b, aType) {
			return true
		}
	}
	if aType.Kind() == reflect.Ptr && bType.Kind() == reflect.Ptr && A.typeFullName(aType) == A.typeFullName(bType) {
		return true
	}

	return false
}

func (A *App) When(a interface{}) *whenLink {
	return &whenLink{
		A,
		a,
	}
}

// Make resolves and returns an instance of type a. Uses the binding registry,
// constructor methods (New), and struct tags (inject/di) to build the result.
// Panics if a required interface or string binding is not found.
func (A *App) Make(a interface{}) interface{} {
	A.lock()
	defer A.unlock()
	return A.makeInternal(a)
}

func (A *App) makeInternal(a interface{}) interface{} {
	return A.makeWithInternal(a, make(map[string]interface{}))
}

// MakeWith resolves type a with per-call field overrides. Map keys are field names.
func (A *App) MakeWith(a interface{}, injectables map[string]interface{}) interface{} {
	A.lock()
	defer A.unlock()
	return A.makeWithInternal(a, injectables)
}

func (A *App) makeWithInternal(a interface{}, injectables map[string]interface{}) interface{} {
	var x *Object
	var e bool

	t := reflect.TypeOf(a)
	if t == nil {
		panic("Make() requires a non-nil type")
	}

	var resolveKey string
	if t.Kind() == reflect.String {
		resolveKey = a.(string)
	} else {
		resolveKey = A.typeFullName(t)
	}

	if A.resolving[resolveKey] {
		panic(fmt.Sprintf("circular dependency detected while resolving %s", resolveKey))
	}
	A.resolving[resolveKey] = true
	defer delete(A.resolving, resolveKey)

	x, e = A.registry[resolveKey].(*Object)

	if e {
		result := A.processObject(x, injectables)
		if t.Kind() != reflect.String {
			rType := reflect.TypeOf(result)
			targetType := A.resolveTypePtr(t)
			if !A.typeChecker.IsTypeCompatible(targetType, rType, false) {
				panic(fmt.Sprintf("Made type %s is not compatible with requested type %s", rType, t))
			}
		}
		return result
	}

	if t.Kind() == reflect.String {
		panic(fmt.Sprintf("no binding found for %s", a))
	}
	if A.resolveTypePtr(t).Kind() == reflect.Interface {
		panic(fmt.Sprintf("no binding found for %s", t))
	}

	return A.autogen(a, injectables)
}

// processObject dispatches on the Object's Kind: follows redirects, returns
// singletons, runs BindFuncs, or auto-generates structs/pointers.
func (A *App) processObject(x *Object, injectables map[string]interface{}) interface{} {
	if x.Kind == Redirect {
		// Follow the redirect
		return A.makeWithInternal(x.Value, injectables)
	}
	if x.IsSingleton() {
		return x.Value
	}

	if x.Kind == Func {
		// Run the BindFunc
		bf := A.interfaceToBindFunc(x.Value)
		result := bf(A)
		return A.processStructTags(result, injectables)
	} else if x.Kind == Struct || x.Kind == Ptr {
		return A.autogen(x.Value, injectables)
	} else if x.Kind == Primitive {
		return x.Value
	}

	// Unknown type, shouldn't trigger
	panic(fmt.Sprintf("Unsupported Type %d", x.Kind))
}

// autogen resolves a type that has no explicit binding, using either a New()
// constructor method or struct tag hints.
func (A *App) autogen(a interface{}, injectables map[string]interface{}) interface{} {
	t := reflect.TypeOf(a)
	_, e := t.MethodByName("New")

	if e {
		return A.makeByNew(a, injectables)
	}
	return A.makeByHints(a, injectables)
}

// makeByHints builds an instance using struct tag hints (inject/di) and the
// When/Needs/Give contextual registry. Used when the type has no New() method.
func (A *App) makeByHints(a interface{}, injectables map[string]interface{}) interface{} {
	ot := reflect.TypeOf(a)
	t := resolveTypePtr(ot)

	// Create a new instance of object to work with, preserving any existing field values
	newobj := reflect.New(t)
	origVal := reflect.ValueOf(a)
	if origVal.Kind() == reflect.Ptr {
		if !origVal.IsNil() {
			newobj.Elem().Set(origVal.Elem())
		}
	} else if origVal.Kind() == reflect.Struct {
		newobj.Elem().Set(origVal)
	}

	// Use injection registry - if x needs y give z
	hintmap, hasmap := A.injectRegistry[A.typeFullName(ot)]

	// Iterate over the fields of the struct
	for fn := 0; fn < t.NumField(); fn++ {
		f := t.Field(fn)
		// Obtain tag inject values
		injectValue, inject := f.Tag.Lookup("inject")
		_, di := f.Tag.Lookup("di")
		newField := newobj.Elem().Field(fn)
		if newField.CanSet() {
			if di && inject && injectValue != "" && isPrimitiveKind(f.Type.Kind()) {
				A.setByTagValue(f.Type.Kind(), newField, injectValue)
			} else if di {
				containerVal := reflect.ValueOf(A)
				if containerVal.Type().AssignableTo(f.Type) {
					newField.Set(containerVal)
				}
			} else if inject {
				var po ObjectInterface
				var poe bool
				if hasmap {
					// If preset map exists, see if a mapping was configured for a type
					if f.Type.Kind() == reflect.Interface {
						// Ensure correct naming for interface
						pPtr := reflect.New(f.Type)
						po, poe = hintmap[A.typeFullName(pPtr.Type())]
					} else {
						po, poe = hintmap[A.typeFullName(f.Type)]
					}
				}
				pv, pe := injectables[f.Name]
				if pe && pv != nil && reflect.ValueOf(pv).Type().AssignableTo(newField.Type()) {
					// Value for this field was provided in MakeWith
					newField.Set(reflect.ValueOf(pv))
				} else if poe {
					c := A.processObject(po.(*Object), make(map[string]interface{}))
					newField.Set(reflect.ValueOf(c))
				} else if injectValue != "" {
					// Inject value provided
					A.setByTagValue(f.Type.Kind(), newField, injectValue)
				} else {
					// Call Make on compatible field types
					var c interface{}

					if f.Type.Kind() == reflect.Ptr {
						pPtr := reflect.New(f.Type.Elem())
						c = A.makeInternal(pPtr.Interface())
					} else if f.Type.Kind() == reflect.Struct {
						pPtr := reflect.New(f.Type)
						c = A.makeInternal(pPtr.Elem().Interface())
					} else if f.Type.Kind() == reflect.Interface {
						pPtr := reflect.New(f.Type)
						c = A.makeInternal(pPtr.Interface())
					} else if isPrimitiveKind(f.Type.Kind()) {
						panic(fmt.Sprintf("Value must be specified when injecting %s", f.Type.Kind()))
					}

					if c == nil {
						panic(fmt.Sprintf("Could not inject %s (%s)", f.Type, f.Type.Kind()))
					}

					newField.Set(reflect.ValueOf(c))
				}
			}
		}
	}

	// Convert Ptr to Struct if requested
	if ot.Kind() == reflect.Struct {
		return newobj.Elem().Interface()
	}

	return newobj.Interface()
}

// makeByNew calls the type's New() constructor method with auto-resolved
// parameters, then runs processStructTags on the result for inject/di tags.
func (A *App) makeByNew(a interface{}, injectables map[string]interface{}) interface{} {
	t := reflect.TypeOf(a)
	valIn := reflect.ValueOf(a)

	// We can't call "New" on a nil ptr so create a new instance of the type to work with
	if t.Kind() == reflect.Ptr && valIn.IsNil() {
		newobj := reflect.New(resolveTypePtr(t))
		a = newobj.Interface()
		valIn = reflect.ValueOf(a)
	}

	// Obtain preset injection map for object
	hintmap, hasmap := A.injectRegistry[A.typeFullName(t)]

	method, _ := t.MethodByName("New")

	injects := []reflect.Value{valIn}

	// Iterate over the function parameters
	for v := 1; v < method.Type.NumIn(); v++ {
		var c interface{}

		childType := method.Type.In(v)

		var pPtr reflect.Value

		if childType.Kind() == reflect.Ptr {
			pPtr = reflect.New(childType.Elem())
		} else {
			pPtr = reflect.New(childType)
		}

		var po ObjectInterface
		var poe bool
		if hasmap {
			// If preset map exists, see if a mapping was configured for a type
			if childType.Kind() == reflect.Interface {
				// Ensure correct naming for interface
				po, poe = hintmap[A.typeFullName(pPtr.Type())]
			} else {
				po, poe = hintmap[A.typeFullName(childType)]
			}
		}
		if poe {
			c = A.processObject(po.(*Object), make(map[string]interface{}))
		} else if childType.Kind() == reflect.Ptr {
			c = A.makeInternal(pPtr.Interface())
		} else if childType.Kind() == reflect.Interface {
			c = A.makeInternal(A.typeFullName(pPtr.Type()))
		} else if childType.Kind() == reflect.Struct {
			c = A.makeInternal(pPtr.Elem().Interface())
		}

		if c == nil {
			// Can not inject this type
			panic(fmt.Sprintf("Could not inject %s", childType))
		}

		// Build up list of parameters to call
		injects = append(injects, reflect.ValueOf(c))
	}

	y := method.Func.Call(injects)

	if len(y) == 0 {
		panic(fmt.Sprintf("Failed creating new %s", t))
	}

	if !A.typeChecker.IsTypeCompatible(t, y[0].Type(), false) {
		panic(fmt.Sprintf("Return type of New %s does not match requested type %s", y[0].Kind(), t.Kind()))
	}

	// Need to ensure return from new matches requested type, convert struct -> ptr, ptr -> struct
	var result interface{}
	if t.Kind() == y[0].Kind() {
		result = y[0].Interface()
	} else if t.Kind() == reflect.Struct && y[0].Kind() == reflect.Ptr {
		result = y[0].Elem().Interface()
	} else if t.Kind() == reflect.Ptr && y[0].Kind() == reflect.Struct {
		z := reflect.New(y[0].Type())
		z.Elem().Set(y[0])
		result = z.Interface()
	} else {
		panic(fmt.Sprintf("Unexpected error occurred, type of New %s does not match requested type %s", y[0].Kind(), t.Kind()))
	}

	// Process di and inject tags on the result
	result = A.processStructTags(result, injectables)

	return result
}

// processStructTags runs after a New() constructor and applies inject/di struct
// tags. Inject tags always overwrite; di tags only inject if the field is zero
// (so values set by New() are preserved). Also consults the When/Needs/Give
// registry for contextual overrides.
func (A *App) processStructTags(a interface{}, injectables map[string]interface{}) interface{} {
	ot := reflect.TypeOf(a)
	if ot == nil {
		return a
	}

	t := resolveTypePtr(ot)

	if t.Kind() != reflect.Struct {
		return a
	}

	var val reflect.Value
	switch ot.Kind() {
	case reflect.Ptr:
		val = reflect.ValueOf(a).Elem()
	default:
		v := reflect.New(t)
		v.Elem().Set(reflect.ValueOf(a))
		val = v.Elem()
	}

	hintmap, hasmap := A.injectRegistry[A.typeFullName(ot)]

	for fn := 0; fn < t.NumField(); fn++ {
		f := t.Field(fn)
		fieldVal := val.Field(fn)
		if !fieldVal.CanSet() {
			continue
		}

		_, di := f.Tag.Lookup("di")
		injectValue, inject := f.Tag.Lookup("inject")

		if di && inject && injectValue != "" && isPrimitiveKind(f.Type.Kind()) {
			A.setByTagValue(f.Type.Kind(), fieldVal, injectValue)
		} else if di && fieldVal.IsZero() {
			containerVal := reflect.ValueOf(A)
			if containerVal.Type().AssignableTo(f.Type) {
				fieldVal.Set(containerVal)
			}
		} else if inject {
			var po ObjectInterface
			var poe bool
			if hasmap {
				if f.Type.Kind() == reflect.Interface {
					pPtr := reflect.New(f.Type)
					po, poe = hintmap[A.typeFullName(pPtr.Type())]
				} else {
					po, poe = hintmap[A.typeFullName(f.Type)]
				}
			}
			pv, pe := injectables[f.Name]
			if pe && pv != nil && reflect.ValueOf(pv).Type().AssignableTo(fieldVal.Type()) {
				fieldVal.Set(reflect.ValueOf(pv))
			} else if poe {
				c := A.processObject(po.(*Object), make(map[string]interface{}))
				fieldVal.Set(reflect.ValueOf(c))
			} else if injectValue != "" {
				A.setByTagValue(f.Type.Kind(), fieldVal, injectValue)
			} else if f.Type.Kind() == reflect.Ptr || f.Type.Kind() == reflect.Struct || f.Type.Kind() == reflect.Interface {
				var pPtr reflect.Value
				if f.Type.Kind() == reflect.Ptr {
					pPtr = reflect.New(f.Type.Elem())
				} else {
					pPtr = reflect.New(f.Type)
				}
				c := A.makeInternal(pPtr.Interface())
				if c != nil {
					fieldVal.Set(reflect.ValueOf(c))
				}
			} else if isPrimitiveKind(f.Type.Kind()) {
				panic(fmt.Sprintf("Value must be specified when injecting %s", f.Type.Kind()))
			}
		}
	}

	if ot.Kind() == reflect.Struct {
		return val.Interface()
	}
	return val.Addr().Interface()
}

func (A *App) setByTagValue(k reflect.Kind, f reflect.Value, v string) {
	// Parsing for inject values on primitives
	switch k {
	case reflect.String:
		f.Set(reflect.ValueOf(v))
		return
	case reflect.Bool:
		iv, err := strconv.ParseBool(v)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(iv))
		return
	case reflect.Float32:
		iv, err := strconv.ParseFloat(v, 32)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(float32(iv)))
		return
	case reflect.Float64:
		iv, err := strconv.ParseFloat(v, 64)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(iv))
		return
	case reflect.Int:
		iv, err := strconv.Atoi(v)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(iv))
		return
	case reflect.Int8:
		iv, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(int8(iv)))
		return
	case reflect.Int16:
		iv, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(int16(iv)))
		return
	case reflect.Int32:
		iv, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(int32(iv)))
		return
	case reflect.Int64:
		iv, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(iv))
		return
	case reflect.Uint:
		iv, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(uint(iv)))
		return
	case reflect.Uint8:
		iv, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(uint8(iv)))
		return
	case reflect.Uint16:
		iv, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(uint16(iv)))
		return
	case reflect.Uint32:
		iv, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(uint32(iv)))
		return
	case reflect.Uint64:
		iv, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			panic(err)
		}
		f.Set(reflect.ValueOf(iv))
		return
	}

	// Can not handle parsing for this type
	panic(fmt.Sprintf("can not initialize value for kind %s", k))
}

func isIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

func isPrimitiveKind(k reflect.Kind) bool {
	switch k {
	case reflect.String, reflect.Bool, reflect.Float32, reflect.Float64:
		return true
	}
	return isIntKind(k)
}

func isNilableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return true
	}
	return false
}

func (A *App) typeFullName(t reflect.Type) string {
	return typeFullName(t)
}

func (A *App) resolveTypePtr(t reflect.Type) reflect.Type {
	return resolveTypePtr(t)
}

func typeFullName(t reflect.Type) string {
	pt := resolveTypePtr(t)
	return pt.PkgPath() + "/" + t.String()
}

func resolveTypePtr(t reflect.Type) reflect.Type {
	for k := t.Kind(); k == reflect.Ptr; {
		t = t.Elem()
		k = t.Kind()
	}
	return t
}

// Converts an interface into a BindFunc
func (A *App) interfaceToBindFunc(a interface{}) BindFunc {
	aType := reflect.TypeOf(a)
	if !aType.ConvertibleTo(bindFuncType) {
		panic("Unsupported function type, must be compatible with BindFunc")
	}
	return reflect.ValueOf(a).Convert(bindFuncType).Interface().(BindFunc)
}

var (
	emptyInterfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
	appPtrType         = reflect.TypeOf((*App)(nil))
	bindFuncType       = reflect.TypeOf(BindFunc(nil))
)

// Checks if a BindFunc's signature is compatible (takes *App, returns one compatible value)
func (A *App) isFuncSignatureCompatible(b interface{}, t reflect.Type) bool {
	bType := reflect.TypeOf(b)
	if bType.Kind() != reflect.Func {
		return false
	}
	if bType.NumIn() != 1 || bType.In(0) != appPtrType {
		return false
	}
	if bType.NumOut() != 1 {
		return false
	}
	outType := bType.Out(0)
	if outType != emptyInterfaceType {
		if !A.typeChecker.IsTypeCompatible(t, outType, false) {
			return false
		}
	}
	return true
}
