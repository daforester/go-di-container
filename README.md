# go-di-container

Go-Di-Container is a Go library for Dependency Injection / Inversion of Control (IoC).

##### Latest Version
v1.1.0

## Features

* **Interface binding** - bind concrete implementations to interfaces
* **Constructor functions** - use `BindFunc` factories for custom setup
* **Constructor methods** - types with a `New()` method are auto-constructed
* **Singletons** - bind a shared instance that's returned on every resolve
* **Named bindings & aliases** - register and resolve by string keys
* **Struct tag injection** - `inject:""` auto-resolves dependencies, `inject:"value"` sets primitives
* **Container injection** - `di:""` tag injects the container itself
* **Dual tags** - fields with both `di` and `inject` tags use the inject value for primitives
* **Contextual injection** - `When().Needs().Give()` overrides bindings per requesting type
* **Per-call overrides** - `MakeWith()` provides field values at resolve time
* **Circular dependency detection** - panics with a clear message instead of stack overflow
* **Thread safety** - all public methods are safe for concurrent use via reentrant locking

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/daforester/go-di-container/di"
)

type Logger interface {
    Log(msg string)
}

type ConsoleLogger struct{}

func (c ConsoleLogger) Log(msg string) { fmt.Println(msg) }

type UserService struct {
    Logger Logger `inject:""`
    Name   string `inject:"user-service"`
}

func main() {
    c := di.New()

    c.Bind((*Logger)(nil), &ConsoleLogger{})

    svc := c.Make(&UserService{}).(*UserService)
    svc.Logger.Log("Hello from " + svc.Name)
}
```

## Struct Tags

```go
type Config struct {
    Port    int          `inject:"8080"`      // primitive value injection
    Host    string       `inject:"localhost"`  // string value injection
    DB      Database     `inject:""`           // auto-resolve interface
    Cache   *RedisCache  `inject:""`           // auto-resolve pointer
    App     *di.App      `di:""`              // inject the container
    Timeout int          `di:"" inject:"30"`  // dual: inject value for primitives
}
```

## Named Primitive Bindings

```go
c.Bind("port", 8080)
c.Bind("debug", true)
c.Bind("rate", 3.14)

port := c.Make("port").(int)  // 8080
```

## Contextual Injection

```go
// When AdminHandler needs a Database, give it AdminDB instead of the default
c.When(&AdminHandler{}).Needs((*Database)(nil)).Give(&AdminDB{})
```

## Constructor Method

```go
type Service struct {
    repo Repository
}

func (s Service) New(repo Repository) *Service {
    return &Service{repo: repo}
}

// Container auto-calls New() with resolved dependencies
svc := c.Make(&Service{}).(*Service)
```

## Documentation

- [API Reference](API_REFERENCE.md)
- [Architecture](ARCHITECTURE.md)
- [Examples](EXAMPLES.md)

## License

Go-Di-Container is published under the [Commons Clause License](https://commonsclause.com/)

###"Commons Clause" License Condition v1.0

The Software is provided to you by the Licensor under the License, as defined below, subject to the following condition.

Without limiting other conditions in the License, the grant of rights under the License will not include, and the License does not grant to you, the right to Sell the Software.

For purposes of the foregoing, "Sell" means practicing any or all of the rights granted to you under the License to provide to third parties, for a fee or other consideration (including without limitation fees for hosting or consulting/ support services related to the Software), a product or service whose value derives, entirely or substantially, from the functionality of the Software. Any license notice or attribution required by the License must also include this Commons Clause License Condition notice.

Software: go-di-container

License: Commons Clause

Licensor: Aaron Parker
