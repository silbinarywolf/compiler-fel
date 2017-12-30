package parser

import (
	"fmt"

	"github.com/silbinarywolf/compiler-fel/ast"
	"github.com/silbinarywolf/compiler-fel/data"
	"github.com/silbinarywolf/compiler-fel/types"
)

type TypeInfoManager struct {
	registeredTypes map[string]TypeInfo

	// built-in
	intInfo    TypeInfo_Int
	floatInfo  TypeInfo_Float
	stringInfo TypeInfo_String
}

func (manager *TypeInfoManager) Init() {
	if manager.registeredTypes != nil {
		panic("Cannot initialize TypeInfoManager twice.")
	}
	manager.registeredTypes = make(map[string]TypeInfo)
	manager.register("int", manager.NewTypeInfoInt())
	manager.register("string", manager.NewTypeInfoString())
	manager.register("float", manager.NewTypeInfoFloat())
	manager.register("bool", types.Bool())
}

func (manager *TypeInfoManager) Clear() {
	if manager.registeredTypes == nil {
		panic("Cannot clear TypeInfoManager if it's already cleared..")
	}
	manager.registeredTypes = nil
}

func (manager *TypeInfoManager) register(name string, typeInfo TypeInfo) {
	_, ok := manager.registeredTypes[name]
	if ok {
		panic(fmt.Sprintf("Already registered \"%s\" type.", name))
	}
	manager.registeredTypes[name] = typeInfo
}

func (manager *TypeInfoManager) get(name string) TypeInfo {
	return manager.registeredTypes[name]
}

type TypeInfo interface {
	String() string
	Create() data.Type
}

// Int
type TypeInfo_Int struct{}

func (info *TypeInfo_Int) String() string    { return "int" }
func (info *TypeInfo_Int) Create() data.Type { return new(data.Integer64) }

func (manager *TypeInfoManager) NewTypeInfoInt() *TypeInfo_Int {
	return &manager.intInfo
}

// Float
type TypeInfo_Float struct{}

func (info *TypeInfo_Float) String() string    { return "float" }
func (info *TypeInfo_Float) Create() data.Type { return new(data.Float64) }

func (manager *TypeInfoManager) NewTypeInfoFloat() *TypeInfo_Float {
	return &manager.floatInfo
}

// String
type TypeInfo_String struct{}

func (info *TypeInfo_String) String() string    { return "string" }
func (info *TypeInfo_String) Create() data.Type { return new(data.String) }

func (manager *TypeInfoManager) NewTypeInfoString() *TypeInfo_String {
	return &manager.stringInfo
}

// Array
type TypeInfo_Array struct {
	underlying TypeInfo
}

func (info *TypeInfo_Array) String() string       { return "[]" + info.underlying.String() }
func (info *TypeInfo_Array) Underlying() TypeInfo { return info.underlying }
func (info *TypeInfo_Array) Create() data.Type    { return data.NewArray(info.underlying.Create()) }

func (manager *TypeInfoManager) NewTypeInfoArray(underlying TypeInfo) TypeInfo {
	result := new(TypeInfo_Array)
	result.underlying = underlying
	return result
}

// Struct
type TypeInfo_Struct struct {
	name       string
	definition *ast.StructDefinition
}

func (info *TypeInfo_Struct) String() string { return "struct " + info.name }
func (info *TypeInfo_Struct) Create() data.Type {
	panic("Handled by `evaluator`")
}
func (info *TypeInfo_Struct) Definition() *ast.StructDefinition { return info.definition }

func (manager *TypeInfoManager) NewStructInfo(definiton *ast.StructDefinition) TypeInfo {
	result := new(TypeInfo_Struct)
	result.name = definiton.Name.String()
	result.definition = definiton
	return result
}

// Functions
func (p *Parser) DetermineType(node *ast.Type) types.TypeInfo {
	var resultType types.TypeInfo

	str := node.Name.String()
	resultType = p.typeinfo.get(str)
	if resultType == nil {
		p.addErrorToken(fmt.Errorf("Undeclared type \"%s\".", str), node.Name)
	}
	if node.ArrayDepth > 0 {
		for i := 0; i < node.ArrayDepth; i++ {
			underlyingType := resultType
			resultType = p.typeinfo.NewTypeInfoArray(underlyingType)
		}
	}
	return resultType
}

func TypeEquals(a TypeInfo, b TypeInfo) bool {
	aAsArray, aOk := a.(*TypeInfo_Array)
	bAsArray, bOk := b.(*TypeInfo_Array)
	if aOk && bOk {
		return TypeEquals(aAsArray.underlying, bAsArray.underlying)
	}
	if a == b {
		return true
	}
	return false
}
