package parser

import (
	"fmt"

	"github.com/silbinarywolf/compiler-fel/ast"
	"github.com/silbinarywolf/compiler-fel/token"
)

func (p *Parser) parseExpression() []ast.Node {
	parenOpenCount := 0
	parenCloseCount := 0

	//childNodes := make([]ast.Node, 0, 10)
	infixNodes := make([]ast.Node, 0, 10)
	operatorNodes := make([]*ast.Token, 0, 10)

Loop:
	for {
		t := p.PeekNextToken()
		switch t.Kind {
		case token.Identifier:
			p.GetNextToken()
			if p.PeekNextToken().Kind == token.ParenOpen {
				panic("parseExpression(): todo: Handle component/function in expression")
			}
			infixNodes = append(infixNodes, &ast.Token{Token: t})
		case token.String:
			p.GetNextToken()
			infixNodes = append(infixNodes, &ast.Token{Token: t})
		case token.Semicolon, token.Newline:
			p.GetNextToken()
			break Loop
		case token.BraceOpen, token.BraceClose, token.Comma:
			break Loop
		case token.ParenOpen:
			parenOpenCount++
		case token.ParenClose:
			// If hit end of parameter list
			if parenCloseCount == 0 && parenOpenCount == 0 {
				break Loop
			}

			parenCloseCount++

			topOperatorNode := operatorNodes[len(operatorNodes)-1]
			if topOperatorNode.Kind == token.ParenOpen {
				infixNodes = append(infixNodes, topOperatorNode)
				operatorNodes = operatorNodes[:len(operatorNodes)-1]
			}
		default:
			if t.IsOperator() {
				p.GetNextToken()

				// https://github.com/SilbinaryWolf/fel/blob/master/c_compiler/parser.h
				for len(operatorNodes) > 0 {
					topOperatorNode := operatorNodes[len(operatorNodes)-1]
					if topOperatorNode.Precedence() < t.Precedence() {
						break
					}
					operatorNodes = operatorNodes[:len(operatorNodes)-1]
					infixNodes = append(infixNodes, topOperatorNode)
				}
				operatorNodes = append(operatorNodes, &ast.Token{Token: t})
				continue
			}
			panic(fmt.Sprintf("parseExpression(): Unhandled token type: \"%s\" (value: %s) on Line %d", t.Kind.String(), t.String(), t.Line))
		}
	}

	for len(operatorNodes) > 0 {
		topOperatorNode := operatorNodes[len(operatorNodes)-1]
		operatorNodes = operatorNodes[:len(operatorNodes)-1]
		infixNodes = append(infixNodes, topOperatorNode)
	}

	if parenOpenCount != parenCloseCount {
		// todo(Jake): better error message
		panic("Mismatching paren open and close count")
	}

	// DEBUG
	//json, _ := json.MarshalIndent(infixNodes, "", "   ")
	//fmt.Printf("%s", string(json))
	//panic("todo: Finish parseExpression() func")

	return infixNodes
}
