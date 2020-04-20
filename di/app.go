package di

import (
	"fmt"
	"reflect"
	"strconv"
)

type AppInterface interface {
	New(...AppConfig) AppInterface                            // Creates new DI App Instance
	Bind(interface{}, interface{}) AppInterface               // Binds an implementation b to type a, b is always a new instance
	Singleton(interface{}, ...interface{}) AppInterface       // Binds an existing instance b (or a if not specified) to type a
	Make(interface{}) interface{}                             // Returns a new instance (or existing if singleton of specified type a)
	MakeWith(interface{}, map[string]interface{}) interface{} // Returns a new instance (or existing if singleton of specified type a) but with specified injections in map
	When(a interface{}) *whenLink                             // Sets up specific binding for a thing so that when a needs a b it gets c
}

type BindFunc func(*App) interface{}

var (
	defaultApp AppInterface
	instances  map[string]AppInterface
)

type App struct {
	objectBuilder  ObjectInterface
	typeChecker    TypeCheckerInterface
	registry       map[string]ObjectInterface            // Defined Bindings
	injectRegistry map[string]map[string]ObjectInterface // Defined Injections for When
}

type AppConfig struct {
	Name          string
	ObjectBuilder ObjectInterface      // Allows for Mock Implementations
	TypeChecker   TypeCheckerInterface // Allows for Mocked Implementations
	Default       bool
}

// Create new App Container Instance with optional config
func New(config ...AppConfig) *App {
	return App{}.New(config...).(*App)
}

// Obtain default or named app instance
func Default(name ...string) AppInterface {
	if len(name) > 0 && len(name[0]) > 0 {
		m := instances[name[0]]
		if m == nil {
			// Named instance not found so create it
			instances[name[0]] = New(AppConfig{name[0], nil, nil, false})
			return instances[name[0]]
		}

		return m
	}

	if defaultApp == nil {
		// Default instance not found so create it
		defaultApp = New(AppConfig{"", nil, nil, true})
		return defaultApp
	}

	return defaultApp
}

// Creates new app instance
func (A App) New(config ...AppConfig) AppInterface {
	a := new(App)
	a.objectBuilder = new(Object)
	a.typeChecker = new(TypeChecker)
	a.registry = make(map[string]ObjectInterface)
	a.injectRegistry = make(map[string]map[string]ObjectInterface)

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

Interface, Struct
Interface, Pointer
Interface, Func

Struct, Func
Pointer, Func

Singleton
Interface, Pointer - Singleton
Interface, Func - Singleton
Pointer, Func - Singleton
Pointer - Singleton
*/

// Binds type b to type a
func (A *App) Bind(a interface{}, b interface{}) AppInterface {
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
			bf := A.interfaceToBindFunc(b)
			if v, _, _ := A.isFuncReturnCompatible(bf, realAType); v {
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
		bf := A.interfaceToBindFunc(b)
		if v, _, _ := A.isFuncReturnCompatible(bf, aType); v {
			return true
		}
	}

	return false
}

// Similar to Bind but ensures that existing instance is always used, c is optional
func (A *App) Singleton(a interface{}, c ...interface{}) AppInterface {
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

		o = A.objectBuilder.New(a)
	} else if len(c) > 1 {
		panic("Too many parameters passed to singleton expected 1 or 2")
	} else {
		b := c[0]
		bType = reflect.TypeOf(b)

		// Obtain result of BindFund and bind to that
		if bType.Kind() == reflect.Func {
			bf := A.interfaceToBindFunc(b)
			b = bf(A)
		} else if reflect.ValueOf(c[0]).IsNil() {
			b = A.Make(c[0])
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
			bf := A.interfaceToBindFunc(b)
			if v, _, _ := A.isFuncReturnCompatible(bf, realAType); v {
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
		bf := A.interfaceToBindFunc(b)
		if v, _, _ := A.isFuncReturnCompatible(bf, aType); v {
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

func (A *App) Make(a interface{}) interface{} {
	// Make with no fields provided
	i := make(map[string]interface{})
	return A.MakeWith(a, i)
}

// Make type A with optional injectables
func (A *App) MakeWith(a interface{}, injectables map[string]interface{}) interface{} {
	// Make with fields provided
	var x *Object
	var e bool

	t := reflect.TypeOf(a)

	if t.Kind() == reflect.String {
		// Obtain object by specified name
		x, e = A.registry[a.(string)].(*Object)
	} else {
		// Obtain object by types full path & name
		x, e = A.registry[A.typeFullName(t)].(*Object)
	}

	if e {
		return A.processObject(x, injectables)
	}

	// Binding must exist if requesting interface or string
	if t.Kind() == reflect.String {
		panic(fmt.Sprintf("no binding found for %s", a))
	}
	if A.resolveTypePtr(t).Kind() == reflect.Interface {
		panic(fmt.Sprintf("no binding found for %s", t))
	}

	// All others can be automatically created
	return A.autogen(a, injectables)
}

func (A *App) processObject(x *Object, injectables map[string]interface{}) interface{} {
	if x.Kind == Redirect {
		// Follow the redirect
		return A.MakeWith(x.Value, injectables)
	}
	if x.IsSingleton() {
		// Singleton can only be Ptr
		if x.Kind != Ptr {
			panic(fmt.Sprintf("Unsupported Singleton Type %d", x.Kind))
		}

		return x.Value
	}

	if x.Kind == Func {
		// Run the BindFunc
		bf := A.interfaceToBindFunc(x.Value)
		return bf(A)
		/*
			res := reflect.ValueOf(x.Value).Call([]reflect.Value{reflect.ValueOf(A)})
			if len(res) == 0 {
				panic(fmt.Sprintf("failed running make func for %s", reflect.TypeOf(a)))
			}

			return res[0].Interface()
		*/
	} else if x.Kind == Struct {
		// Build the structure automatically
		return A.autogen(x.Value, injectables)
	} else if x.Kind == Ptr {
		// Build the pointer automatically
		return A.autogen(x.Value, injectables)
	}

	// Unknown type, shouldn't trigger
	panic(fmt.Sprintf("Unsupported Type %d", x.Kind))
}

func (A *App) autogen(a interface{}, injectables map[string]interface{}) interface{} {
	t := reflect.TypeOf(a)
	_, e := t.MethodByName("New")

	if e {
		// A function called New is defined, use as a constructor
		//return A.makeByNew(a, injectables) // Not yet supported
		return A.makeByNew(a)
	} else {
		// No function exists, examine the tags
		return A.makeByHints(a, injectables)
	}
}

func (A *App) makeByHints(a interface{}, injectables map[string]interface{}) interface{} {
	t := reflect.TypeOf(a)
	ot := t

	// Resolve the object type to actual thing the pointer points to
	for k := t.Kind(); k == reflect.Ptr; {
		t = t.Elem()
		k = t.Kind()
	}

	// Create a new instance of object to work with
	newobj := reflect.New(t)

	// Use injection registry - if x needs y give z
	hintmap, hasmap := A.injectRegistry[A.typeFullName(reflect.TypeOf(a))]

	// Iterate over the fields of the struct
	for fn := 0; fn < t.NumField(); fn++ {
		f := t.Field(fn)
		// Obtain tag inject values
		injectValue, inject := f.Tag.Lookup("inject")
		newField := newobj.Elem().Field(fn)
		if inject && newField.CanSet() {
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
			if pe && newField.Type() == reflect.TypeOf(pv) {
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
					c = A.Make(pPtr.Interface())
				} else if f.Type.Kind() == reflect.Struct {
					pPtr := reflect.New(f.Type)
					c = A.Make(pPtr.Elem().Interface())
				} else if f.Type.Kind() == reflect.Interface {
					pPtr := reflect.New(f.Type)
					c = A.Make(pPtr.Interface())
				} else if f.Type.Kind() == reflect.Int {
					panic(fmt.Sprintf("Value must be specified when injecting int"))
				} else if f.Type.Kind() == reflect.String {
					panic(fmt.Sprintf("Value must be specified when injecting string"))
				}

				if c == nil {
					// Not a valid type to inject on, remove the tag
					panic(fmt.Sprintf("Could not inject %s (%s)", f.Type, f.Type.Kind()))
				}

				newField.Set(reflect.ValueOf(c))
			}
		}
	}

	// Convert Ptr to Struct if requested
	if ot.Kind() == reflect.Struct {
		return newobj.Elem().Interface()
	}

	return newobj.Interface()
}

/*
func (A *App) makeByNew(a interface{}, injectables map[string]interface{}) interface{} {
	// Injectables for function not yet supported
	// Go does not provide names of parameters
	// Solution to be decided upon
*/

func (A *App) makeByNew(a interface{}) interface{} {
	t := reflect.TypeOf(a)
	valIn := reflect.ValueOf(a)

	// We can't call "New" on a nil ptr object so create a new intance of type to work with
	if t.Kind() == reflect.Ptr && valIn.IsNil() {
		ct := t

		for k := ct.Kind(); k == reflect.Ptr; {
			ct = ct.Elem()
			k = ct.Kind()
		}

		newobj := reflect.New(ct)
		a = newobj.Interface()
		valIn = reflect.ValueOf(a)
	}

	// Obtain preset injection map for object
	hintmap, hasmap := A.injectRegistry[A.typeFullName(reflect.TypeOf(a))]

	method, _ := t.MethodByName("New")

	injects := []reflect.Value{valIn}

	// Iterate over the function parameters
	for v := 1; v < method.Type.NumIn(); v++ {
		var c interface{}

		childType := method.Type.In(v)

		var pPtr reflect.Value

		fmt.Println(childType.Kind())

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
			c = A.Make(pPtr.Interface())
		} else if childType.Kind() == reflect.Interface {
			c = A.Make(A.typeFullName(pPtr.Type()))
		} else if childType.Kind() == reflect.Struct {
			c = A.Make(pPtr.Elem().Interface())
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
	if t.Kind() == y[0].Kind() {
		return y[0].Interface()
	} else if t.Kind() == reflect.Struct && y[0].Kind() == reflect.Ptr {
		return y[0].Elem().Interface()
	} else if t.Kind() == reflect.Ptr && y[0].Kind() == reflect.Struct {
		z := reflect.New(y[0].Type())
		z.Elem().Set(y[0])
		return z.Interface()
	}

	panic(fmt.Sprintf("Unexpected error occurred, type of New %s does not match requested type %s", y[0].Kind(), t.Kind()))
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
		iv, err := strconv.ParseInt(v, 10, 15)
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

// Get full path & name for type for almost unique naming
func (A *App) typeFullName(t reflect.Type) string {
	pt := A.resolveTypePtr(t)
	return pt.PkgPath() + "/" + t.String()
}

// If a Ptr type, obtains type it points to
func (A *App) resolveTypePtr(t reflect.Type) reflect.Type {
	for k := t.Kind(); k == reflect.Ptr; {
		t = t.Elem()
		k = t.Kind()
	}
	return t
}

// Converts an interface into a BindFunc
func (A *App) interfaceToBindFunc(a interface{}) BindFunc {
	var bf BindFunc
	aType := reflect.TypeOf(a)
	if !aType.ConvertibleTo(reflect.TypeOf(bf)) {
		panic("Unsupported function type, must be compatible with BindFunc")
	}
	bf = reflect.ValueOf(a).Convert(reflect.TypeOf(bf)).Interface().(BindFunc)
	return bf
}

// Checks if Func returns type, returns bool, the type & value
func (A *App) isFuncReturnCompatible(bf BindFunc, t reflect.Type) (bool, reflect.Type, interface{}) {
	r := bf(A)
	rType := reflect.TypeOf(r)

	return A.typeChecker.IsTypeCompatible(t, rType, true), rType, r
}
