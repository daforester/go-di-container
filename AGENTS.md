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
| Interface → impl | `c.Bind((*I)(nil), S{})` |
| Interface → ptr | `c.Bind((*I)(nil), &S{})` |
| Interface → factory | `c.Bind((*I)(nil), func(a *di.App) interface{} { return &S{} })` |
| Named | `c.Bind("key", value)` |
| Alias | `c.Bind("alias", "key")` |
| Constructor | `c.Bind(&S{}, func(a *di.App) interface{} { ... })` |

## Singleton Patterns
| What | Syntax |
|------|--------|
| Instance | `c.Singleton(&S{})` |
| Interface → instance | `c.Singleton((*I)(nil), &S{})` |
| Interface → factory | `c.Singleton((*I)(nil), func(a *di.App) interface{} { return &S{} })` |

## Injection Tags
```go
type S struct {
    IntField    int     `inject:"42"`
    StrField    string  `inject:"hello"`
    BoolField   bool    `inject:"true"`
    FloatField  float64 `inject:"3.14"`
    DepField    MyInterface `inject:""`     // auto-resolve
    PtrDepField *MyStruct   `inject:""`     // auto-resolve ptr
}
```

## Constructor Method
If a type has `New(...)` method, it's called as constructor:
```go
func (s S) New(dep Dependency) *S { return &S{dep: dep} }
```

## Contextual Injection
```go
c.When(&RequestingType{}).Needs((*Dependency)(nil)).Give(&Concrete{})
```

## Override at Make Time
```go
c.MakeWith(&S{}, map[string]interface{}{"FieldName": value})
```

## Important Behaviors
- **Panics on errors** - no error returns, invalid usage panics
- **Auto-generation** - structs/pointers without bindings are auto-created via reflection
- **Global state** - `defaultApp` and `instances` are package-level globals
- **Type naming** - uses `PkgPath + "/" + Type.String()` as registry keys
- **Interface binding syntax** - must use `(*Interface)(nil)` pattern, not `new(Interface)`
- **Removing bindings** - pass `nil` as the second argument to `Bind` or `Singleton`

## Testing
- Tests use `MockObject` and `MockTypeChecker` injected via `AppConfig`
- Run tests: `go test ./di/`

## Common Pitfalls
1. Use `(*Interface)(nil)` not `nil` or `new(Interface)` for interface bindings
2. `int` and `string` fields with empty `inject:""` tag will panic (must provide value)
3. Singleton must be a pointer type
4. `BindFunc` must match `func(*di.App) interface{}` signature
5. Contextual injection (`When`) only applies during `Make`/`MakeWith` resolution
