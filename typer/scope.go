package typer

import (
	"github.com/silbinarywolf/compiler-fel/ast"
	"github.com/silbinarywolf/compiler-fel/types"
)

type Scope struct {
	identifiers map[string]types.TypeInfo

	cssDefinitions       map[string]*ast.CSSDefinition
	cssConfigDefinitions map[string]*ast.CSSConfigDefinition
	htmlDefinitions      map[string]*ast.HTMLComponentDefinition
	structDefinitions    map[string]*ast.StructDefinition

	parent *Scope
}

func NewScope(parent *Scope) *Scope {
	result := new(Scope)

	result.identifiers = make(map[string]types.TypeInfo)

	// todo(jake): Refactor this so they all just go into "result.identifiers".
	//			   Will need some kind of system so that " :: html", " :: css" etc
	//			   blocks with the same name be stored in the same key.
	result.cssDefinitions = make(map[string]*ast.CSSDefinition)
	result.cssConfigDefinitions = make(map[string]*ast.CSSConfigDefinition)
	result.htmlDefinitions = make(map[string]*ast.HTMLComponentDefinition)
	result.structDefinitions = make(map[string]*ast.StructDefinition)

	result.parent = parent
	return result
}

func (scope *Scope) Set(name string, info types.TypeInfo) {
	scope.identifiers[name] = info
}

func (scope *Scope) Get(name string) (types.TypeInfo, bool) {
	info, ok := scope.identifiers[name]
	if !ok && scope.parent != nil {
		info, ok = scope.parent.Get(name)
	}
	return info, ok
}

func (scope *Scope) GetHTMLDefinition(name string) (*ast.HTMLComponentDefinition, bool) {
	value, ok := scope.htmlDefinitions[name]
	if !ok && scope.parent != nil {
		value, ok = scope.parent.GetHTMLDefinition(name)
	}
	return value, ok
}

func (scope *Scope) GetStructDefinition(name string) (*ast.StructDefinition, bool) {
	value, ok := scope.structDefinitions[name]
	if !ok && scope.parent != nil {
		value, ok = scope.parent.GetStructDefinition(name)
	}
	return value, ok
}

func (scope *Scope) GetCSSDefinition(name string) (*ast.CSSDefinition, bool) {
	value, ok := scope.cssDefinitions[name]
	if !ok && scope.parent != nil {
		value, ok = scope.parent.GetCSSDefinition(name)
	}
	return value, ok
}

func (scope *Scope) GetCSSConfigDefinition(name string) (*ast.CSSConfigDefinition, bool) {
	value, ok := scope.cssConfigDefinitions[name]
	if !ok && scope.parent != nil {
		value, ok = scope.parent.GetCSSConfigDefinition(name)
	}
	return value, ok
}

func (scope *Scope) GetFromThisScope(name string) (types.TypeInfo, bool) {
	info, ok := scope.identifiers[name]
	return info, ok
}