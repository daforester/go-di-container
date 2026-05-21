# go-di-container - API Reference

## Package
```go
import "github.com/daforester/go-di-container/di"
```

## Interfaces

### `AppInterface`
```go
type AppInterface interface {
    New(...AppConfig) AppInterface
    Bind(interface{}, interface{}) AppInterface
    Singleton(interface{}, ...interface{}) AppInterface
    Make(interface{}) interface{}
    MakeWith(interface{}, map[string]interface{}) interface{}
    When(a interface{}) *whenLink
}
```

### `ObjectInterface`
```go
type ObjectInterface interface {
    New(interface{}, ...Kind) ObjectInterface
    Singleton() ObjectInterface
    IsSingleton() bool
}
```

### `TypeCheckerInterface`
```go
type TypeCheckerInterface interface {
    IsTypeCompatible(reflect.Type, reflect.Type, bool) bool
}
```

### `BindFunc`
```go
type BindFunc func(*App) interface{}
```
Factory function that receives the container and returns an instance. BindFuncs
can safely call `Make` on the container they receive.

## Types

### `App`
The DI container. Created via `di.New()` or `di.Default()`. All public methods
are safe for concurrent use.

### `AppConfig`
```go
type AppConfig struct {
    Name          string             // Named instance identifier
    ObjectBuilder ObjectInterface    // Custom object builder (for testing)
    TypeChecker   TypeCheckerInterface // Custom type checker (for testing)
    Default       bool               // Make this the default app
}
```

### `Object`
Internal wrapper for bound values.

### `Kind` (uint)
```go
const (
    Unknown   Kind = iota
    Func           // BindFunc factory
    Ptr            // pointer to a struct
    Redirect       // string alias to another binding
    Struct         // concrete struct value
    Primitive      // primitive value
)
```

## Functions

### `di.New(config ...AppConfig) *App`
Creates a new DI container. Sets itself as default if no default exists.

### `di.Default(name ...string) AppInterface`
Returns the default app or a named instance. Creates one if it doesn't exist.

## App Methods

### `Bind(a, b interface{}) AppInterface`
Registers a binding. Valid combinations:
- String -> String (alias redirect)
- String -> Struct/Ptr/Func/Interface/Primitive (named binding)
- Interface -> Struct/Ptr/Func (interface implementation)
- Struct/Ptr -> Func (constructor function)
- Pass `nil` as `b` to remove a binding

### `Singleton(a interface{}, c ...interface{}) AppInterface`
Registers a singleton (always returns same instance):
- `Singleton(&instance)` - singleton of the instance's type
- `Singleton((*Interface)(nil), &instance)` - singleton for interface
- `Singleton((*Interface)(nil), bindFunc)` - singleton built via func
- Pass `nil` as second arg to remove binding

When a BindFunc is provided, its return value is validated at registration time
to ensure type compatibility with the target type.

### `Make(a interface{}) interface{}`
Resolves and returns an instance:
- If binding exists: uses registry
- If struct/ptr with no binding: auto-generates using `inject`/`di` tags or `New()` method
- Panics if interface or string binding not found
- Panics if a circular dependency is detected

### `MakeWith(a interface{}, injectables map[string]interface{}) interface{}`
Same as `Make` but with per-call field overrides. Map keys are exported field names.
MakeWith overrides take precedence over When bindings and tag values.

### `When(a interface{}) *whenLink`
Starts a contextual binding chain:
```go
app.When(&RequestingType{}).Needs((*DependencyInterface)(nil)).Give(&ConcreteImpl{})
```
When bindings apply to both struct field injection and `New()` constructor parameters.
Pass `nil` to `Give()` to remove a contextual binding.

## Struct Tags

### `inject` tag
```go
type MyStruct struct {
    // Literal values (primitives only)
    Count   int     `inject:"42"`
    Name    string  `inject:"hello"`
    Flag    bool    `inject:"true"`
    Pi      float64 `inject:"3.14159"`

    // Auto-resolve dependency (empty tag)
    Service MyInterface `inject:""`
    Repo    *MyRepo     `inject:""`
}
```
Supported primitive types: bool, string, float32/64, int/int8/16/32/64, uint/8/16/32/64.

**Important**: `inject` tags always overwrite the field value, even after a `New()` constructor runs.

### `di` tag
```go
type MyStruct struct {
    Container *di.App      `di:""`  // inject the container
    App       di.AppInterface `di:""` // also works with the interface
}
```
The `di` tag injects the container itself. Unlike `inject`, `di` tags only set the field
if its value is zero (preserving values set by a `New()` constructor).

### Dual `di` + `inject` tags
```go
type MyStruct struct {
    Port      int          `di:"" inject:"8080"`     // uses inject value (primitive)
    Container *di.App      `di:"" inject:""`          // uses di (non-primitive)
}
```
When both tags are present on a primitive field with an inject value, the inject value is used.

## Constructor Pattern

Define a `New()` method on your struct to use as a constructor:
```go
type MyService struct {
    dep  Dependency
    name string
}

func (m MyService) New(dep Dependency) *MyService {
    return &MyService{dep: dep, name: "default"}
}
```
The container calls `New()` with auto-resolved parameters, then processes remaining
`inject`/`di` tags on the result. When bindings apply to both `New()` parameters
and inject-tagged fields.

## Thread Safety

All public methods on `App` are protected by a reentrant mutex. This means:
- Multiple goroutines can safely call `Make`, `Bind`, etc. concurrently
- BindFunc callbacks can safely call `Make` on the container they receive
- Container setup and resolution can happen from different goroutines

The global `Default()` and named instance registries use a separate `sync.RWMutex`.

## Precedence Order

When resolving an `inject:""` tagged field, the container checks (in order):
1. **MakeWith overrides** - per-call field values
2. **When/Needs/Give** - contextual binding for this requesting type
3. **inject value** - literal value from the tag (e.g., `inject:"42"`)
4. **Auto-resolve** - recursive `Make` call for the field's type
