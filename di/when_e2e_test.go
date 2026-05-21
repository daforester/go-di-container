package di

import (
	"testing"
)

type WhenTestRepo interface {
	Save() string
}

type WhenTestCache interface {
	Get() string
}

type WhenTestDBRepo struct {
	Name string
}

func (w WhenTestDBRepo) Save() string {
	return "db:" + w.Name
}

type WhenTestCacheRepo struct {
	Name string
}

func (w WhenTestCacheRepo) Save() string {
	return "cache:" + w.Name
}

type WhenTestMemoryCache struct{}

func (w WhenTestMemoryCache) Get() string {
	return "memory"
}

type WhenTestRedisCache struct{}

func (w WhenTestRedisCache) Get() string {
	return "redis"
}

type WhenTestHandler struct {
	Repo  WhenTestRepo  `inject:""`
	Cache WhenTestCache `inject:""`
}

type WhenTestHandlerPtr struct {
	Repo  WhenTestRepo  `inject:""`
	Cache WhenTestCache `inject:""`
}

type WhenTestHandlerWithNew struct {
	Repo  WhenTestRepo  `inject:""`
	Cache WhenTestCache `inject:""`
}

func (w WhenTestHandlerWithNew) New(repo WhenTestRepo) *WhenTestHandlerWithNew {
	return &WhenTestHandlerWithNew{
		Repo: repo,
	}
}

type WhenTestService struct{}

type WhenTestServicePtr struct{}

func TestWhen_WhenNeedsGive_WithStruct(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return WhenTestCacheRepo{Name: "override"}
	})

	handler := c.Make(&WhenTestHandler{}).(*WhenTestHandler)

	if handler.Repo.Save() != "cache:override" {
		t.Errorf("Expected cache:override, got %s", handler.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_WithPtr(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "override"}
	})

	handler := c.Make(&WhenTestHandler{}).(*WhenTestHandler)

	if handler.Repo.Save() != "cache:override" {
		t.Errorf("Expected cache:override, got %s", handler.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_WithFunc(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "func-override"}
	})

	handler := c.Make(&WhenTestHandler{}).(*WhenTestHandler)

	if handler.Repo.Save() != "cache:func-override" {
		t.Errorf("Expected cache:func-override, got %s", handler.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_WithAlias(t *testing.T) {
	c := New()

	c.Bind("defaultRepo", func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind("overrideRepo", func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "alias-override"}
	})

	c.When(&WhenTestHandler{}).Needs("defaultRepo").Give("overrideRepo")

	repo := c.Make("defaultRepo").(*WhenTestDBRepo)
	if repo.Name != "default" {
		t.Errorf("Named Make should still return default, got %s", repo.Name)
	}
}

func TestWhen_WhenNeedsGive_MultipleHandlers(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "handler"}
	})

	h1 := c.Make(&WhenTestHandler{}).(*WhenTestHandler)

	if h1.Repo.Save() != "cache:handler" {
		t.Errorf("Handler expected cache:handler, got %s", h1.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_StringNeedStructGive(t *testing.T) {
	c := New()

	c.Bind("primaryRepo", func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "primary"}
	})

	c.When(&WhenTestHandler{}).Needs("primaryRepo").Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "override"}
	})

	repo := c.Make("primaryRepo").(*WhenTestDBRepo)
	if repo.Name != "primary" {
		t.Errorf("Named Make should still return primary, got %s", repo.Name)
	}
}

func TestWhen_WhenNeedsGive_RemoveBinding(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "override"}
	})

	h1 := c.Make(&WhenTestHandler{}).(*WhenTestHandler)
	if h1.Repo.Save() != "cache:override" {
		t.Errorf("Expected cache:override, got %s", h1.Repo.Save())
	}

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(nil)

	h2 := c.Make(&WhenTestHandler{}).(*WhenTestHandler)
	if h2.Repo.Save() != "db:default" {
		t.Errorf("Expected db:default after removal, got %s", h2.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_IncompatibleInterfaceToStruct(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when giving struct that doesn't implement interface")
		}
	}()

	c := New()
	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(WhenTestMemoryCache{})
}

func TestWhen_WhenNeedsGive_IncompatibleInterfaceToPtr(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when giving ptr that doesn't implement interface")
		}
	}()

	c := New()
	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(&WhenTestMemoryCache{})
}

func TestWhen_WhenNeedsGive_IncompatibleInterfaceToFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when making with incompatible func return type")
		}
	}()

	c := New()
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})
	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.Make(&WhenTestHandler{})
}

func TestWhen_WhenNeedsGive_IncompatibleStructToFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when making with incompatible func return type")
		}
	}()

	c := New()
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})
	c.When(&WhenTestHandler{}).Needs(WhenTestDBRepo{}).Give(func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.Make(&WhenTestHandler{})
}

func TestWhen_WhenNeedsGive_IncompatiblePtrToFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when making with incompatible func return type")
		}
	}()

	c := New()
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})
	c.When(&WhenTestHandler{}).Needs(&WhenTestDBRepo{}).Give(func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.Make(&WhenTestHandler{})
}

func TestWhen_WhenNeedsGive_IncompatibleTypes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when giving incompatible types")
		}
	}()

	c := New()
	c.When(&WhenTestHandler{}).Needs(1).Give(2)
}

func TestWhen_WhenNeedsGive_StringNeedStringGive(t *testing.T) {
	c := New()

	c.Bind("primaryRepo", func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "primary"}
	})
	c.Bind("secondaryRepo", func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "secondary"}
	})

	c.When(&WhenTestHandler{}).Needs("primaryRepo").Give("secondaryRepo")

	repo := c.Make("primaryRepo").(*WhenTestDBRepo)
	if repo.Name != "primary" {
		t.Errorf("Named Make should still return primary, got %s", repo.Name)
	}
}

func TestWhen_WhenNeedsGive_InterfaceToIncompatibleFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when giving func with wrong signature")
		}
	}()

	c := New()
	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func() string {
		return "wrong"
	})
}

func TestWhen_WhenNeedsGive_InterfaceToFuncWithIncompatibleReturn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when giving func with incompatible return")
		}
	}()

	c := New()
	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return 42
	})

	c.Make(&WhenTestHandler{})
}

func TestWhen_WhenNeedsGive_InterfaceToCompatibleFunc(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "func-repo"}
	})

	handler := c.Make(&WhenTestHandler{}).(*WhenTestHandler)

	if handler.Repo.Save() != "cache:func-repo" {
		t.Errorf("Expected cache:func-repo, got %s", handler.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_WithNilRequestingType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when using nil as requesting type")
		}
	}()

	c := New()
	c.When(nil).Needs((*WhenTestRepo)(nil)).Give(&WhenTestDBRepo{})
}

func TestWhen_WhenNeedsGive_StringNeedPtrGive(t *testing.T) {
	c := New()

	c.Bind("primaryRepo", func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "primary"}
	})

	c.When(&WhenTestHandler{}).Needs("primaryRepo").Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "override"}
	})

	repo := c.Make("primaryRepo").(*WhenTestDBRepo)
	if repo.Name != "primary" {
		t.Errorf("Named Make should still return primary, got %s", repo.Name)
	}
}

func TestWhen_WhenNeedsGive_StringNeedFuncGive(t *testing.T) {
	c := New()

	c.Bind("primaryRepo", func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "primary"}
	})

	c.When(&WhenTestHandler{}).Needs("primaryRepo").Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "func-override"}
	})

	repo := c.Make("primaryRepo").(*WhenTestDBRepo)
	if repo.Name != "primary" {
		t.Errorf("Named Make should still return primary, got %s", repo.Name)
	}
}

func TestWhen_WhenNeedsGive_MultipleNeedsSameHandler(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default-repo"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "repo-override"}
	})
	c.When(&WhenTestHandler{}).Needs((*WhenTestCache)(nil)).Give(func(a *App) interface{} {
		return &WhenTestRedisCache{}
	})

	handler := c.Make(&WhenTestHandler{}).(*WhenTestHandler)

	if handler.Repo.Save() != "cache:repo-override" {
		t.Errorf("Expected cache:repo-override, got %s", handler.Repo.Save())
	}
	if handler.Cache.Get() != "redis" {
		t.Errorf("Expected redis, got %s", handler.Cache.Get())
	}
}

func TestWhen_WhenNeedsGive_DoesNotAffectOtherHandlers(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "override"}
	})

	h1 := c.Make(&WhenTestHandler{}).(*WhenTestHandler)
	h2 := c.Make(&WhenTestHandlerPtr{}).(*WhenTestHandlerPtr)

	if h1.Repo.Save() != "cache:override" {
		t.Errorf("Handler expected cache:override, got %s", h1.Repo.Save())
	}
	if h2.Repo.Save() != "db:default" {
		t.Errorf("HandlerPtr expected db:default, got %s", h2.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_WithInterfaceRequestingType(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When((*WhenTestHandler)(nil)).Needs((*WhenTestRepo)(nil)).Give(WhenTestCacheRepo{Name: "interface-override"})

	handler := c.Make(&WhenTestHandler{}).(*WhenTestHandler)

	if handler.Repo.Save() != "cache:interface-override" {
		t.Errorf("Expected cache:interface-override, got %s", handler.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_WithPtrRequestingType(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	c.When(&WhenTestHandler{}).Needs((*WhenTestRepo)(nil)).Give(WhenTestCacheRepo{Name: "ptr-override"})

	handler := c.Make(&WhenTestHandler{}).(*WhenTestHandler)

	if handler.Repo.Save() != "cache:ptr-override" {
		t.Errorf("Expected cache:ptr-override, got %s", handler.Repo.Save())
	}
}

func TestWhen_WhenNeedsGive_AppliesToInjectFieldsAfterNew(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	// When binding for the Cache field (inject tag, resolved via processStructTags after New())
	c.When(&WhenTestHandlerWithNew{}).Needs((*WhenTestCache)(nil)).Give(func(a *App) interface{} {
		return &WhenTestRedisCache{}
	})

	handler := c.Make(&WhenTestHandlerWithNew{}).(*WhenTestHandlerWithNew)

	// Repo comes from the New() constructor parameter (default binding)
	if handler.Repo.Save() != "db:default" {
		t.Errorf("Expected db:default for Repo (from New param), got %s", handler.Repo.Save())
	}

	// Cache should use the When override, not the default binding
	if handler.Cache.Get() != "redis" {
		t.Errorf("Expected redis for Cache (When override on inject field after New), got %s", handler.Cache.Get())
	}
}

func TestWhen_WhenNeedsGive_NewParamAndFieldBothOverridden(t *testing.T) {
	c := New()

	c.Bind((*WhenTestRepo)(nil), func(a *App) interface{} {
		return &WhenTestDBRepo{Name: "default"}
	})
	c.Bind((*WhenTestCache)(nil), func(a *App) interface{} {
		return &WhenTestMemoryCache{}
	})

	// Override both the New() parameter and the inject field via When
	c.When(&WhenTestHandlerWithNew{}).Needs((*WhenTestRepo)(nil)).Give(func(a *App) interface{} {
		return &WhenTestCacheRepo{Name: "when-repo"}
	})
	c.When(&WhenTestHandlerWithNew{}).Needs((*WhenTestCache)(nil)).Give(func(a *App) interface{} {
		return &WhenTestRedisCache{}
	})

	handler := c.Make(&WhenTestHandlerWithNew{}).(*WhenTestHandlerWithNew)

	if handler.Repo.Save() != "cache:when-repo" {
		t.Errorf("Expected cache:when-repo for Repo (When override on New param), got %s", handler.Repo.Save())
	}
	if handler.Cache.Get() != "redis" {
		t.Errorf("Expected redis for Cache (When override on inject field), got %s", handler.Cache.Get())
	}
}
