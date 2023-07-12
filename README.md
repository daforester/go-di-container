# go-di-container

[![Build Status](https://travis-ci.org/daforester/go-di-container.svg?branch=master)](https://travis-ci.org/daforester/go-di-container)
[![Coverage Status](https://coveralls.io/repos/github/daforester/go-di-container/badge.svg?branch=master)](https://coveralls.io/github/daforester/go-di-container?branch=master)
[![GolangCI](https://golangci.com/badges/github.com/daforester/go-di-container.svg)](https://golangci.com)

Go-Di-Container is a go library for providing Dependency Injection IoC.

##### Latest Version
v1.0.3

## Features

* Bind Funcs to perform additional object setup
* Bind a single response (singleton) for an class
* Bind specific struct to interface
* Bind aliases
* Specify injectable fields in structs
* Specify default values for int & string type fields in structs
* Override injectable values

## Example Usage

    type StructInterface interface {
    
    }
    
    type StructA struct {
        A int
        B string
    }
    
    type StructB struct {
        A int `inject:"201"`
        B string `inject:"202"`
    }
    
    type StructC struct {
        A int `inject:"301"`
        B string `inject:"303"`
    }
    
    type StructInject struct {
        TestStructI StructInterface `inject:""`
        TestStructS StructA    `inject:""`
        TestStructP *StructA   `inject:""`
        A           int        `inject:"101"`
        B           string     `inject:"TestString"`
    }

    func main() {
        var c *di.App
        c = di.New()
    
        var testStruct interface{}
        
        // Bind type StructC to StructInterface
        
        c.Bind((*StructInterface)(nil), StructC{})
        testStruct = c.Make(&StructInject{})
        fmt.Println(testStruct)
    
        // When StructInject needs a StructInterface provide it with a new StructB instance
        c.When(&StructInject{}).Needs((*StructInterface)(nil)).Give(&StructB{})
        testStruct = c.Make(&StructInject{})
        fmt.Println(testStruct)
        fmt.Println((*testStruct.(*StructInject)).TestStructI)
    
        // When StructA is created always use return provided StructA
        c.Singleton(&StructA{
            A: 5,
            B: "foobar",
        })
        testStruct = c.Make(&StructA{})
        fmt.Println(testStruct)
    }
    
## License

Go-Di-Container is published under the [Commons Clause License](https://commonsclause.com/)

###“Commons Clause” License Condition v1.0

The Software is provided to you by the Licensor under the License, as defined below, subject to the following condition.

Without limiting other conditions in the License, the grant of rights under the License will not include, and the License does not grant to you, the right to Sell the Software.

For purposes of the foregoing, “Sell” means practicing any or all of the rights granted to you under the License to provide to third parties, for a fee or other consideration (including without limitation fees for hosting or consulting/ support services related to the Software), a product or service whose value derives, entirely or substantially, from the functionality of the Software. Any license notice or attribution required by the License must also include this Commons Clause License Condition notice.

Software: go-di-container

License: Commons Clause

Licensor: Aaron Parker