package ast

import "github.com/silbinarywolf/compiler-fel/token"

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
	Base
}

type Expression struct {
	Base
}

/*type NamedBlock struct {
	Name token.Token
	Block
}*/

type HTMLNode struct {
	Name       token.Token
	Parameters []Parameter
	Base
}

type DeclareStatement struct {
	Name token.Token
	Base
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
