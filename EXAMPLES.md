# go-di-container - Common Patterns & Examples

## Basic Setup
```go
c := di.New()
// or use the global singleton
c := di.Default()
```

## Interface Binding
```go
type Service interface { Do() }
type MyService struct{}
func (m MyService) Do() {}

c.Bind((*Service)(nil), MyService{})
s := c.Make((*Service)(nil)).(Service)
```

## Constructor Function Binding
```go
c.Bind((*Service)(nil), func(a *di.App) interface{} {
    return &MyService{ /* init */ }
})
```

## Singleton
```go
// Singleton instance
c.Singleton(&MyService{Config: "prod"})

// Singleton via constructor
c.Singleton((*Service)(nil), func(a *di.App) interface{} {
    return &MyService{Config: "prod"}
})

// Singleton for interface with nil pointer (auto-construct)
c.Singleton((*Service)(nil), (*MyService)(nil))
```

## Struct Field Injection via `inject` Tag
```go
type Config struct {
    Port     int     `inject:"8080"`
    Host     string  `inject:"localhost"`
    Debug    bool    `inject:"true"`
    Timeout  float64 `inject:"30.5"`
}

cfg := c.Make(Config{}).(Config)
// cfg.Port == 8080, cfg.Host == "localhost", etc.
```

## Dependency Injection via `inject` Tag
```go
type Handler struct {
    DB    Database    `inject:""`
    Cache *RedisCache `inject:""`
    Name  string      `inject:"my-handler"`
}

c.Bind((*Database)(nil), &PostgresDB{})
h := c.Make(&Handler{}).(*Handler)
// h.DB is auto-resolved, h.Name == "my-handler"
```

## Dependency Injection via `di` Tag
```go
type OrderProcessor struct {
    Repo    *OrderRepo  `di:""`  // resolved from container
    Mailer  Mailer      `di:""`  // interface resolved from container
}

c.Bind((*Mailer)(nil), &SMTPMailer{})
p := c.Make(&OrderProcessor{}).(*OrderProcessor)
// p.Repo auto-constructed, p.Mailer is SMTPMailer
```

The `di` tag is an alternative to listing dependencies as `New()` parameters.
It only fills zero-value fields, so values set by a `New()` constructor are preserved.

## Container Self-Injection via `di` Tag
```go
type MyService struct {
    Container *di.App         `di:""`  // inject as concrete type
    App       di.AppInterface `di:""`  // inject as interface
}

svc := c.Make(&MyService{}).(*MyService)
// svc.Container is the container that resolved it
```
When the field type is `*di.App` or `di.AppInterface`, the container injects itself.

## Dual `di` + `inject` Tags
When both tags are present, the `inject` value is used for primitive fields; `di`
resolution applies for non-primitive fields (where an inject value would be meaningless).

```go
type AppConfig struct {
    Container *di.App `di:"" inject:""`     // non-primitive: di resolves (injects container)
    Port      int     `di:"" inject:"8080"` // primitive: inject value used
    Name      string  `di:"" inject:"app"`  // primitive: inject value used
}

cfg := c.Make(&AppConfig{}).(*AppConfig)
// cfg.Container == c, cfg.Port == 8080, cfg.Name == "app"
```

## Contextual Injection (When/Needs/Give)
```go
// When AdminHandler needs a Database, give it AdminDB
c.When(&AdminHandler{}).Needs((*Database)(nil)).Give(&AdminDB{})

// When UserHandler needs a Database, give it UserDB
c.When(&UserHandler{}).Needs((*Database)(nil)).Give(&UserDB{})

// Works with both struct field injection and New() constructor parameters
```

## Constructor Method Pattern
```go
type Service struct {
    repo Repository
    name string
}

func (s Service) New(repo Repository) *Service {
    return &Service{repo: repo, name: "default"}
}

c.Bind((*Repository)(nil), &MemoryRepo{})
s := c.Make(&Service{}).(*Service)
// s.repo is auto-injected via New(), s.name == "default"
```

## MakeWith (Per-Call Overrides)
```go
type Config struct {
    Port int    `inject:"8080"`
    Host string `inject:"localhost"`
}

cfg := c.MakeWith(&Config{}, map[string]interface{}{
    "Port": 9090,
}).(*Config)
// cfg.Port == 9090, cfg.Host == "localhost" (tag default)
```

## Named Bindings & Aliases
```go
// Named binding
c.Bind("primaryDB", &PostgresDB{})
c.Bind("replicaDB", &ReadOnlyDB{})

// Alias
c.Bind("defaultDB", "primaryDB")

// Retrieve by name
db := c.Make("primaryDB")
```

## Named Primitive Bindings
```go
c.Bind("port", 8080)
c.Bind("host", "localhost")  // Note: strings create redirects, not primitive bindings
c.Bind("debug", true)
c.Bind("rate", 3.14)

port := c.Make("port").(int)      // 8080
debug := c.Make("debug").(bool)   // true
rate := c.Make("rate").(float64)  // 3.14
```

## Removing Bindings
```go
c.Bind((*Service)(nil), nil)       // Remove binding
c.Singleton((*Service)(nil), nil)  // Remove singleton

// Remove contextual binding
c.When(&Handler{}).Needs((*Service)(nil)).Give(nil)
```

## Named App Instances
```go
app1 := di.Default("app1")
app2 := di.Default("app2")
// Each has its own independent registry
```

## BindFunc Calling Make (Nested Resolution)
```go
c.Bind((*Service)(nil), func(a *di.App) interface{} {
    // Safe: reentrant locking allows this
    config := a.Make(&Config{}).(*Config)
    return &MyService{Port: config.Port}
})
```

## Concurrent Usage
```go
c := di.New()
c.Bind((*Service)(nil), func(a *di.App) interface{} {
    return &MyService{}
})

// Safe to call from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        svc := c.Make((*Service)(nil)).(Service)
        svc.Do()
    }()
}
wg.Wait()
```
