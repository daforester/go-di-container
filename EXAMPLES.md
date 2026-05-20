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
c.Bind((*Service)(nil), func(a *App) interface{} {
    return &MyService{ /* init */ }
})
```

## Singleton
```go
// Singleton instance
c.Singleton(&MyService{Config: "prod"})

// Singleton via constructor
c.Singleton((*Service)(nil), func(a *App) interface{} {
    return &MyService{Config: "prod"}
})

// Singleton for interface with nil pointer (auto-construct)
c.Singleton((*Service)(nil), (*MyService)(nil))
```

## Struct Field Injection via Tags
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

## Dependency Injection via Tags
```go
type Handler struct {
    DB    Database    `inject:""`
    Cache *RedisCache `inject:""`
    Name  string      `inject:"my-handler"`
}

c.Bind((*Database)(nil), &PostgresDB{})
c.Bind((*RedisCache)(nil), &RedisCache{})
h := c.Make(&Handler{}).(*Handler)
// h.DB and h.Cache are auto-resolved
```

## Contextual Injection (When/Needs/Give)
```go
// When AdminHandler needs a Database, give it AdminDB
c.When(&AdminHandler{}).Needs((*Database)(nil)).Give(&AdminDB{})

// When UserHandler needs a Database, give it UserDB
c.When(&UserHandler{}).Needs((*Database)(nil)).Give(&UserDB{})
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
// s.repo is auto-injected, s.name == "default"
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

## Removing Bindings
```go
c.Bind((*Service)(nil), nil)       // Remove binding
c.Singleton((*Service)(nil), nil)  // Remove singleton
```

## Named App Instances
```go
app1 := di.Default("app1")
app2 := di.Default("app2")
// Each has its own registry
```
