# go-di-container - Architecture Overview

## Summary
A Go dependency injection (DI) / Inversion of Control (IoC) container library using reflection. Module: `github.com/daforester/go-di-container`, Go 1.20.

## Core Components

### `di.App` (app.go)
The main DI container. Key methods:
- `New(config ...AppConfig) *App` - Creates a new container instance
- `Default(name ...string) AppInterface` - Gets/creates the global default or named app instance
- `Bind(a, b interface{}) AppInterface` - Registers a binding (implementation `b` for type `a`)
- `Singleton(a interface{}, c ...interface{}) AppInterface` - Registers a singleton binding
- `Make(a interface{}) interface{}` - Resolves and creates an instance of type `a`
- `MakeWith(a interface{}, injectables map[string]interface{}) interface{}` - Make with field overrides
- `When(a interface{}) *whenLink` - Fluent API for contextual bindings (When X needs Y, give Z)

### `di.Object` (object.go)
Wraps a bound value with metadata. Kinds: `Func`, `Ptr`, `Redirect`, `Struct`, `Primitive`, `Unknown`.

### `di.TypeChecker` (typechecker.go)
Validates type compatibility (interface implementation, pointer/struct equivalence).

### `whenLink` / `needLink` (when.go)
Fluent builder for contextual injection: `app.When(&A{}).Needs((*B)(nil)).Give(&C{})`

## Binding Types

| Pattern | Example |
|---------|---------|
| Interface to struct | `Bind((*MyInterface)(nil), MyStruct{})` |
| Interface to pointer | `Bind((*MyInterface)(nil), &MyStruct{})` |
| Interface to func | `Bind((*MyInterface)(nil), func(a *App) interface{} { return &MyStruct{} })` |
| String alias | `Bind("myalias", "otheralias")` |
| String to struct | `Bind("mykey", MyStruct{})` |
| Struct/Ptr to func (constructor) | `Bind(&MyStruct{}, func(a *App) interface{} { ... })` |

## Injection Mechanisms

1. **`inject` struct tags**: `Field int \`inject:"42"\`` or `Field MyInterface \`inject:""\``
2. **`New()` method**: If a type has a `New(...)` method, it's used as a constructor
3. **`When().Needs().Give()`**: Contextual injection for specific types
4. **`MakeWith()`**: Per-call field overrides via map

## Registries
- `registry map[string]ObjectInterface` - Main binding registry, keyed by type full name or string alias
- `injectRegistry map[string]map[string]ObjectInterface` - Contextual injection rules, keyed by [requesting type][needed type]

## Key Design Notes
- Uses `reflect` extensively for runtime type resolution
- Type names use `PkgPath + "/" + Type.String()` for uniqueness
- Global state: `defaultApp` and `instances` maps are package-level singletons
- `AppConfig` allows injecting mock `ObjectBuilder` and `TypeChecker` for testing
- Errors are raised via `panic()`, not returned errors
