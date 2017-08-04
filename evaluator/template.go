package evaluator

import (
	"github.com/silbinarywolf/compiler-fel/ast"
	"github.com/silbinarywolf/compiler-fel/data"
)

func (program *Program) evaluateHTMLNode(node *ast.HTMLNode, parentScope *Scope) *data.HTMLNode {
	resultDataNode := new(data.HTMLNode)
	resultDataNode.Name = node.Name.String()

	// Evaluate parameters
	if parameterSet := node.Parameters; parameterSet != nil {
		for _, parameter := range parameterSet {
			value := program.evaluateExpression(parameter.Nodes(), parentScope)
			attributeNode := data.HTMLAttribute{
				Name:  parameter.Name.String(),
				Value: value.String(),
			}
			resultDataNode.Attributes = append(resultDataNode.Attributes, attributeNode)
		}
	}

	scope := NewScope(parentScope)

	for _, itNode := range node.Nodes() {
		switch node := itNode.(type) {
		case *ast.HTMLNode:
			subResultDataNode := program.evaluateHTMLNode(node, scope)
			resultDataNode.ChildNodes = append(resultDataNode.ChildNodes, subResultDataNode)
		default:
			program.evaluateStatement(itNode, scope)
		}
	}
	return resultDataNode
}

func (program *Program) evaluateTemplate(nodes []ast.Node, scope *Scope) []*data.HTMLNode {
	var resultDataNodeSet []*data.HTMLNode

	for _, itNode := range nodes {
		switch node := itNode.(type) {
		case *ast.HTMLNode:
			subResultDataNode := program.evaluateHTMLNode(node, scope)
			resultDataNodeSet = append(resultDataNodeSet, subResultDataNode)
		default:
			program.evaluateStatement(itNode, scope)
		}
	}

	return resultDataNodeSet
}
