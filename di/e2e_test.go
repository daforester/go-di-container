package di

import (
	"context"
	"sync"
	"testing"
	"time"
)

type UserService interface {
	GetUserName() string
}

type MockUserService struct {
	Name string
}

func (m MockUserService) GetUserName() string {
	return m.Name
}

type EmailService interface {
	Send(to, message string)
}

type MockEmailService struct {
	SentMessages []string
}

func (m *MockEmailService) Send(to, message string) {
	m.SentMessages = append(m.SentMessages, message)
}

type ConfigService struct {
	AppName string `inject:"MyApp"`
	Env     string `inject:"production"`
	Port    int    `inject:"8080"`
	Debug   bool   `inject:"false"`
}

type UserRepository struct {
	Config *ConfigService `inject:""`
}

type UserController struct {
	UserService  UserService  `inject:""`
	EmailService EmailService `inject:""`
	Container    *App         `di:""`
}

type OrderProcessor struct {
	Container    AppInterface `di:""`
	UserService  UserService  `inject:""`
	EmailService EmailService `inject:""`
}

func (o OrderProcessor) New(userService UserService) *OrderProcessor {
	return &OrderProcessor{
		UserService: userService,
	}
}

type NotificationService struct {
	Container    *App         `di:""`
	EmailService EmailService `inject:""`
	Processed    bool
}

func (n NotificationService) New() *NotificationService {
	return &NotificationService{
		Processed: true,
	}
}

func TestEndUser_ContainerInjection(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Alice"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	controller := c.Make(&UserController{}).(*UserController)

	if controller.UserService == nil {
		t.Error("UserService should be injected via inject tag")
	}
	if controller.UserService.GetUserName() != "Alice" {
		t.Error("UserService should return Alice")
	}
	if controller.EmailService == nil {
		t.Error("EmailService should be injected via inject tag")
	}
	if controller.Container == nil {
		t.Error("Container should be injected via di tag")
	}
}

func TestEndUser_ConfigInjection(t *testing.T) {
	c := New()

	config := c.Make(&ConfigService{}).(*ConfigService)

	if config.AppName != "MyApp" {
		t.Errorf("Expected MyApp, got %s", config.AppName)
	}
	if config.Env != "production" {
		t.Errorf("Expected production, got %s", config.Env)
	}
	if config.Port != 8080 {
		t.Errorf("Expected 8080, got %d", config.Port)
	}
	if config.Debug != false {
		t.Errorf("Expected false, got %v", config.Debug)
	}
}

func TestEndUser_DiTagWithNewMethod(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Bob"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	processor := c.Make(&OrderProcessor{}).(*OrderProcessor)

	if processor.UserService == nil {
		t.Error("UserService should be set by New method")
	}
	if processor.UserService.GetUserName() != "Bob" {
		t.Error("UserService should return Bob")
	}
	if processor.Container == nil {
		t.Error("Container should be injected via di tag since New left it nil")
	}
	if processor.EmailService == nil {
		t.Error("EmailService should be injected via inject tag")
	}
}

func TestEndUser_DiTagNewPriority(t *testing.T) {
	c := New()

	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	notif := c.Make(&NotificationService{}).(*NotificationService)

	if notif.Processed != true {
		t.Error("New method should set Processed to true")
	}
	if notif.Container == nil {
		t.Error("Container should be injected via di tag since New left it nil")
	}
	if notif.EmailService == nil {
		t.Error("EmailService should be injected via inject tag since New left it nil")
	}
}

func TestEndUser_SingletonInjection(t *testing.T) {
	c := New()

	singleton := &MockEmailService{}
	c.Singleton((*EmailService)(nil), func(a *App) interface{} {
		return singleton
	})
	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Test"}
	})

	controller1 := c.Make(&UserController{}).(*UserController)
	controller2 := c.Make(&UserController{}).(*UserController)

	if controller1.EmailService != controller2.EmailService {
		t.Error("Both controllers should receive the same singleton EmailService")
	}
	if controller1.EmailService != singleton {
		t.Error("EmailService should be the singleton instance")
	}
}

func TestEndUser_WithContextualInjection(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Default"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	c.When(&UserController{}).Needs((*UserService)(nil)).Give(func(a *App) interface{} {
		return &MockUserService{Name: "Special"}
	})

	controller := c.Make(&UserController{}).(*UserController)

	if controller.UserService.GetUserName() != "Special" {
		t.Errorf("Expected Special user service, got %s", controller.UserService.GetUserName())
	}
}

func TestEndUser_MakeWithOverrides(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Default"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	customService := &MockUserService{Name: "Custom"}
	controller := c.MakeWith(&UserController{}, map[string]interface{}{
		"UserService": customService,
	}).(*UserController)

	if controller.UserService.GetUserName() != "Custom" {
		t.Errorf("Expected Custom user service, got %s", controller.UserService.GetUserName())
	}
	if controller.Container == nil {
		t.Error("Container should still be injected via di tag")
	}
}

func TestEndUser_NamedBindings(t *testing.T) {
	c := New()

	c.Bind("primaryUser", func(a *App) interface{} {
		return &MockUserService{Name: "Primary"}
	})
	c.Bind("secondaryUser", func(a *App) interface{} {
		return &MockUserService{Name: "Secondary"}
	})

	primary := c.Make("primaryUser").(*MockUserService)
	secondary := c.Make("secondaryUser").(*MockUserService)

	if primary.Name != "Primary" {
		t.Errorf("Expected Primary, got %s", primary.Name)
	}
	if secondary.Name != "Secondary" {
		t.Errorf("Expected Secondary, got %s", secondary.Name)
	}
}

func TestEndUser_FactoryBinding(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Factory"}
	})
	c.Bind((*EmailService)(nil), &MockEmailService{})

	controller := c.Make(&UserController{}).(*UserController)

	if controller.UserService.GetUserName() != "Factory" {
		t.Errorf("Expected Factory, got %s", controller.UserService.GetUserName())
	}
	if controller.Container == nil {
		t.Error("Container should be injected via di tag on factory result")
	}
}

func TestEndUser_InterfaceDiTag(t *testing.T) {
	c := New()

	type ServiceWithContainer struct {
		Container AppInterface `di:""`
	}

	svc := c.Make(&ServiceWithContainer{}).(*ServiceWithContainer)

	if svc.Container == nil {
		t.Error("Container should be injected into AppInterface field via di tag")
	}
}

func TestEndUser_ComplexDependencyChain(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Chain"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	repo := c.Make(&UserRepository{}).(*UserRepository)
	if repo.Config == nil {
		t.Error("ConfigService should be auto-created and injected")
	}
	if repo.Config.AppName != "MyApp" {
		t.Errorf("Expected MyApp, got %s", repo.Config.AppName)
	}

	controller := c.Make(&UserController{}).(*UserController)
	if controller.UserService == nil {
		t.Error("UserService should be injected")
	}
	if controller.Container == nil {
		t.Error("Container should be injected via di tag")
	}
}

func TestEndUser_DefaultApp(t *testing.T) {
	Default()

	c := Default()
	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Default"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	controller := c.Make(&UserController{}).(*UserController)

	if controller.UserService.GetUserName() != "Default" {
		t.Errorf("Expected Default, got %s", controller.UserService.GetUserName())
	}
}

func TestEndUser_NamedAppInstances(t *testing.T) {
	app1 := Default("api")
	app2 := Default("worker")

	app1.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "API"}
	})
	app2.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Worker"}
	})

	type SimpleController struct {
		UserService UserService `inject:""`
	}

	c1 := app1.Make(&SimpleController{}).(*SimpleController)
	c2 := app2.Make(&SimpleController{}).(*SimpleController)

	if c1.UserService.GetUserName() != "API" {
		t.Errorf("Expected API, got %s", c1.UserService.GetUserName())
	}
	if c2.UserService.GetUserName() != "Worker" {
		t.Errorf("Expected Worker, got %s", c2.UserService.GetUserName())
	}
}

func TestEndUser_RemoveBinding(t *testing.T) {
	c := New()

	c.Bind("temp", func(a *App) interface{} {
		return &MockUserService{Name: "Temp"}
	})
	svc := c.Make("temp").(*MockUserService)
	if svc.Name != "Temp" {
		t.Errorf("Expected Temp, got %s", svc.Name)
	}

	c.Bind("temp", nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when making removed binding")
		}
	}()
	c.Make("temp")
}

type CircularA struct {
	B *CircularB `inject:""`
}

type CircularB struct {
	A *CircularA `inject:""`
}

type CircularSelfRef struct {
	Self *CircularSelfRef `inject:""`
}

type CircularChainA struct {
	B *CircularChainB `inject:""`
}

type CircularChainB struct {
	C *CircularChainC `inject:""`
}

type CircularChainC struct {
	A *CircularChainA `inject:""`
}

func TestEndUser_CircularDependency_Direct(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected panic for circular dependency")
		}
		msg, ok := r.(string)
		if !ok || len(msg) == 0 {
			t.Fatalf("Expected string panic message, got %v", r)
		}
	}()

	c := New()
	c.Make(&CircularA{})
}

func TestEndUser_CircularDependency_SelfReference(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected panic for self-referencing dependency")
		}
	}()

	c := New()
	c.Make(&CircularSelfRef{})
}

func TestEndUser_CircularDependency_ThreeWayChain(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Expected panic for three-way circular dependency")
		}
	}()

	c := New()
	c.Make(&CircularChainA{})
}

func TestEndUser_CircularDependency_DoesNotFalsePositive(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Alice"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	controller := c.Make(&UserController{}).(*UserController)
	if controller.UserService == nil {
		t.Error("UserService should be injected")
	}

	controller2 := c.Make(&UserController{}).(*UserController)
	if controller2.UserService == nil {
		t.Error("Second Make should also work (resolving map cleaned up)")
	}
}

func TestEndUser_FuncReturnTypeValidation_IncompatibleConcreteReturn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic for incompatible concrete return type")
		}
	}()

	c := New()
	c.Bind((*UserService)(nil), func(a *App) *MockEmailService {
		return &MockEmailService{}
	})
}

func TestEndUser_FuncReturnTypeValidation_IncompatibleRuntimeReturn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic for incompatible runtime return value")
		}
	}()

	c := New()
	// interface{} return type passes bind-time validation, but returns wrong type at runtime
	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	c.Make((*UserService)(nil))
}

func TestEndUser_ConcurrentMake(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Concurrent"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Concurrent Make panicked: %v", r)
				}
				done <- true
			}()
			controller := c.Make(&UserController{}).(*UserController)
			if controller.UserService == nil {
				t.Error("UserService should be injected in concurrent Make")
			}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestEndUser_ConcurrentBindAndMake(t *testing.T) {
	c := New()

	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Initial"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			c.Make(&UserController{})
		}()
	}
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			c.Bind((*UserService)(nil), func(a *App) interface{} {
				return &MockUserService{Name: "Rebound"}
			})
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestEndUser_BindFuncCallsMake(t *testing.T) {
	c := New()

	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})
	c.Bind((*UserService)(nil), func(a *App) interface{} {
		email := a.Make((*EmailService)(nil))
		if email == nil {
			t.Error("BindFunc should be able to call Make on the container")
		}
		return &MockUserService{Name: "FromBindFunc"}
	})

	svc := c.Make((*UserService)(nil)).(*MockUserService)
	if svc.Name != "FromBindFunc" {
		t.Errorf("Expected FromBindFunc, got %s", svc.Name)
	}
}

type StructWithNewAndEmptyInjectPrimitive struct {
	Count int `inject:""`
}

func (s StructWithNewAndEmptyInjectPrimitive) New() *StructWithNewAndEmptyInjectPrimitive {
	return &StructWithNewAndEmptyInjectPrimitive{Count: 42}
}

func TestEndUser_ProcessStructTags_EmptyInjectOnPrimitivePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic for inject:\"\" on primitive field after New() constructor")
		}
	}()

	c := New()
	c.Make(&StructWithNewAndEmptyInjectPrimitive{})
}

func TestEndUser_NamedPrimitiveBinding(t *testing.T) {
	c := New()

	c.Bind("port", 8080)
	c.Bind("debug", true)
	c.Bind("rate", 3.14)

	if c.Make("port") != 8080 {
		t.Errorf("Expected 8080, got %v", c.Make("port"))
	}
	if c.Make("debug") != true {
		t.Errorf("Expected true, got %v", c.Make("debug"))
	}
	if c.Make("rate") != 3.14 {
		t.Errorf("Expected 3.14, got %v", c.Make("rate"))
	}
}

// DiTag injecting a concrete type (not the container) via makeByHints path.
type DiTagConcreteService struct {
	Config *ConfigService `di:""`
}

func TestEndUser_DiTag_InjectsConcreteType(t *testing.T) {
	c := New()

	svc := c.Make(&DiTagConcreteService{}).(*DiTagConcreteService)

	if svc.Config == nil {
		t.Fatal("Config should be injected via di tag")
	}
	if svc.Config.AppName != "MyApp" {
		t.Errorf("Expected MyApp, got %s", svc.Config.AppName)
	}
}

// DiTag injecting an interface type from the container.
type DiTagInterfaceService struct {
	UserSvc UserService `di:""`
}

func TestEndUser_DiTag_InjectsInterface(t *testing.T) {
	c := New()
	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "DiInjected"}
	})

	svc := c.Make(&DiTagInterfaceService{}).(*DiTagInterfaceService)

	if svc.UserSvc == nil {
		t.Fatal("UserSvc should be injected via di tag")
	}
	if svc.UserSvc.GetUserName() != "DiInjected" {
		t.Errorf("Expected DiInjected, got %s", svc.UserSvc.GetUserName())
	}
}

// DiTag on a New-bearing type respects the zero-value guard (New set it → di skips).
type DiTagNewPreservesValue struct {
	Config *ConfigService `di:""`
}

var presetConfig = &ConfigService{AppName: "Preset"}

func (d DiTagNewPreservesValue) New() *DiTagNewPreservesValue {
	return &DiTagNewPreservesValue{Config: presetConfig}
}

func TestEndUser_DiTag_DoesNotOverwriteNewValue(t *testing.T) {
	c := New()

	svc := c.Make(&DiTagNewPreservesValue{}).(*DiTagNewPreservesValue)

	if svc.Config != presetConfig {
		t.Error("di tag should not overwrite value set by New()")
	}
}

// DiTag on a New-bearing type injects when New left the field nil.
type DiTagNewFillsNil struct {
	Config *ConfigService `di:""`
}

func (d DiTagNewFillsNil) New() *DiTagNewFillsNil {
	return &DiTagNewFillsNil{}
}

func TestEndUser_DiTag_FillsNilAfterNew(t *testing.T) {
	c := New()

	svc := c.Make(&DiTagNewFillsNil{}).(*DiTagNewFillsNil)

	if svc.Config == nil {
		t.Fatal("Config should be injected via di tag when New left it nil")
	}
	if svc.Config.AppName != "MyApp" {
		t.Errorf("Expected MyApp, got %s", svc.Config.AppName)
	}
}

func TestEndUser_DiTagWithStructBinding(t *testing.T) {
	c := New()

	c.Bind(&UserController{}, func(a *App) interface{} {
		return &UserController{}
	})
	c.Bind((*UserService)(nil), func(a *App) interface{} {
		return &MockUserService{Name: "Bound"}
	})
	c.Bind((*EmailService)(nil), func(a *App) interface{} {
		return &MockEmailService{}
	})

	controller := c.Make(&UserController{}).(*UserController)

	if controller.Container == nil {
		t.Error("Container should be injected via di tag on bound struct")
	}
	if controller.UserService == nil {
		t.Error("UserService should be injected via inject tag")
	}
}

type NilReturningStruct struct {
	Name string `inject:"test"`
}

func (n NilReturningStruct) New() *NilReturningStruct {
	return nil
}

func TestEndUser_NilPtrFromNew_DoesNotPanic(t *testing.T) {
	c := New()
	result := c.Make(&NilReturningStruct{})
	if result != nil && result.(*NilReturningStruct) != nil {
		// acceptable — just must not panic
	}
}

func TestEndUser_NilPtrFromBindFunc_DoesNotPanic(t *testing.T) {
	c := New()
	c.Bind(&NilReturningStruct{}, func(a *App) interface{} {
		return (*NilReturningStruct)(nil)
	})
	result := c.Make(&NilReturningStruct{})
	if result != nil && result.(*NilReturningStruct) != nil {
		// acceptable — just must not panic
	}
}

type Context interface {
	GetMessage() string
	Reply(message string) string
}

type MessageHandlerFunc = func(Context)
type ErrorsHandler func(err error)
type DebugHandler func(format string, args ...any)
type Middleware func(next HandlerFunc) HandlerFunc
type HandlerFunc func(ctx context.Context, bot *Bot, update string)
type MatchFunc func(update string) bool

type handler struct {
	f MessageHandlerFunc
	i string
}

type Bot struct {
	lastUpdateID int64

	url                string
	token              string
	pollTimeout        time.Duration
	skipGetMe          bool
	webhookSecretToken string
	testEnvironment    bool
	workers            int
	notAsyncHandlers   bool

	defaultHandlerFunc HandlerFunc

	errorsHandler ErrorsHandler
	debugHandler  DebugHandler

	middlewares []Middleware

	handlersMx sync.RWMutex
	handlers   []handler

	isDebug          bool
	checkInitTimeout time.Duration
}

type Config struct {
	Client string
	Secret string
}

type Engine struct {
	bot      *Bot
	cancel   context.CancelFunc
	ctx      context.Context
	handlers map[string][]handler
	lock     *sync.RWMutex
}

func TestEndUser_MakePointerStruct(t *testing.T) {
	c := New()

	app.When((*Engine)(nil)).Needs((*Config)(nil)).Give(&Config{
		Client: "CLIENT_ID",
		Secret: "SECRET_STRING",
	}).Singleton()

	e := c.Make((*Engine)(nil)).(*Engine)

	if e == nil {
		t.Fatal("Expected Engine instance, got nil")
	}
}
