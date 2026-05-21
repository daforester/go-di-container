# go-di-container - Agent Quick Reference

## Project Structure
```
go-di-container/
├── di/
│   ├── app.go          # Main container: App, Bind, Singleton, Make, MakeWith, When
│   ├── object.go       # Object wrapper: Kind enum (Func/Ptr/Redirect/Struct/Primitive)
│   ├── typechecker.go  # Type compatibility checking
│   ├── when.go         # Fluent API: When().Needs().Give()
│   └── *_test.go       # Tests with mock implementations
├── go.mod              # module github.com/daforester/go-di-container, go 1.20
└── README.md
```

## Key Entry Points
- `di.New()` - new container
- `di.Default()` - global singleton container
- `di.Default("name")` - named container instance

## Core Workflow
1. Create container: `c := di.New()`
2. Register bindings: `c.Bind((*Interface)(nil), Impl{})`
3. Resolve: `obj := c.Make((*Interface)(nil))`

## Binding Patterns
| What | Syntax |
|------|--------|
| Interface -> impl | `c.Bind((*I)(nil), S{})` |
| Interface -> ptr | `c.Bind((*I)(nil), &S{})` |
| Interface -> factory | `c.Bind((*I)(nil), func(a *di.App) interface{} { return &S{} })` |
| Named | `c.Bind("key", value)` |
| Named primitive | `c.Bind("port", 8080)` |
| Alias | `c.Bind("alias", "key")` |
| Constructor | `c.Bind(&S{}, func(a *di.App) interface{} { ... })` |

## Singleton Patterns
| What | Syntax |
|------|--------|
| Instance | `c.Singleton(&S{})` |
| Interface -> instance | `c.Singleton((*I)(nil), &S{})` |
| Interface -> factory | `c.Singleton((*I)(nil), func(a *di.App) interface{} { return &S{} })` |

## Struct Tags
| Tag | Effect |
|-----|--------|
| `inject:"42"` | Set literal value (primitives only) |
| `inject:""` | Auto-resolve dependency from container |
| `di:""` | Inject the container itself |
| `di:"" inject:"42"` | Primitives: use inject value; non-primitives: inject container |

## Constructor Method
If a type has `New(...)` method, it's called as constructor:
```go
func (s S) New(dep Dependency) *S { return &S{dep: dep} }
```

## Contextual Injection
```go
c.When(&RequestingType{}).Needs((*Dependency)(nil)).Give(&Concrete{})
```
Applies to both `New()` parameters and inject-tagged struct fields.

## Override at Make Time
```go
c.MakeWith(&S{}, map[string]interface{}{"FieldName": value})
```

## Resolution Precedence (for inject-tagged fields)
1. MakeWith overrides (per-call)
2. When/Needs/Give (contextual)
3. inject tag value (literal)
4. Auto-resolve (recursive Make)

## Internal Architecture
- **Public methods** (`Bind`, `Singleton`, `Make`, `MakeWith`) acquire a reentrant lock, then delegate to internal methods
- **makeByHints** - builds instances via struct tag inspection (no `New()` method)
- **makeByNew** - calls the `New()` constructor, then runs `processStructTags`
- **processStructTags** - post-constructor: inject tags always overwrite, di tags only set zero fields
- **processObject** - dispatches by Object.Kind (Redirect/Singleton/Func/Struct/Ptr/Primitive)

## Important Behaviors
- **Thread safe** - all public methods use a reentrant per-container mutex
- **Circular dependency detection** - panics instead of stack overflow
- **Panics on errors** - no error returns, invalid usage panics
- **Auto-generation** - structs/pointers without bindings are auto-created via reflection
- **Type naming** - uses `PkgPath + "/" + Type.String()` as registry keys
- **Interface binding syntax** - must use `(*Interface)(nil)` pattern
- **Removing bindings** - pass `nil` as the second argument to `Bind`, `Singleton`, or `Give`
- **BindFunc re-entry** - BindFuncs can safely call `Make` on the container they receive

## Testing
- Tests use `MockObject` and `MockTypeChecker` injected via `AppConfig`
- Run tests: `go test -race ./di/`

## Common Pitfalls
1. Use `(*Interface)(nil)` not `nil` or `new(Interface)` for interface bindings
2. Primitive fields with empty `inject:""` tag will panic (must provide a value)
3. Singleton must be a pointer type (the first argument)
4. `BindFunc` must match `func(*di.App) interface{}` signature
5. `inject` tags always overwrite field values, even those set by `New()` constructors
6. `di` tags only inject if the field is zero (preserves `New()` constructor values)
7. `Bind("key", "value")` creates a redirect (alias), not a primitive string binding
8. Singleton BindFunc return types are validated at registration time
