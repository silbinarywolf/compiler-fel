package ast

import (
	"github.com/silbinarywolf/compiler-fel/data"
	"github.com/silbinarywolf/compiler-fel/token"
)

type TypeKind int

const (
	TypeUnknown TypeKind = 0 + iota
	TypeString
	TypeInteger64
	TypeFloat64
	TypeHTMLDefinitionNode
)

type CSSRuleKind int

const (
	CSSKindUnknown CSSRuleKind = 0 + iota
	CSSKindRule
	CSSKindAtKeyword
)

type Node interface {
	Nodes() []Node
}

type Base struct {
	ChildNodes []Node
}

func (node *Base) Nodes() []Node {
	return node.ChildNodes
}

type File struct {
	Filepath string
	Base
}

type Block struct {
	Base
}

type Parameter struct {
	Name token.Token
	Expression
}

type Expression struct {
	TypeToken token.Token
	Type      data.Kind
	Base
}

type HTMLBlock struct {
	Base
}

type HTMLProperties struct {
	Statements []*DeclareStatement
}

func (node *HTMLProperties) Nodes() []Node {
	return nil
}

type HTMLComponentDefinition struct {
	Name          token.Token
	Dependencies  map[string]*HTMLNode
	Properties    *HTMLProperties
	CSSDefinition *CSSDefinition // optional
	Base
}

type HTMLNode struct {
	Name           token.Token
	Parameters     []Parameter
	HTMLDefinition *HTMLComponentDefinition // optional
	Base
}

type DeclareStatement struct {
	Name token.Token
	Expression
}

type Token struct {
	token.Token
}

func (node *Token) Nodes() []Node {
	return nil
}

type CSSDefinition struct {
	Name token.Token
	Base
}

type CSSRule struct {
	Kind      CSSRuleKind
	Selectors []CSSSelector
	Base
}

type CSSSelector struct {
	Base
}

type CSSAttributeSelector struct {
	Name     token.Token
	Operator token.Token
	Value    token.Token
}

func (node *CSSAttributeSelector) Nodes() []Node {
	return nil
}

type CSSProperty struct {
	Name token.Token
	Base
}
