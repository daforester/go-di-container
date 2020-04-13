package di

import (
	"reflect"
	"testing"
)

var app *App

type AppTestInterface interface {
	IsAppTestInterface() bool
}

type AppTestStruct struct {
	A int
	B string
}

func (a AppTestStruct) IsAppTestInterface() bool {
	return true
}

type AppTestInjectStruct struct {
	TestStruct  AppTestInterface `inject:""`
	TestStructS AppTestStruct    `inject:""`
	TestStructP *AppTestStruct   `inject:""`
	A           int              `inject:"1"`
	B           string           `inject:"string"`
}

type AppTestNewStruct struct {
	TestStruct AppTestInterface
	A          int
	B          string
}

func (a AppTestNewStruct) New(i AppTestInterface) *AppTestNewStruct {
	return &AppTestNewStruct{
		TestStruct: i,
		A:          5,
		B:          "string",
	}
}

type AppTestNewStructS struct {
	TestStruct AppTestInterface
	A          int
	B          string
}

func (a AppTestNewStructS) New(i AppTestInterface) AppTestNewStructS {
	return AppTestNewStructS{
		TestStruct: i,
		A:          55,
		B:          "strings",
	}
}

type AppTestTypeInjects struct {
	B    bool    `inject:"true"`
	F32  float32 `inject:"3.1432"`
	F64  float64 `inject:"3.14164"`
	I    int     `inject:"-3"`
	I8   int8    `inject:"-8"`
	I16  int16   `inject:"-16"`
	I32  int32   `inject:"-32"`
	I64  int64   `inject:"-64"`
	Ui   uint    `inject:"3"`
	Ui8  uint8   `inject:"8"`
	Ui16 uint16  `inject:"16"`
	Ui32 uint32  `inject:"32"`
	Ui64 uint64  `inject:"64"`
}

type AppTestBadBoolInject struct {
	B bool `inject:"yes"`
}

type AppTestBadFloat32Inject struct {
	F32 float32 `inject:"yes"`
}

type AppTestBadFloat64Inject struct {
	F32 float64 `inject:"yes"`
}

type AppTestBadIntInject struct {
	I int `inject:"yes"`
}

type AppTestBadInt8Inject struct {
	I int8 `inject:"yes"`
}

type AppTestBadInt16Inject struct {
	I int16 `inject:"yes"`
}

type AppTestBadInt32Inject struct {
	I int32 `inject:"yes"`
}

type AppTestBadInt64Inject struct {
	I int64 `inject:"yes"`
}

type AppTestBadUIntInject struct {
	I uint `inject:"yes"`
}

type AppTestBadUInt8Inject struct {
	I uint8 `inject:"yes"`
}

type AppTestBadUInt16Inject struct {
	I uint16 `inject:"yes"`
}

type AppTestBadUInt32Inject struct {
	I uint32 `inject:"yes"`
}

type AppTestBadUInt64Inject struct {
	I uint64 `inject:"yes"`
}

type AppTestBadInjectTypeValue struct {
	M map[string]string `inject:"yes"`
}

type AppTestBadInjectInt struct {
	I int `inject:""`
}

type AppTestBadInjectString struct {
	S string `inject:""`
}

type AppTestBadInjectType struct {
	M map[string]string `inject:""`
}

type AppTestNewBadInjectType struct {
	x map[string]string
}

func (a AppTestNewBadInjectType) New(x map[string]string) *AppTestNewBadInjectType {
	n := new(AppTestNewBadInjectType)
	n.x = x
	return n
}

type AppTestNewBadNilReturn struct{}

func (a AppTestNewBadNilReturn) New() {}

type AppTestNewBadIncompatibleReturn struct{}

func (a AppTestNewBadIncompatibleReturn) New() *AppTestStruct {
	x := new(AppTestStruct)
	return x
}

type AppTestNewUnexpectedReturn struct{}

func (a AppTestNewUnexpectedReturn) New() int {
	return 1
}

func TestNewApp(t *testing.T) {
	a := Default()
	if a == nil {
		t.Error("Default app not created")
	}

	a = Default("foobar")
	if a == nil {
		t.Error("Foobar app not created")
	}

	config := AppConfig{
		Default: true,
		Name: "default",
		ObjectBuilder: new(MockObject),
		TypeChecker: new(MockTypeChecker),
	}
	app = New(config)

	if app == nil {
		t.Error("App not created")
	}

	if reflect.TypeOf(app) != reflect.TypeOf(&App{}) {
		t.Error("NewApp should return pointer to app instance")
	}
}

func TestDefault(t *testing.T) {
	a := Default("default").(*App)
	if a.objectBuilder == nil {
		t.Error("Default app we setup was not returned, but a new one instead")
	}

	a = Default().(*App)
	if a.objectBuilder == nil {
		t.Error("Default app we setup was not returned, but a new one instead")
	}
}

func TestApp_Bind(t *testing.T) {
	app.Bind("teststruct", struct {
		A int
		B string
	}{
		A: 1,
		B: "string",
	})

	app.Bind("teststructptr", &struct {
		A int
		B string
	}{
		A: 1,
		B: "string",
	})

	app.Bind("testfunc", func(a *App) interface{} {
		return struct {
			A int
			B string
		}{
			A: 1,
			B: "string",
		}
	})

	app.Bind("testalias", "testfunc")

	app.Bind(&AppTestStruct{}, func(a *App) interface{} {
		return &AppTestStruct{
			A: 1,
			B: "B",
		}
	})

	app.Bind((*AppTestInterface)(nil), AppTestStruct{})

	app.Bind((*AppTestInterface)(nil), &AppTestStruct{})

	app.Bind("testalias", (*AppTestInterface)(nil))

	app.Bind((*AppTestInterface)(nil), "testalias")
	app.Bind(new(AppTestInterface), "testalias")

	app.Bind((*AppTestInterface)(nil), func(a *App) interface{} {
		return &AppTestStruct{
			A: 2,
			B: "pointer",
		}
	})

	app.Bind("deletebindtest", &AppTestStruct{})
}

func TestApp_Bind_NillInputFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Bind with nil input")
		}
	}()

	app.Bind(nil, &AppTestStruct{})
}

func TestApp_Bind_InterfaceToIncompatibleFuncFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Bind with Incompatible Func to Interface")
		}
	}()

	app.Bind((*AppTestInterface)(nil), func() int {
		return 1
	})
}

func TestApp_Bind_InterfaceToFuncWithIncompatibleReturnFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Bind with BindFunc to incompatible type for interface")
		}

	}()

	var bf BindFunc = func(i *App) interface{} {
		return 1
	}

	app.Bind((*AppTestInterface)(nil), bf)
}

func TestApp_Bind_InterfaceToIncompatiblePointerFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Bind with Incompatible Ptr to Interface")
		}
	}()

	app.Bind((*AppTestInterface)(nil), &struct{}{})
}

func TestApp_Bind_StructToFuncWithIncompatibleReturnFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Bind with BindFunc to incompatible type for struct")
		}

	}()

	var bf BindFunc = func(i *App) interface{} {
		return 1
	}

	app.Bind(&AppTestStruct{}, bf)
}

func TestApp_Bind_IncompatibleInputFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Bind with Incompatible bind type")
		}
	}()

	app.Bind(1, &struct{}{})
}

func TestApp_Bind_NullObjectFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Bind not creating object")
		}
		app.objectBuilder.(*MockObject).DisableOverride()
	}()

	app.objectBuilder.(*MockObject).OverrideNew(nil)
	app.Bind((*AppTestInterface)(nil), &AppTestStruct{})
}

func TestApp_Bind_RemoveBinding(t *testing.T) {
	app.Bind(&AppTestNewStructS{}, nil)
	app.Bind("deletebindtest", nil)
}

func TestApp_Singleton(t *testing.T) {
	app.Singleton(&AppTestStruct{})

	var bf BindFunc = func(a *App) interface{} {
		return &AppTestStruct{}
	}
	app.Singleton(&AppTestStruct{}, bf)
	app.Singleton((*AppTestInterface)(nil), bf)
	app.Singleton((*AppTestInterface)(nil), &AppTestStruct{})
}

func TestApp_Singleton_InvalidInput1P_Fail1(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to invalid input to Singleton")
		}
	}()

	app.Singleton(func() int {
		return 1
	})
}

func TestApp_Singleton_InvalidInput1P_Fail2(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to invalid input to Singleton")
		}
	}()

	app.Singleton(1)
}

func TestApp_Singleton_InvalidInput1P_Fail3(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to invalid input to Singleton")
		}
	}()

	app.Singleton("foobar")
}

func TestApp_Singleton_InvalidInput3P_Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to too many inputs to Singleton")
		}
	}()

	app.Singleton((*AppTestInterface)(nil), &AppTestStruct{}, &AppTestStruct{})
}

func TestApp_Singleton_NillInputFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton with nil input")
		}
	}()

	app.Singleton(nil, &AppTestStruct{})
}

func TestApp_Singleton_InterfaceToIncompatibleFuncFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton with Incompatible Func to Interface")
		}
	}()

	app.Singleton((*AppTestInterface)(nil), func() int {
		return 1
	})
}

func TestApp_Singleton_InterfaceToFuncWithIncompatibleReturnFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton with BindFunc to incompatible type for interface")
		}

	}()

	var bf BindFunc = func(i *App) interface{} {
		return 1
	}

	app.Singleton((*AppTestInterface)(nil), bf)
}

func TestApp_Singleton_InterfaceToStructFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton with Struct")
		}
	}()

	app.Singleton(struct{}{})
}

func TestApp_Singleton_InterfaceToIncompatiblePointerFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton with Incompatible Ptr to Interface")
		}
	}()

	app.Singleton((*AppTestInterface)(nil), &struct{}{})
}

func TestApp_Singleton_StructToFuncWithIncompatibleReturnFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton with BindFunc to incompatible type for struct")
		}

	}()

	var bf BindFunc = func(i *App) interface{} {
		return 1
	}

	app.Singleton(&AppTestStruct{}, bf)
}

func TestApp_Singleton_IncompatibleInputFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton with Incompatible bind type")
		}
	}()

	app.Singleton(1, &struct{}{})
}

func TestApp_Singleton_IncompatiblePointerFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton with Incompatible bind type")
		}
	}()

	i := 1
	app.Singleton(&i)
}

func TestApp_Singleton_NullObjectFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to Singleton not creating object")
		}
		app.objectBuilder.(*MockObject).DisableOverride()
	}()

	app.objectBuilder.(*MockObject).OverrideNew(nil)
	app.Singleton((*AppTestInterface)(nil), &AppTestStruct{})
}

func TestApp_Singleton_RemoveBinding(t *testing.T) {
	app.Singleton(AppTestNewStructS{}, nil)
}

func TestApp_Make(t *testing.T) {
	o := app.Make((*AppTestInterface)(nil))
	if reflect.TypeOf(o) != reflect.TypeOf(&AppTestStruct{}) {
		t.Error("App did not make an AppTestStruct")
	}
	o = app.Make(&AppTestStruct{})
	if reflect.TypeOf(o) != reflect.TypeOf(&AppTestStruct{}) {
		t.Error("App did not make an AppTestStruct")
	}
	o = app.Make("testalias")
	if reflect.TypeOf(o) != reflect.TypeOf(&AppTestStruct{}) {
		t.Error("App did not make an AppTestStruct")
	}
	o = app.Make("testfunc")
	if reflect.TypeOf(o) != reflect.TypeOf(struct {
		A int
		B string
	}{}) {
		t.Error("App did not make an AppTestStruct")
	}
	o = app.Make("teststruct")
	if reflect.TypeOf(o) != reflect.TypeOf(struct {
		A int
		B string
	}{}) {
		t.Error("App did not make a Struct")
	}
	o = app.Make("teststructptr")
	if reflect.TypeOf(o) != reflect.TypeOf(&struct {
		A int
		B string
	}{}) {
		t.Error("App did not make a Struct Ptr")
	}
	o = app.Make(AppTestInjectStruct{})
	if reflect.TypeOf(o) != reflect.TypeOf(AppTestInjectStruct{}) {
		t.Error("App did not make an AppTestInjectStruct")
	}
	o = app.Make(&AppTestInjectStruct{})
	if reflect.TypeOf(o) != reflect.TypeOf(&AppTestInjectStruct{}) {
		t.Error("App did not make an *AppTestInjectStruct")
	}
	o = app.Make(AppTestNewStruct{})
	if reflect.TypeOf(o) != reflect.TypeOf(AppTestNewStruct{}) {
		t.Error("App did not make an AppTestNewStruct")
	}
	if o.(AppTestNewStruct).A != 5 || o.(AppTestNewStruct).B != "string" {
		t.Error("App returned correct type but not created by New")
	}
	o = app.Make(&AppTestNewStruct{})
	if reflect.TypeOf(o) != reflect.TypeOf(&AppTestNewStruct{}) {
		t.Error("App did not make an *AppTestNewStruct")
	}
	if o.(*AppTestNewStruct).A != 5 || o.(*AppTestNewStruct).B != "string" {
		t.Error("App returned correct type but not created by New")
	}
	o = app.Make(AppTestNewStructS{})
	if reflect.TypeOf(o) != reflect.TypeOf(AppTestNewStructS{}) {
		t.Error("App did not make an AppTestNewStructS")
	}
	if o.(AppTestNewStructS).A != 55 || o.(AppTestNewStructS).B != "strings" {
		t.Error("App returned correct type but not created by New")
	}
	o = app.Make(&AppTestNewStructS{})
	if reflect.TypeOf(o) != reflect.TypeOf(&AppTestNewStructS{}) {
		t.Error("App did not make an *AppTestNewStructS")
	}
	if o.(*AppTestNewStructS).A != 55 || o.(*AppTestNewStructS).B != "strings" {
		t.Error("App returned correct type but not created by New")
	}

	o = app.Make(AppTestTypeInjects{})
	if reflect.TypeOf(o) != reflect.TypeOf(AppTestTypeInjects{}) {
		t.Error("App did not make an *AppTestTypeInjects")
	}
	ov := o.(AppTestTypeInjects)
	if ov.B != true {
		t.Error("Bool should be set to true")
	}
	if ov.F32 != 3.1432 {
		t.Error("Float32 was not set to 3.1432")
	}
	if ov.F64 != 3.14164 {
		t.Error("Float32 was not set to 3.14164")
	}
	if ov.I != -3 {
		t.Error("Int was not set to -3")
	}
	if ov.I8 != -8 {
		t.Error("Int8 was not set to -8")
	}
	if ov.I16 != -16 {
		t.Error("Int16 was not set to -16")
	}
	if ov.I32 != -32 {
		t.Error("Int32 was not set to -32")
	}
	if ov.I64 != -64 {
		t.Error("Int64 was not set to -64")
	}
	if ov.Ui != 3 {
		t.Error("Uint was not set to 3")
	}
	if ov.Ui8 != 8 {
		t.Error("Uint8 was not set to 8")
	}
	if ov.Ui16 != 16 {
		t.Error("Uint16 was not set to 16")
	}
	if ov.Ui32 != 32 {
		t.Error("Uint32 was not set to 32")
	}
	if ov.Ui64 != 64 {
		t.Error("Uint64 was not set to 64")
	}
}

func TestApp_MakeWith(t *testing.T) {
	o := app.MakeWith(&AppTestInjectStruct{}, map[string]interface{}{
		"A": 8,
		"B": "Foobar",
	})
	if reflect.TypeOf(o) != reflect.TypeOf(&AppTestInjectStruct{}) {
		t.Error("App did not make an *AppTestNewStruct")
	}
	if o.(*AppTestInjectStruct).A != 8 {
		t.Error("App did not inject 5 into integer")
	}
	if o.(*AppTestInjectStruct).B != "Foobar" {
		t.Error("App did not inject \"Foobar\" into string")
	}
}

func TestApp_Make_SingletonNonPtrObjectFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to invalid Singleton Object")
		}
	}()

	app.registry["BadSingletonNonPtrObject"] = &Object{
		Value:        nil,
		Name:         "BadSingletonNonPtrObject",
		Kind:         0,
		singleton:    true,
		reflectType:  nil,
		reflectValue: reflect.Value{},
	}
	app.Make("BadSingletonNonPtrObject")
}

func TestApp_Make_BadFuncObjectFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to invalid Object Func with no return")
		}
	}()

	app.registry["BadFuncObject"] = &Object{
		Value:        func(a *App) {},
		Name:         "BadFuncObject",
		Kind:         Func,
		singleton:    false,
		reflectType:  nil,
		reflectValue: reflect.Value{},
	}
	app.Make("BadFuncObject")
}

func TestApp_Make_BadObjectTypeFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to invalid Object Type")
		}
	}()

	app.registry["BadObjectType"] = &Object{
		Value:        0,
		Name:         "BadObjectType",
		Kind:         Unknown,
		singleton:    false,
		reflectType:  nil,
		reflectValue: reflect.Value{},
	}
	app.Make("BadObjectType")
}

func TestApp_Make_NoBindingFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to non-existent Binding")
		}
	}()

	app.Make("BadBinding")
}

func TestApp_Make_InjectIntFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable int")
		}
	}()

	app.Make(&AppTestBadInjectInt{})
}

func TestApp_Make_InjectStringFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable string")
		}
	}()

	app.Make(&AppTestBadInjectString{})
}

func TestApp_Make_InjectTypeFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable type")
		}
	}()

	app.Make(&AppTestBadInjectType{})
}

func TestApp_Make_NewTypeFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable type")
		}
	}()

	app.Make(&AppTestNewBadInjectType{})
}

func TestApp_Make_NewNilFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to nil return in new")
		}
	}()

	app.Make(&AppTestNewBadNilReturn{})
}

func TestApp_Make_NewIncompatibleFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to incompatible type return in new")
		}
	}()

	app.Make(&AppTestNewBadIncompatibleReturn{})
}

func TestApp_Make_NewUnexpectedFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to unexpected compatibility success")
		}
		app.typeChecker.(*MockTypeChecker).DisableOverride()
	}()

	app.typeChecker.(*MockTypeChecker).Override(true)
	app.Make(&AppTestNewUnexpectedReturn{})
}

func TestApp_Make_InjectValueTypeFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable type value")
		}
	}()

	app.Make(&AppTestBadInjectTypeValue{})
}

func TestApp_Make_InjectValueBoolFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable bool value")
		}
	}()

	app.Make(&AppTestBadBoolInject{})
}

func TestApp_Make_InjectValueFloat32Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable float32 value")
		}
	}()

	app.Make(&AppTestBadFloat32Inject{})
}

func TestApp_Make_InjectValueFloat64Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable float64 value")
		}
	}()

	app.Make(&AppTestBadFloat64Inject{})
}

func TestApp_Make_InjectValueIntFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable int value")
		}
	}()

	app.Make(&AppTestBadIntInject{})
}

func TestApp_Make_InjectValueInt8Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable int8 value")
		}
	}()

	app.Make(&AppTestBadInt8Inject{})
}

func TestApp_Make_InjectValueInt16Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable int16 value")
		}
	}()

	app.Make(&AppTestBadInt16Inject{})
}

func TestApp_Make_InjectValueInt32Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable int32 value")
		}
	}()

	app.Make(&AppTestBadInt32Inject{})
}

func TestApp_Make_InjectValueInt64Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable int64 value")
		}
	}()

	app.Make(&AppTestBadInt64Inject{})
}

func TestApp_Make_InjectValueUIntFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable uint value")
		}
	}()

	app.Make(&AppTestBadUIntInject{})
}

func TestApp_Make_InjectValueUInt8Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable uint8 value")
		}
	}()

	app.Make(&AppTestBadUInt8Inject{})
}

func TestApp_Make_InjectValueUInt16Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable uint16 value")
		}
	}()

	app.Make(&AppTestBadUInt16Inject{})
}

func TestApp_Make_InjectValueUInt32Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable uint32 value")
		}
	}()

	app.Make(&AppTestBadUInt32Inject{})
}

func TestApp_Make_InjectValueUInt64Fail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad injectable uint64 value")
		}
	}()

	app.Make(&AppTestBadUInt64Inject{})
}

func TestApp_When(t *testing.T) {
	app.When(&AppTestNewStruct{}).Needs((*AppTestInterface)(nil)).Give(func(a *App) interface{} {
		return &AppTestStruct{
			1001,
			"stringBBB",
		}
	})

	o := app.Make(&AppTestNewStruct{})
	if o == nil {
		t.Error("Error creating object with given needs")
	}
	if o.(*AppTestNewStruct).TestStruct == nil {
		t.Error("Object field not setup")
	}
	if o.(*AppTestNewStruct).TestStruct.(*AppTestStruct).A != 1001 || o.(*AppTestNewStruct).TestStruct.(*AppTestStruct).B != "stringBBB" {
		t.Error("Object field values not set")
	}

	app.When(&AppTestInjectStruct{}).Needs((*AppTestInterface)(nil)).Give(func(a *App) interface{} {
		return &AppTestStruct{
			2002,
			"stringCCC",
		}
	})

	o = app.Make(&AppTestInjectStruct{})
	if o == nil {
		t.Error("Error creating object with given needs")
	}
	if o.(*AppTestInjectStruct).TestStruct == nil {
		t.Error("Object field not setup")
	}
	if o.(*AppTestInjectStruct).TestStruct.(*AppTestStruct).A != 2002 || o.(*AppTestInjectStruct).TestStruct.(*AppTestStruct).B != "stringCCC" {
		t.Error("Object field values not set")
	}

}

func TestApp_When_BindFail(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in response to bad binding for When, Need, Give")
		}
	}()

	app.When(&AppTestNewStruct{}).Needs((*AppTestInterface)(nil)).Give(func(a *App) interface{} {
		return &AppTestInjectStruct{
			AppTestStruct{},
			AppTestStruct{},
			&AppTestStruct{},
			1001,
			"stringBBB",
		}
	})
}

func TestApp_When_Delete(t *testing.T) {
	app.When(&AppTestNewStruct{}).Needs((*AppTestInterface)(nil)).Give(nil)
}
