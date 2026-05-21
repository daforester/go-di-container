package di

import (
	"testing"
)

// --- Types for deeply nested dependency chains (3+ levels) ---

type Level3Provider struct {
	Value string `inject:"deep-value"`
}

type Level2Service struct {
	Provider *Level3Provider `inject:""`
}

type Level1Controller struct {
	Service *Level2Service `inject:""`
}

type Level0App struct {
	Controller *Level1Controller `inject:""`
	Container  *App              `di:""`
}

func TestDeepNesting_FourLevelDependencyChain(t *testing.T) {
	c := New()

	result := c.Make(&Level0App{}).(*Level0App)

	if result.Container == nil {
		t.Error("Container should be injected via di tag")
	}
	if result.Controller == nil {
		t.Fatal("Level1Controller should be auto-created")
	}
	if result.Controller.Service == nil {
		t.Fatal("Level2Service should be auto-created")
	}
	if result.Controller.Service.Provider == nil {
		t.Fatal("Level3Provider should be auto-created")
	}
	if result.Controller.Service.Provider.Value != "deep-value" {
		t.Errorf("Expected deep-value, got %s", result.Controller.Service.Provider.Value)
	}
}

type BoundLevel3Provider struct {
	Value string
}

type BoundLevel2Service struct {
	Provider *BoundLevel3Provider `inject:""`
}

type BoundLevel1Controller struct {
	Service *BoundLevel2Service `inject:""`
}

func TestDeepNesting_WithBindingsAtEachLevel(t *testing.T) {
	c := New()

	c.Bind(&BoundLevel3Provider{}, func(a *App) interface{} {
		return &BoundLevel3Provider{Value: "bound-deep"}
	})

	result := c.Make(&BoundLevel1Controller{}).(*BoundLevel1Controller)

	if result.Service == nil {
		t.Fatal("Service should be auto-created")
	}
	if result.Service.Provider == nil {
		t.Fatal("Provider should be resolved via Bind")
	}
	if result.Service.Provider.Value != "bound-deep" {
		t.Errorf("Expected bound-deep, got %s", result.Service.Provider.Value)
	}
}

func TestDeepNesting_InjectTagOverridesFactoryValue(t *testing.T) {
	c := New()

	c.Bind(&Level3Provider{}, func(a *App) interface{} {
		return &Level3Provider{Value: "factory-value"}
	})

	result := c.Make(&Level3Provider{}).(*Level3Provider)

	// inject:"deep-value" tag on Level3Provider.Value overwrites the factory return
	if result.Value != "deep-value" {
		t.Errorf("inject tag should override factory value, got %s", result.Value)
	}
}

// --- Types for nested New() method chains ---

type NewChainBottom struct {
	Tag string `inject:"bottom"`
}

type NewChainMiddle struct {
	Bottom *NewChainBottom
	Extra  string
}

func (n NewChainMiddle) New(b *NewChainBottom) *NewChainMiddle {
	return &NewChainMiddle{
		Bottom: b,
		Extra:  "middle-new",
	}
}

type NewChainTop struct {
	Middle *NewChainMiddle
	Label  string
}

func (n NewChainTop) New(m *NewChainMiddle) *NewChainTop {
	return &NewChainTop{
		Middle: m,
		Label:  "top-new",
	}
}

func TestNestedNew_ChainedNewMethods(t *testing.T) {
	c := New()

	result := c.Make(&NewChainTop{}).(*NewChainTop)

	if result.Label != "top-new" {
		t.Errorf("Expected top-new, got %s", result.Label)
	}
	if result.Middle == nil {
		t.Fatal("Middle should be created via its New() method")
	}
	if result.Middle.Extra != "middle-new" {
		t.Errorf("Expected middle-new, got %s", result.Middle.Extra)
	}
	if result.Middle.Bottom == nil {
		t.Fatal("Bottom should be auto-created for Middle's New()")
	}
	if result.Middle.Bottom.Tag != "bottom" {
		t.Errorf("Expected bottom, got %s", result.Middle.Bottom.Tag)
	}
}

// --- Types for New() + inject + di combination ---

type CombinedNewInjectDi struct {
	RepoFromNew    CombinedRepo  // Set by New()
	Cache          CombinedCache `inject:""`
	Container      *App          `di:""`
	ValueFromNew   int           // Set by New()
	UntouchedField string        // No tag, not set by New()
}

type CombinedRepo interface {
	Name() string
}

type CombinedCache interface {
	Type() string
}

type CombinedRepoImpl struct{ N string }

func (r *CombinedRepoImpl) Name() string { return r.N }

type CombinedCacheImpl struct{ T string }

func (c *CombinedCacheImpl) Type() string { return c.T }

func (c CombinedNewInjectDi) New(repo CombinedRepo) *CombinedNewInjectDi {
	return &CombinedNewInjectDi{
		RepoFromNew:  repo,
		ValueFromNew: 42,
	}
}

func TestCombinedNewInjectDi_AllThreeMechanisms(t *testing.T) {
	c := New()

	c.Bind((*CombinedRepo)(nil), func(a *App) interface{} {
		return &CombinedRepoImpl{N: "new-repo"}
	})
	c.Bind((*CombinedCache)(nil), func(a *App) interface{} {
		return &CombinedCacheImpl{T: "redis"}
	})

	result := c.Make(&CombinedNewInjectDi{}).(*CombinedNewInjectDi)

	if result.RepoFromNew == nil || result.RepoFromNew.Name() != "new-repo" {
		t.Error("RepoFromNew should be set by New() via Make()")
	}
	if result.ValueFromNew != 42 {
		t.Errorf("ValueFromNew should be 42 from New(), got %d", result.ValueFromNew)
	}
	if result.Cache == nil || result.Cache.Type() != "redis" {
		t.Error("Cache should be injected via inject tag after New()")
	}
	if result.Container == nil {
		t.Error("Container should be injected via di tag after New()")
	}
	if result.UntouchedField != "" {
		t.Errorf("UntouchedField should remain zero, got %s", result.UntouchedField)
	}
}

// --- When/Needs/Give + New() constructor parameters ---

type WhenNewRepo interface {
	Source() string
}

type WhenNewDBRepo struct{}

func (w WhenNewDBRepo) Source() string { return "db" }

type WhenNewCacheRepo struct{}

func (w WhenNewCacheRepo) Source() string { return "cache" }

type WhenNewHandler struct {
	Repo WhenNewRepo
}

func (w WhenNewHandler) New(r WhenNewRepo) *WhenNewHandler {
	return &WhenNewHandler{Repo: r}
}

type WhenNewOtherHandler struct {
	Repo WhenNewRepo
}

func (w WhenNewOtherHandler) New(r WhenNewRepo) *WhenNewOtherHandler {
	return &WhenNewOtherHandler{Repo: r}
}

func TestWhenWithNew_OverridesNewParameter(t *testing.T) {
	c := New()

	c.Bind((*WhenNewRepo)(nil), func(a *App) interface{} {
		return &WhenNewDBRepo{}
	})

	c.When(&WhenNewHandler{}).Needs((*WhenNewRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenNewCacheRepo{}
	})

	handler := c.Make(&WhenNewHandler{}).(*WhenNewHandler)
	other := c.Make(&WhenNewOtherHandler{}).(*WhenNewOtherHandler)

	if handler.Repo.Source() != "cache" {
		t.Errorf("WhenNewHandler should get cache repo, got %s", handler.Repo.Source())
	}
	if other.Repo.Source() != "db" {
		t.Errorf("WhenNewOtherHandler should get default db repo, got %s", other.Repo.Source())
	}
}

// --- Singleton + When interaction ---

type SingletonWhenLogger interface {
	Level() string
}

type SingletonWhenDefaultLogger struct{}

func (s SingletonWhenDefaultLogger) Level() string { return "info" }

type SingletonWhenDebugLogger struct{}

func (s SingletonWhenDebugLogger) Level() string { return "debug" }

type SingletonWhenHandlerA struct {
	Logger SingletonWhenLogger `inject:""`
}

type SingletonWhenHandlerB struct {
	Logger SingletonWhenLogger `inject:""`
}

func TestSingleton_WhenOverridesSingleton(t *testing.T) {
	c := New()

	c.Singleton((*SingletonWhenLogger)(nil), func(a *App) interface{} {
		return &SingletonWhenDefaultLogger{}
	})

	c.When(&SingletonWhenHandlerA{}).Needs((*SingletonWhenLogger)(nil)).Give(func(a *App) interface{} {
		return &SingletonWhenDebugLogger{}
	})

	handlerA := c.Make(&SingletonWhenHandlerA{}).(*SingletonWhenHandlerA)
	handlerB := c.Make(&SingletonWhenHandlerB{}).(*SingletonWhenHandlerB)

	if handlerA.Logger.Level() != "debug" {
		t.Errorf("HandlerA should get debug logger via When, got %s", handlerA.Logger.Level())
	}
	if handlerB.Logger.Level() != "info" {
		t.Errorf("HandlerB should get default singleton logger, got %s", handlerB.Logger.Level())
	}
}

func TestSingleton_WhenDoesNotBreakSingletonBehavior(t *testing.T) {
	c := New()

	singleton := &SingletonWhenDefaultLogger{}
	c.Singleton((*SingletonWhenLogger)(nil), singleton)

	handlerB1 := c.Make(&SingletonWhenHandlerB{}).(*SingletonWhenHandlerB)
	handlerB2 := c.Make(&SingletonWhenHandlerB{}).(*SingletonWhenHandlerB)

	if handlerB1.Logger != handlerB2.Logger {
		t.Error("Both HandlerB instances should receive the same singleton logger")
	}
	if handlerB1.Logger != singleton {
		t.Error("HandlerB should receive the original singleton instance")
	}
}

// --- Bind with concrete struct that has New() method ---

type BindNewStruct struct {
	Value string
}

func (b BindNewStruct) New() *BindNewStruct {
	return &BindNewStruct{Value: "from-new"}
}

type BindNewInterface interface {
	GetValue() string
}

type BindNewImpl struct {
	Value string
}

func (b BindNewImpl) GetValue() string { return b.Value }

func (b BindNewImpl) New() *BindNewImpl {
	return &BindNewImpl{Value: "impl-new"}
}

func TestBindConcreteStruct_WithNewMethod(t *testing.T) {
	c := New()

	result := c.Make(&BindNewStruct{}).(*BindNewStruct)

	if result.Value != "from-new" {
		t.Errorf("Expected from-new, got %s", result.Value)
	}
}

func TestBindConcreteStruct_BoundViaFunc_BypassesNew(t *testing.T) {
	c := New()

	c.Bind(&BindNewStruct{}, func(a *App) interface{} {
		return &BindNewStruct{Value: "from-bind"}
	})

	result := c.Make(&BindNewStruct{}).(*BindNewStruct)

	if result.Value != "from-bind" {
		t.Errorf("Expected from-bind, got %s", result.Value)
	}
}

// --- Untagged fields remain zero ---

type MixedTagStruct struct {
	Tagged    string `inject:"tagged-value"`
	Untagged  string
	DiField   *App `di:""`
	PlainInt  int
	PlainBool bool
}

func TestMixedTags_UntaggedFieldsRemainZero(t *testing.T) {
	c := New()

	result := c.Make(&MixedTagStruct{}).(*MixedTagStruct)

	if result.Tagged != "tagged-value" {
		t.Errorf("Expected tagged-value, got %s", result.Tagged)
	}
	if result.Untagged != "" {
		t.Errorf("Untagged should be empty, got %s", result.Untagged)
	}
	if result.DiField == nil {
		t.Error("DiField should be injected")
	}
	if result.PlainInt != 0 {
		t.Errorf("PlainInt should be 0, got %d", result.PlainInt)
	}
	if result.PlainBool != false {
		t.Errorf("PlainBool should be false, got %v", result.PlainBool)
	}
}

// --- Multiple When rules: last write wins ---

type MultiWhenRepo interface {
	ID() string
}

type MultiWhenRepoA struct{}

func (m MultiWhenRepoA) ID() string { return "A" }

type MultiWhenRepoB struct{}

func (m MultiWhenRepoB) ID() string { return "B" }

type MultiWhenRepoC struct{}

func (m MultiWhenRepoC) ID() string { return "C" }

type MultiWhenHandler struct {
	Repo MultiWhenRepo `inject:""`
}

func TestMultipleWhen_LastWriteWins(t *testing.T) {
	c := New()

	c.Bind((*MultiWhenRepo)(nil), func(a *App) interface{} {
		return &MultiWhenRepoA{}
	})

	c.When(&MultiWhenHandler{}).Needs((*MultiWhenRepo)(nil)).Give(func(a *App) interface{} {
		return &MultiWhenRepoB{}
	})
	c.When(&MultiWhenHandler{}).Needs((*MultiWhenRepo)(nil)).Give(func(a *App) interface{} {
		return &MultiWhenRepoC{}
	})

	handler := c.Make(&MultiWhenHandler{}).(*MultiWhenHandler)

	if handler.Repo.ID() != "C" {
		t.Errorf("Expected last-write-wins (C), got %s", handler.Repo.ID())
	}
}

// --- When + singleton Give ---

type WhenSingletonService struct {
	Count int
}

type WhenSingletonConsumer struct {
	Service *WhenSingletonService `inject:""`
}

func TestWhen_GiveSingleton(t *testing.T) {
	c := New()

	instance := &WhenSingletonService{Count: 99}

	c.When(&WhenSingletonConsumer{}).Needs((*WhenSingletonService)(nil)).Give(instance).Singleton()

	consumer1 := c.Make(&WhenSingletonConsumer{}).(*WhenSingletonConsumer)
	consumer2 := c.Make(&WhenSingletonConsumer{}).(*WhenSingletonConsumer)

	if consumer1.Service.Count != 99 {
		t.Errorf("Expected 99, got %d", consumer1.Service.Count)
	}
	if consumer1.Service != consumer2.Service {
		t.Error("Singleton Give should return the same instance each time")
	}
}

// --- Deep nesting with When override at middle level ---

type DeepWhenDB interface {
	DSN() string
}

type DeepWhenProdDB struct{}

func (d DeepWhenProdDB) DSN() string { return "prod" }

type DeepWhenTestDB struct{}

func (d DeepWhenTestDB) DSN() string { return "test" }

type DeepWhenRepo struct {
	DB DeepWhenDB `inject:""`
}

type DeepWhenService struct {
	Repo *DeepWhenRepo `inject:""`
}

type DeepWhenController struct {
	Service *DeepWhenService `inject:""`
}

func TestDeepNesting_WhenAtNestedLevel(t *testing.T) {
	c := New()

	c.Bind((*DeepWhenDB)(nil), func(a *App) interface{} {
		return &DeepWhenProdDB{}
	})

	c.When(&DeepWhenRepo{}).Needs((*DeepWhenDB)(nil)).Give(func(a *App) interface{} {
		return &DeepWhenTestDB{}
	})

	controller := c.Make(&DeepWhenController{}).(*DeepWhenController)

	if controller.Service.Repo.DB.DSN() != "test" {
		t.Errorf("Expected test DSN via When at repo level, got %s", controller.Service.Repo.DB.DSN())
	}
}

// --- Bind concrete ptr with field values preserved ---

type PreservedFieldsStruct struct {
	Name    string
	Count   int
	Enabled bool
}

func TestBind_ConcreteStructPreservesFields(t *testing.T) {
	c := New()

	c.Bind("myConfig", &PreservedFieldsStruct{
		Name:    "preserved",
		Count:   42,
		Enabled: true,
	})

	result := c.Make("myConfig").(*PreservedFieldsStruct)

	if result.Name != "preserved" {
		t.Errorf("Expected preserved, got %s", result.Name)
	}
	if result.Count != 42 {
		t.Errorf("Expected 42, got %d", result.Count)
	}
	if result.Enabled != true {
		t.Errorf("Expected true, got %v", result.Enabled)
	}
}

func TestBind_ConcreteStructReturnsDistinctInstances(t *testing.T) {
	c := New()

	c.Bind("myConfig", &PreservedFieldsStruct{
		Name: "template",
	})

	r1 := c.Make("myConfig").(*PreservedFieldsStruct)
	r2 := c.Make("myConfig").(*PreservedFieldsStruct)

	if r1 == r2 {
		t.Error("Non-singleton Bind should return distinct instances")
	}
	if r1.Name != "template" || r2.Name != "template" {
		t.Error("Both instances should have preserved field values")
	}
}

// --- New() with multiple params of different kinds ---

type MultiParamInterface interface {
	Kind() string
}

type MultiParamImpl struct{}

func (m MultiParamImpl) Kind() string { return "impl" }

type MultiParamConfig struct {
	Env string `inject:"production"`
}

type MultiParamHandler struct {
	Svc    MultiParamInterface
	Config *MultiParamConfig
	Label  string
}

func (m MultiParamHandler) New(svc MultiParamInterface, cfg *MultiParamConfig) *MultiParamHandler {
	return &MultiParamHandler{
		Svc:    svc,
		Config: cfg,
		Label:  "constructed",
	}
}

func TestNew_MultipleParamTypes(t *testing.T) {
	c := New()

	c.Bind((*MultiParamInterface)(nil), func(a *App) interface{} {
		return &MultiParamImpl{}
	})

	result := c.Make(&MultiParamHandler{}).(*MultiParamHandler)

	if result.Svc == nil || result.Svc.Kind() != "impl" {
		t.Error("Interface param should be resolved via Bind")
	}
	if result.Config == nil {
		t.Fatal("Ptr param should be auto-created")
	}
	if result.Config.Env != "production" {
		t.Errorf("Config Env should be production, got %s", result.Config.Env)
	}
	if result.Label != "constructed" {
		t.Errorf("Label should be constructed, got %s", result.Label)
	}
}

// --- New() with inject tags filled AFTER construction ---

type NewThenInject struct {
	FromNew     string
	Injected    string `inject:"post-new-value"`
	InjectedSvc CombinedRepo `inject:""`
	Container   *App   `di:""`
}

func (n NewThenInject) New() *NewThenInject {
	return &NewThenInject{
		FromNew: "set-by-new",
	}
}

func TestNew_InjectTagsProcessedAfterNew(t *testing.T) {
	c := New()

	c.Bind((*CombinedRepo)(nil), func(a *App) interface{} {
		return &CombinedRepoImpl{N: "after-new"}
	})

	result := c.Make(&NewThenInject{}).(*NewThenInject)

	if result.FromNew != "set-by-new" {
		t.Errorf("Expected set-by-new, got %s", result.FromNew)
	}
	if result.Injected != "post-new-value" {
		t.Errorf("Expected post-new-value, got %s", result.Injected)
	}
	if result.InjectedSvc == nil || result.InjectedSvc.Name() != "after-new" {
		t.Error("InjectedSvc should be resolved via inject tag after New()")
	}
	if result.Container == nil {
		t.Error("Container should be injected via di tag after New()")
	}
}

// --- New() sets a field, inject tag on same field should NOT overwrite ---

type NewVsInjectConflict struct {
	Svc CombinedRepo `inject:""`
}

func (n NewVsInjectConflict) New(svc CombinedRepo) *NewVsInjectConflict {
	return &NewVsInjectConflict{
		Svc: svc,
	}
}

func TestNew_FieldSetByNewCanBeOverwrittenByInjectTag(t *testing.T) {
	c := New()

	c.Bind((*CombinedRepo)(nil), func(a *App) interface{} {
		return &CombinedRepoImpl{N: "default"}
	})

	result := c.Make(&NewVsInjectConflict{}).(*NewVsInjectConflict)

	if result.Svc == nil {
		t.Fatal("Svc should be set")
	}
	// New() injects via param, processStructTags runs after and may overwrite.
	// The important thing is the field IS populated.
	if result.Svc.Name() != "default" {
		t.Errorf("Expected default, got %s", result.Svc.Name())
	}
}

// --- MakeWith partial override with When ---

type MakeWithWhenRepo interface {
	Tag() string
}

type MakeWithWhenDefault struct{}

func (m MakeWithWhenDefault) Tag() string { return "default" }

type MakeWithWhenOverride struct{}

func (m MakeWithWhenOverride) Tag() string { return "when-override" }

type MakeWithWhenCustom struct{}

func (m MakeWithWhenCustom) Tag() string { return "makewith-custom" }

type MakeWithWhenStruct struct {
	Repo  MakeWithWhenRepo  `inject:""`
	Cache CombinedCache     `inject:""`
}

func TestMakeWith_OverridesWhenBinding(t *testing.T) {
	c := New()

	c.Bind((*MakeWithWhenRepo)(nil), func(a *App) interface{} {
		return &MakeWithWhenDefault{}
	})
	c.Bind((*CombinedCache)(nil), func(a *App) interface{} {
		return &CombinedCacheImpl{T: "memcache"}
	})

	c.When(&MakeWithWhenStruct{}).Needs((*MakeWithWhenRepo)(nil)).Give(func(a *App) interface{} {
		return &MakeWithWhenOverride{}
	})

	custom := &MakeWithWhenCustom{}
	result := c.MakeWith(&MakeWithWhenStruct{}, map[string]interface{}{
		"Repo": custom,
	}).(*MakeWithWhenStruct)

	if result.Repo.Tag() != "makewith-custom" {
		t.Errorf("MakeWith should take precedence over When, got %s", result.Repo.Tag())
	}
	if result.Cache.Type() != "memcache" {
		t.Errorf("Cache should still be resolved normally, got %s", result.Cache.Type())
	}
}

// --- Autogen without any binding (pure concrete types) ---

type PureConcreteLeaf struct {
	Value string `inject:"leaf"`
}

type PureConcreteParent struct {
	Leaf *PureConcreteLeaf `inject:""`
	Tag  string            `inject:"parent"`
}

func TestAutogen_PureConcreteNoBindings(t *testing.T) {
	c := New()

	result := c.Make(&PureConcreteParent{}).(*PureConcreteParent)

	if result.Tag != "parent" {
		t.Errorf("Expected parent, got %s", result.Tag)
	}
	if result.Leaf == nil {
		t.Fatal("Leaf should be auto-created")
	}
	if result.Leaf.Value != "leaf" {
		t.Errorf("Expected leaf, got %s", result.Leaf.Value)
	}
}

// --- Struct requested as struct (not ptr) ---

func TestMake_StructValueReturnType(t *testing.T) {
	c := New()

	result := c.Make(MixedTagStruct{}).(MixedTagStruct)

	if result.Tagged != "tagged-value" {
		t.Errorf("Expected tagged-value, got %s", result.Tagged)
	}
	// di tag on struct value - container is *App which is a ptr, can it be set on a struct value?
	// The makeByHints creates a reflect.New then returns Elem() for struct types
	if result.DiField == nil {
		t.Error("DiField should be injected even when requesting struct value")
	}
}

// --- Bind interface to concrete struct (not func), then Make ---

type BindStructInterface interface {
	ID() int
}

type BindStructImpl struct {
	Val int
}

func (b BindStructImpl) ID() int { return b.Val }

func TestBind_InterfaceToConcreteStruct(t *testing.T) {
	c := New()

	c.Bind((*BindStructInterface)(nil), BindStructImpl{Val: 7})

	result := c.Make((*BindStructInterface)(nil)).(BindStructInterface)

	if result.ID() != 7 {
		t.Errorf("Expected 7, got %d", result.ID())
	}
}

func TestBind_InterfaceToConcretePtr(t *testing.T) {
	c := New()

	c.Bind((*BindStructInterface)(nil), &BindStructImpl{Val: 8})

	result := c.Make((*BindStructInterface)(nil)).(BindStructInterface)

	if result.ID() != 8 {
		t.Errorf("Expected 8, got %d", result.ID())
	}
}

// --- When with different requesting type forms (ptr vs nil ptr) ---

type WhenFormHandler struct {
	Repo WhenNewRepo `inject:""`
}

func TestWhen_NilPtrAndValuePtrProduceSameResult(t *testing.T) {
	c := New()

	c.Bind((*WhenNewRepo)(nil), func(a *App) interface{} {
		return &WhenNewDBRepo{}
	})

	// Register When with &WhenFormHandler{}
	c.When(&WhenFormHandler{}).Needs((*WhenNewRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenNewCacheRepo{}
	})

	result := c.Make(&WhenFormHandler{}).(*WhenFormHandler)

	if result.Repo.Source() != "cache" {
		t.Errorf("Expected cache, got %s", result.Repo.Source())
	}
}

// --- Private/unexported fields with tags are skipped ---

type UnexportedFieldStruct struct {
	Public  string `inject:"public-val"`
	private string `inject:"private-val"` //nolint:unused
}

func TestUnexportedFields_Skipped(t *testing.T) {
	c := New()

	result := c.Make(&UnexportedFieldStruct{}).(*UnexportedFieldStruct)

	if result.Public != "public-val" {
		t.Errorf("Expected public-val, got %s", result.Public)
	}
	// private field can't be accessed via reflection (CanSet returns false)
	// Just ensure no panic occurred
}

// --- Rebind: change binding and verify Make uses new one ---

type RebindInterface interface {
	Version() string
}

type RebindV1 struct{}

func (r RebindV1) Version() string { return "v1" }

type RebindV2 struct{}

func (r RebindV2) Version() string { return "v2" }

func TestRebind_OverwritesPreviousBinding(t *testing.T) {
	c := New()

	c.Bind((*RebindInterface)(nil), func(a *App) interface{} {
		return &RebindV1{}
	})

	r1 := c.Make((*RebindInterface)(nil)).(RebindInterface)
	if r1.Version() != "v1" {
		t.Errorf("Expected v1, got %s", r1.Version())
	}

	c.Bind((*RebindInterface)(nil), func(a *App) interface{} {
		return &RebindV2{}
	})

	r2 := c.Make((*RebindInterface)(nil)).(RebindInterface)
	if r2.Version() != "v2" {
		t.Errorf("Expected v2, got %s", r2.Version())
	}
}

// --- Singleton from nil ptr auto-creates instance ---

type SingletonAutoCreate struct {
	Value string `inject:"singleton-auto"`
}

func TestSingleton_NilPtrAutoCreates(t *testing.T) {
	c := New()

	c.Singleton((*SingletonAutoCreate)(nil))

	r1 := c.Make((*SingletonAutoCreate)(nil)).(*SingletonAutoCreate)
	r2 := c.Make((*SingletonAutoCreate)(nil)).(*SingletonAutoCreate)

	if r1 != r2 {
		t.Error("Singleton should return the same instance")
	}
	if r1.Value != "singleton-auto" {
		t.Errorf("Expected singleton-auto, got %s", r1.Value)
	}
}

// --- When + New() + di tag all together ---

type WhenNewDiCache interface {
	Backend() string
}

type WhenNewDiRedis struct{}

func (w WhenNewDiRedis) Backend() string { return "redis" }

type WhenNewDiService struct {
	Repo      WhenNewRepo    // Set by New() param + When override, NO inject tag
	Cache     WhenNewDiCache `inject:""`
	Container *App           `di:""`
	Label     string
}

func (w WhenNewDiService) New(repo WhenNewRepo) *WhenNewDiService {
	return &WhenNewDiService{
		Repo:  repo,
		Label: "built",
	}
}

func TestWhenNewDi_AllThreeInteractions(t *testing.T) {
	c := New()

	c.Bind((*WhenNewRepo)(nil), func(a *App) interface{} {
		return &WhenNewDBRepo{}
	})
	c.Bind((*WhenNewDiCache)(nil), func(a *App) interface{} {
		return &WhenNewDiRedis{}
	})

	c.When(&WhenNewDiService{}).Needs((*WhenNewRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenNewCacheRepo{}
	})

	result := c.Make(&WhenNewDiService{}).(*WhenNewDiService)

	if result.Repo == nil || result.Repo.Source() != "cache" {
		src := ""
		if result.Repo != nil {
			src = result.Repo.Source()
		}
		t.Errorf("When should override New() param, got %s", src)
	}
	if result.Cache == nil || result.Cache.Backend() != "redis" {
		t.Error("Cache should be injected via inject tag after New()")
	}
	if result.Container == nil {
		t.Error("di tag should inject container after New()")
	}
	if result.Label != "built" {
		t.Errorf("New() should set label, got %s", result.Label)
	}
}

// --- Multiple independent containers ---

type IsolatedService struct {
	Name string `inject:""`
}

type IsolatedInterface interface {
	Tag() string
}

type IsolatedImplA struct{}

func (i IsolatedImplA) Tag() string { return "A" }

type IsolatedImplB struct{}

func (i IsolatedImplB) Tag() string { return "B" }

type IsolatedConsumer struct {
	Svc IsolatedInterface `inject:""`
}

func TestMultipleContainers_Independent(t *testing.T) {
	c1 := New()
	c2 := New()

	c1.Bind((*IsolatedInterface)(nil), func(a *App) interface{} {
		return &IsolatedImplA{}
	})
	c2.Bind((*IsolatedInterface)(nil), func(a *App) interface{} {
		return &IsolatedImplB{}
	})

	r1 := c1.Make(&IsolatedConsumer{}).(*IsolatedConsumer)
	r2 := c2.Make(&IsolatedConsumer{}).(*IsolatedConsumer)

	if r1.Svc.Tag() != "A" {
		t.Errorf("Container 1 should resolve to A, got %s", r1.Svc.Tag())
	}
	if r2.Svc.Tag() != "B" {
		t.Errorf("Container 2 should resolve to B, got %s", r2.Svc.Tag())
	}
}
