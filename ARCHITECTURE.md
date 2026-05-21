# go-di-container - Architecture Overview

## Summary
A Go dependency injection (DI) / Inversion of Control (IoC) container library using reflection. Module: `github.com/daforester/go-di-container`, Go 1.20.

## Core Components

### `di.App` (app.go)
The main DI container. Key methods:
- `New(config ...AppConfig) *App` - Creates a new container instance
- `Default(name ...string) AppInterface` - Gets/creates the global default or named instance
- `Bind(a, b interface{}) AppInterface` - Registers a binding (implementation `b` for type `a`)
- `Singleton(a interface{}, c ...interface{}) AppInterface` - Registers a singleton binding
- `Make(a interface{}) interface{}` - Resolves and creates an instance of type `a`
- `MakeWith(a interface{}, injectables map[string]interface{}) interface{}` - Make with per-call field overrides
- `When(a interface{}) *whenLink` - Fluent API for contextual bindings (When X needs Y, give Z)

### `di.Object` (object.go)
Wraps a bound value with metadata. Kinds: `Func`, `Ptr`, `Redirect`, `Struct`, `Primitive`, `Unknown`.

### `di.TypeChecker` (typechecker.go)
Validates type compatibility: interface implementation, pointer/struct equivalence (non-strict mode), and exact type matches (strict mode).

### `whenLink` / `needLink` (when.go)
Fluent builder for contextual injection: `app.When(&A{}).Needs((*B)(nil)).Give(&C{})`

## Resolution Order

When `Make(a)` is called:

1. **Registry lookup** - check `registry` for a binding matching type `a`
2. **processObject** - if found, dispatch by Kind:
   - `Redirect` → follow to another binding
   - `Singleton` → return cached value (any Kind)
   - `Func` → run BindFunc, then `processStructTags`
   - `Struct`/`Ptr` → `autogen`
   - `Primitive` → return value directly
3. **autogen** - if no binding exists (or dispatched from processObject):
   - If type has a `New()` method → `makeByNew`
   - Otherwise → `makeByHints`
4. **makeByNew** - calls the `New()` constructor with auto-resolved parameters (checking When registry for overrides), then runs `processStructTags` on the result
5. **makeByHints** - creates a new instance and processes each field by tag:
   - `di` + `inject` with value on a primitive → set the literal value
   - `di` alone → inject the container (`*App` or `AppInterface`)
   - `inject` → check MakeWith overrides, then When registry, then tag value, then auto-resolve
6. **processStructTags** - post-constructor tag processing (runs after `New()`):
   - `inject` tags always overwrite (even values set by `New()`)
   - `di` tags only inject if the field is still zero (preserves `New()` values)
   - Consults MakeWith overrides and When registry

## Binding Types

| Pattern | Example |
|---------|---------|
| Interface to struct | `Bind((*MyInterface)(nil), MyStruct{})` |
| Interface to pointer | `Bind((*MyInterface)(nil), &MyStruct{})` |
| Interface to func | `Bind((*MyInterface)(nil), func(a *App) interface{} { return &MyStruct{} })` |
| String alias | `Bind("myalias", "otheralias")` |
| String to struct | `Bind("mykey", MyStruct{})` |
| String to primitive | `Bind("port", 8080)` |
| Struct/Ptr to func | `Bind(&MyStruct{}, func(a *App) interface{} { ... })` |

## Struct Tags

| Tag | Behavior |
|-----|----------|
| `inject:""` | Auto-resolve the field's type from the container |
| `inject:"value"` | Parse and set a literal value (primitives only) |
| `di:""` | Inject the container itself (`*App` or `AppInterface`) |
| `di:"" inject:"value"` | For primitives: use the inject value; for non-primitives: inject container |

## Registries
- `registry map[string]ObjectInterface` - Main binding registry, keyed by type full name or string alias
- `injectRegistry map[string]map[string]ObjectInterface` - Contextual injection rules, keyed by `[requesting type][needed type]`

## Thread Safety
All public methods (`Bind`, `Singleton`, `Make`, `MakeWith`, `When().Needs().Give()`) acquire a per-container reentrant mutex. The lock is reentrant so that BindFunc callbacks can safely call `Make` on the same container without deadlocking.

Internal methods (`makeInternal`, `makeWithInternal`, etc.) operate without locking and are called from within the lock scope.

The global `defaultApp` and `instances` maps are protected by a separate `sync.RWMutex`.

## Circular Dependency Detection
`makeWithInternal` tracks which types are currently being resolved in a `resolving` map. If a type appears while already being resolved (A needs B, B needs A), the container panics with a clear message instead of causing a stack overflow.

## Key Design Notes
- Uses `reflect` extensively for runtime type resolution
- Type names use `PkgPath + "/" + Type.String()` for uniqueness
- `AppConfig` allows injecting mock `ObjectBuilder` and `TypeChecker` for testing
- Errors are raised via `panic()`, not returned errors
- `isFuncSignatureCompatible` validates BindFunc return types against the target when concrete (non-`interface{}`) return types are used
- `Singleton` with a BindFunc validates the return type at registration time, not just at resolve time
- `typeFullName` and `resolveTypePtr` are package-level functions shared by both `App` and `Object`
