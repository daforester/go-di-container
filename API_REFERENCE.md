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

## Types

### `App`
The DI container. Created via `di.New()` or `di.Default()`.

### `AppConfig`
```go
type AppConfig struct {
    Name          string             // Named app instance identifier
    ObjectBuilder ObjectInterface    // Custom object builder (for mocking)
    TypeChecker   TypeCheckerInterface // Custom type checker (for mocking)
    Default       bool               // Make this the default app
}
```

### `Object`
Internal wrapper for bound values.

### `Kind` (uint)
```go
const (
    Unknown = iota
    Func
    Ptr
    Redirect
    Struct
    Primitive
)
```

## Functions

### `di.New(config ...AppConfig) *App`
Creates a new DI container. Sets itself as default if no default exists.

### `di.Default(name ...string) AppInterface`
Returns the default app or a named app instance. Creates if not found.

## App Methods

### `Bind(a, b interface{}) AppInterface`
Registers a binding. Valid combinations:
- String → String (alias redirect)
- String → Struct/Ptr/Func/Interface (named binding)
- Interface → Struct/Ptr/Func (interface implementation)
- Struct/Ptr → Func (constructor function)
- Pass `nil` as `b` to remove a binding

### `Singleton(a interface{}, c ...interface{}) AppInterface`
Registers a singleton (always returns same instance):
- `Singleton(&instance)` - singleton of the instance's type
- `Singleton((*Interface)(nil), &instance)` - singleton for interface
- `Singleton((*Interface)(nil), bindFunc)` - singleton built via func
- Pass `nil` as second arg to remove binding

### `Make(a interface{}) interface{}`
Resolves and returns an instance:
- If binding exists: uses registry
- If struct/ptr with no binding: auto-generates using `inject` tags or `New()` method
- Panics if interface or string binding not found

### `MakeWith(a interface{}, injectables map[string]interface{}) interface{}`
Same as `Make` but with per-call field overrides. Map keys are field names.

### `When(a interface{}) *whenLink`
Starts a contextual binding chain:
```go
app.When(&RequestingType{}).Needs((*DependencyInterface)(nil)).Give(&ConcreteImpl{})
```

## Struct Tags

### `inject`
```go
type MyStruct struct {
    // Inject a literal value
    Count int `inject:"42"`
    Name  string `inject:"hello"`
    Flag  bool `inject:"true"`
    Pi    float64 `inject:"3.14159"`
    
    // Inject a dependency (empty tag = auto-resolve)
    Service MyInterface `inject:""`
    
    // Inject a pointer dependency
    Repo *MyRepository `inject:""`
}
```

Supported primitive types for literal injection: bool, float32/64, int/int8/16/32/64, uint/8/16/32/64, string.

## Constructor Pattern

Define a `New()` method on your struct to use as a constructor:
```go
type MyService struct {
    dep Dependency
}

func (m MyService) New(dep Dependency) *MyService {
    return &MyService{dep: dep}
}
```
The DI container will call `New()` with auto-resolved dependencies.
