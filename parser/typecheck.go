package parser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/silbinarywolf/compiler-fel/ast"
	"github.com/silbinarywolf/compiler-fel/token"
	"github.com/silbinarywolf/compiler-fel/types"
	"github.com/silbinarywolf/compiler-fel/util"
)

/*type Checker struct {
	*Parser
	scope *Scope
}

func (checker *Checker) PushScope() {
	scope := NewScope(checker.scope)
	checker.scope = scope
}

func (checker *Checker) PopScope() {
	parentScope := checker.scope.parent
	if parentScope == nil {
		panic("Cannot pop last scope item.")
	}
	checker.scope = parentScope
}*/

// func getDataTypeFromTokaen(t token.Token) data.Kind {
// 	switch t.Kind {
// 	case token.Identifier:
// 		typename := t.String()
// 		switch typename {
// 		case "string":
// 			return data.KindString
// 		case "int", "int64":
// 			return data.KindInteger64
// 		case "float", "float64":
// 			return data.KindFloat64
// 		case "html_node":
// 			return data.KindHTMLNode
// 		default:
// 			panic(fmt.Sprintf("Unknown type name: %s", typename))
// 		}
// 	default:
// 		panic(fmt.Sprintf("Cannot use token kind %s in type declaration", t.Kind.String()))
// 	}
// }

func (p *Parser) typecheckStructLiteral(scope *Scope, literal *ast.StructLiteral) {
	name := literal.Name.String()
	def, ok := scope.GetStructDefinition(name)
	if !ok {
		p.addErrorToken(fmt.Errorf("Undeclared \"%s :: struct\"", name), literal.Name)
		return
	}
	if len(def.Fields) == 0 && len(literal.Fields) > 0 {
		p.addErrorToken(fmt.Errorf("Struct %s does not have any fields.", name), literal.Name)
		return
	}
	literal.TypeInfo = p.typeinfo.get(name)
	if literal.TypeInfo == nil {
		p.fatalErrorToken(fmt.Errorf("Missing type info for \"%s :: struct\".", name), literal.Name)
		return
	}

	// Check literal against definition
	for _, defField := range def.Fields {
		defTypeInfo := defField.Expression.TypeInfo
		if defTypeInfo == nil {
			p.fatalErrorToken(fmt.Errorf("Missing type info from field \"%s\" on \"%s :: struct\".", defField.Name.String(), name), defField.Name)
			return
		}
		defName := defField.Name.String()
		for i := 0; i < len(literal.Fields); i++ {
			property := &literal.Fields[i]
			if defName != property.Name.String() {
				continue
			}
			p.typecheckExpression(scope, &property.Expression)
			litTypeInfo := property.Expression.TypeInfo
			if litTypeInfo != defTypeInfo {
				p.addErrorToken(fmt.Errorf("Mismatching type, expected \"%s\" but got \"%s\"", defField.TypeIdentifier.Name.String(), property.Expression.TypeInfo.String()), property.Name)
			}
		}
	}

	for _, property := range literal.Fields {
		propertyName := property.Name.String()
		hasFieldNameOnDef := false
		for _, defField := range def.Fields {
			hasFieldNameOnDef = hasFieldNameOnDef || defField.Name.String() == propertyName
		}
		if !hasFieldNameOnDef {
			p.addErrorToken(fmt.Errorf("Field \"%s\" does not exist on \"%s :: struct\"", propertyName, name), property.Name)
		}
	}
}

func (p *Parser) typecheckArrayLiteral(scope *Scope, literal *ast.ArrayLiteral) {
	//test := [][]string{
	//	[]string{"test"}
	//}
	//if len(test) > 0 {
	//
	//}

	typeIdentName := literal.TypeIdentifier.Name
	typeIdentString := typeIdentName.String()
	typeInfo := p.DetermineType(&literal.TypeIdentifier)
	if typeInfo == nil {
		p.addErrorToken(fmt.Errorf("Undeclared type \"%s\" used for array literal", typeIdentString), typeIdentName)
		return
	}
	literal.TypeInfo = typeInfo

	//
	resultTypeInfo, ok := typeInfo.(*TypeInfo_Array)
	if !ok {
		p.fatalErrorToken(fmt.Errorf("Expected array type but got \"%s\".", typeIdentString), typeIdentName)
		return
	}
	underlyingTypeInfo := resultTypeInfo.Underlying()

	// Run type checking on each array element
	nodes := literal.Nodes()
	for i, itNode := range nodes {
		switch node := itNode.(type) {
		case *ast.Expression:
			// NOTE(Jake): Set to 'string' type info so
			//			   type checking will catch things immediately
			//			   when we call `typecheckExpression`
			//			   ie. Won't infer, will mark as invalid.
			if node.TypeInfo == nil {
				node.TypeInfo = underlyingTypeInfo
			}
			p.typecheckExpression(scope, node)

			if node.TypeInfo == nil {
				panic(fmt.Sprintf("typecheckArrayLiteral: Missing type on array literal item #%d.", i))
			}
			continue
		}
		panic(fmt.Sprintf("typecheckArrayLiteral: Unhandled type: %T", itNode))
	}
}

func (p *Parser) typecheckCall(scope *Scope, node *ast.Call, resultTypeInfo types.TypeInfo) {
	typeInfo := p.typeinfo.get(node.Name.String())
	callTypeInfo, ok := typeInfo.(*TypeInfo_Procedure)
	if !ok {
		p.addErrorToken(fmt.Errorf("Expected %s to be a procedure.", node.Name.String()), node.Name)
		return
	}
	//panic("todo: Typecheck the parameters")
	procDefinition := callTypeInfo.Definition()
	expectedTypeInfo := procDefinition.TypeInfo
	if resultTypeInfo == nil {
		resultTypeInfo = expectedTypeInfo
	}
	if !TypeEquals(resultTypeInfo, expectedTypeInfo) {
		p.addErrorToken(fmt.Errorf("Cannot mix call type %s with %s", expectedTypeInfo.String(), resultTypeInfo.String()), node.Name)
	}
	parameters := node.Parameters
	definitionParameters := procDefinition.Parameters
	hasMismatchingTypes := len(definitionParameters) != len(parameters)
	for i := 0; i < len(parameters); i++ {
		parameter := parameters[i]
		p.typecheckExpression(scope, &parameter.Expression)
		if hasMismatchingTypes == false && i < len(definitionParameters) {
			definitionParameter := definitionParameters[i]
			if !TypeEquals(parameter.TypeInfo, definitionParameter.TypeInfo) {
				hasMismatchingTypes = true
			}
		}
	}
	if hasMismatchingTypes {
		haveStr := "("
		for i := 0; i < len(parameters); i++ {
			parameter := parameters[i]
			if i != 0 {
				haveStr += ", "
			}
			if parameter.TypeInfo == nil {
				// NOTE(Jake): 2018-01-03
				//
				// This should be applied in p.typecheckExpression(scope, &parameter.Expression)
				//
				haveStr += "missing"
				continue
			}
			haveStr += parameter.TypeInfo.String()
		}
		haveStr += ")"
		wantStr := "("
		for i := 0; i < len(procDefinition.Parameters); i++ {
			parameter := procDefinition.Parameters[i]
			if i != 0 {
				wantStr += ", "
			}
			if parameter.TypeInfo == nil {
				// NOTE(Jake): 2018-01-03
				//
				// This should be applied in p.typecheckExpression(scope, &parameter.Expression)
				//
				haveStr += "missing"
				continue
			}
			wantStr += parameter.TypeInfo.String()
		}
		wantStr += ")"
		callStr := node.Name.String()
		if len(definitionParameters) != len(parameters) {
			p.addErrorToken(fmt.Errorf("Expected %d parameters, instead got %d parameters on call \"%s\".\nhave %s\nwant %s", len(definitionParameters), len(parameters), callStr, haveStr, wantStr), node.Name)
		} else {
			p.addErrorToken(fmt.Errorf("Mismatching types on call \"%s\".\nhave %s\nwant %s", callStr, haveStr, wantStr), node.Name)
		}
	}
}

func (p *Parser) typecheckExpression(scope *Scope, expression *ast.Expression) {
	resultTypeInfo := expression.TypeInfo

	// Get type info from text (ie. "string", "int", etc)
	if typeIdent := expression.TypeIdentifier.Name; resultTypeInfo == nil && typeIdent.Kind != token.Unknown {
		typeIdentString := typeIdent.String()
		resultTypeInfo = p.DetermineType(&expression.TypeIdentifier)
		if resultTypeInfo == nil {
			p.addErrorToken(fmt.Errorf("Undeclared type %s", typeIdentString), typeIdent)
			return
		}
	}

	var leftToken token.Token
	nodes := expression.Nodes()
	for i, itNode := range nodes {
		switch node := itNode.(type) {
		case *ast.StructLiteral:
			p.typecheckStructLiteral(scope, node)
			expectedTypeInfo := node.TypeInfo
			if expectedTypeInfo == nil {
				p.fatalError(fmt.Errorf("Missing type info for \"%s :: struct\".", node.Name.String()))
				continue
			}
			if resultTypeInfo == nil {
				resultTypeInfo = expectedTypeInfo
			}
			if !TypeEquals(resultTypeInfo, expectedTypeInfo) {
				p.addErrorToken(fmt.Errorf("Cannot mix struct literal %s with %s", expectedTypeInfo.String(), resultTypeInfo.String()), node.Name)
			}
			continue
		case *ast.ArrayLiteral:
			p.typecheckArrayLiteral(scope, node)
			expectedTypeInfo := node.TypeInfo
			if resultTypeInfo == nil {
				resultTypeInfo = expectedTypeInfo
			}
			if !TypeEquals(resultTypeInfo, expectedTypeInfo) {
				p.addErrorToken(fmt.Errorf("Cannot mix array literal %s with %s", expectedTypeInfo.String(), resultTypeInfo.String()), node.TypeIdentifier.Name)
			}
			continue
		case *ast.Call:
			p.typecheckCall(scope, node, resultTypeInfo)
			/*typeInfo := p.typeinfo.get(node.Name.String())
			callTypeInfo, ok := typeInfo.(*TypeInfo_Procedure)
			if !ok {
				p.addErrorToken(fmt.Errorf("Expected %s to be a procedure.", node.Name.String()), node.Name)
			}
			//panic("todo: Typecheck the parameters")
			procDefinition := callTypeInfo.Definition()
			expectedTypeInfo := procDefinition.TypeInfo
			if resultTypeInfo == nil {
				resultTypeInfo = expectedTypeInfo
			}
			if !TypeEquals(resultTypeInfo, expectedTypeInfo) {
				p.addErrorToken(fmt.Errorf("Cannot mix call type %s with %s", expectedTypeInfo.String(), resultTypeInfo.String()), node.Name)
			}
			parameters := node.Parameters
			definitionParameters := procDefinition.Parameters
			hasMismatchingTypes := len(definitionParameters) != len(parameters)
			for i := 0; i < len(parameters); i++ {
				parameter := parameters[i]
				p.typecheckExpression(scope, &parameter.Expression)
				if hasMismatchingTypes == false && i < len(definitionParameters) {
					definitionParameter := definitionParameters[i]
					if !TypeEquals(parameter.TypeInfo, definitionParameter.TypeInfo) {
						hasMismatchingTypes = true
					}
				}
			}
			if hasMismatchingTypes {
				haveStr := "("
				for i := 0; i < len(parameters); i++ {
					parameter := parameters[i]
					if i != 0 {
						haveStr += ", "
					}
					if parameter.TypeInfo == nil {
						// NOTE(Jake): 2018-01-03
						//
						// This should be applied in p.typecheckExpression(scope, &parameter.Expression)
						//
						haveStr += "missing"
						continue
					}
					haveStr += parameter.TypeInfo.String()
				}
				haveStr += ")"
				wantStr := "("
				for i := 0; i < len(procDefinition.Parameters); i++ {
					parameter := procDefinition.Parameters[i]
					if i != 0 {
						wantStr += ", "
					}
					if parameter.TypeInfo == nil {
						// NOTE(Jake): 2018-01-03
						//
						// This should be applied in p.typecheckExpression(scope, &parameter.Expression)
						//
						haveStr += "missing"
						continue
					}
					wantStr += parameter.TypeInfo.String()
				}
				wantStr += ")"
				callStr := node.Name.String()
				if len(definitionParameters) != len(parameters) {
					p.addErrorToken(fmt.Errorf("Expected %d parameters, instead got %d parameters on call \"%s\".\nhave %s\nwant %s", len(definitionParameters), len(parameters), callStr, haveStr, wantStr), node.Name)
				} else {
					p.addErrorToken(fmt.Errorf("Mismatching types on call \"%s\".\nhave %s\nwant %s", callStr, haveStr, wantStr), node.Name)
				}
			}*/
			continue
		case *ast.HTMLBlock:
			panic("typecheckExpression: todo(Jake): Fix HTMLBlock")
			/*variableType := data.KindHTMLNode
			if exprType == data.KindUnknown {
				exprType = variableType
			}
			if exprType != variableType {
				p.addErrorToken(fmt.Errorf("\":: html\" must be a %s not %s.", exprType.String(), variableType.String()), node.HTMLKeyword)
			}
			p.typecheckHTMLBlock(node, scope)*/
		case *ast.TokenList:
			tokens := node.Tokens()
			concatPropertyName := tokens[0].String()
			for i := 1; i < len(tokens); i++ {
				concatPropertyName += "." + tokens[i].String()
			}
			expectedTypeInfo := p.getTypeFromLeftHandSide(tokens, scope)
			if resultTypeInfo == nil {
				resultTypeInfo = expectedTypeInfo
			}
			if !TypeEquals(resultTypeInfo, expectedTypeInfo) {
				p.addErrorToken(fmt.Errorf("Cannot mix variable \"%s\" type %s with %s", concatPropertyName, expectedTypeInfo.String(), resultTypeInfo.String()), tokens[0])
			}
			//panic("todo(Jake): tokenlist")
			continue
		case *ast.Token:
			if node.IsOperator() {
				continue
			}

			var opToken token.Token
			if i+1 < len(nodes) {
				switch node := nodes[i+1].(type) {
				case *ast.Token:
					opToken = node.Token
				case *ast.TokenList:
					opToken = node.Tokens()[0]
				default:
					panic("Expected *ast.Token or *ast.TokenList")
				}
			}

			switch node.Kind {
			case token.Identifier:
				name := node.String()
				variableTypeInfo, ok := scope.Get(name)
				if !ok {
					_, ok := scope.GetHTMLDefinition(name)
					if ok {
						p.addErrorToken(fmt.Errorf("Undeclared identifier \"%s\". Did you mean \"%s()\" or \"%s{ }\" to reference the \"%s :: html\" component?", name, name, name, name), node.Token)
						continue
					}
					p.addErrorToken(fmt.Errorf("Undeclared identifier \"%s\".", name), node.Token)
					continue
				}
				if resultTypeInfo == nil {
					resultTypeInfo = variableTypeInfo
				}
				if !TypeEquals(resultTypeInfo, variableTypeInfo) {
					if variableTypeInfo == nil {
						p.fatalErrorToken(fmt.Errorf("Unable to determine type, typechecker must have failed."), node.Token)
						return
					}
					p.addErrorToken(fmt.Errorf("Identifier \"%s\" must be a %s not %s.", name, resultTypeInfo.String(), variableTypeInfo.String()), node.Token)
				}
			case token.String:
				expectedTypeInfo := p.typeinfo.NewTypeInfoString()
				if resultTypeInfo == nil {
					resultTypeInfo = expectedTypeInfo
				}
				if !TypeEquals(resultTypeInfo, expectedTypeInfo) {
					p.addErrorToken(fmt.Errorf("Cannot %s (%s) %s %s (\"%s\"), mismatching types.", resultTypeInfo.String(), leftToken.String(), opToken.String(), expectedTypeInfo.String(), node.String()), node.Token)
				}
			case token.Number:
				IntTypeInfo := p.typeinfo.NewTypeInfoInt()
				FloatTypeInfo := p.typeinfo.NewTypeInfoFloat()

				if resultTypeInfo == nil {
					resultTypeInfo = IntTypeInfo
					if strings.ContainsRune(node.Data, '.') {
						resultTypeInfo = FloatTypeInfo
					}
				}
				if !TypeEquals(resultTypeInfo, IntTypeInfo) && !TypeEquals(resultTypeInfo, FloatTypeInfo) {
					p.addErrorToken(fmt.Errorf("Cannot use %s with number \"%s\"", resultTypeInfo.String(), node.String()), node.Token)
				}
			case token.KeywordTrue, token.KeywordFalse:
				expectedTypeInfo := types.Bool()
				if resultTypeInfo == nil {
					resultTypeInfo = expectedTypeInfo
				}
				if !TypeEquals(resultTypeInfo, expectedTypeInfo) {
					p.addErrorToken(fmt.Errorf("Cannot use %s with %s \"%s\"", resultTypeInfo.String(), expectedTypeInfo.String(), node.String()), node.Token)
				}
			default:
				panic(fmt.Sprintf("typecheckExpression: Unhandled token kind: \"%s\" with value: %s", node.Kind.String(), node.String()))
			}
			leftToken = node.Token
			continue
		}
		panic(fmt.Sprintf("typecheckExpression: Unhandled type %T", itNode))
	}

	expression.TypeInfo = resultTypeInfo
}

func (p *Parser) typecheckHTMLBlock(htmlBlock *ast.HTMLBlock, scope *Scope) {
	scope = NewScope(scope)
	p.typecheckStatements(htmlBlock, scope)
}

func (p *Parser) getTypeFromLeftHandSide(leftHandSideTokens []token.Token, scope *Scope) types.TypeInfo {
	nameToken := leftHandSideTokens[0]
	name := nameToken.String()
	variableTypeInfo, ok := scope.Get(name)
	if !ok {
		p.addErrorToken(fmt.Errorf("Undeclared variable \"%s\".", name), nameToken)
		return nil
	}
	if variableTypeInfo == nil {
		p.fatalErrorToken(fmt.Errorf("Variable declared \"%s\" but it has \"nil\" type information, this is a bug in typechecking scope.", name), nameToken)
		return nil
	}
	var concatPropertyName bytes.Buffer
	concatPropertyName.WriteString(name)
	for i := 1; i < len(leftHandSideTokens); i++ {
		propertyName := leftHandSideTokens[i].String()
		concatPropertyName.WriteString(".")
		concatPropertyName.WriteString(propertyName)
		structInfo, ok := variableTypeInfo.(*TypeInfo_Struct)
		if !ok {
			p.addErrorToken(fmt.Errorf("Property \"%s\" does not exist on type \"%s\".", concatPropertyName.String(), variableTypeInfo.String()), nameToken)
			return nil
		}
		structDef := structInfo.Definition()
		structField := structDef.GetFieldByName(propertyName)
		if structField == nil {
			p.addErrorToken(fmt.Errorf("Property \"%s\" does not exist on \"%s :: struct\".", concatPropertyName.String(), structDef.Name), nameToken)
			return nil
		}
		variableTypeInfo = structField.TypeInfo
	}
	return variableTypeInfo
}

func (p *Parser) typecheckHTMLDefinition(htmlDefinition *ast.HTMLComponentDefinition, parentScope *Scope) {
	// Attach CSSDefinition if found
	name := htmlDefinition.Name.String()
	if cssDefinition, ok := parentScope.GetCSSDefinition(name); ok {
		htmlDefinition.CSSDefinition = cssDefinition
	}

	// Attach CSSConfigDefinition if found
	if cssConfigDefinition, ok := parentScope.GetCSSConfigDefinition(name); ok {
		htmlDefinition.CSSConfigDefinition = cssConfigDefinition
	}

	// Attach StructDefinition if found
	if structDef, ok := parentScope.GetStructDefinition(name); ok {
		if htmlDefinition.Struct != nil {
			anonymousStructDef := htmlDefinition.Struct
			p.addErrorToken(fmt.Errorf("Cannot have  \"%s :: struct\" and embedded \":: struct\" inside \"%s :: html\"", structDef.Name.String(), htmlDefinition.Name.String()), anonymousStructDef.Name)
		} else {
			htmlDefinition.Struct = structDef
		}
	}

	//
	var globalScopeNoVariables Scope = *parentScope
	globalScopeNoVariables.identifiers = nil
	scope := NewScope(&globalScopeNoVariables)
	scope.Set("children", types.HTMLNode())

	if structDef := htmlDefinition.Struct; structDef != nil {
		for i, _ := range structDef.Fields {
			var propertyNode *ast.StructField = &structDef.Fields[i]
			p.typecheckExpression(scope, &propertyNode.Expression)
			name := propertyNode.Name.String()
			_, ok := scope.Get(name)
			if ok {
				if name == "children" {
					p.addErrorToken(fmt.Errorf("Cannot use \"children\" as it's a reserved property."), propertyNode.Name)
					continue
				}
				p.addErrorToken(fmt.Errorf("Property \"%s\" declared twice.", name), propertyNode.Name)
				continue
			}
			scope.Set(name, propertyNode.TypeInfo)
		}
	}

	if p.typecheckHtmlNodeDependencies != nil {
		panic("typecheckHtmlNodeDependencies must be nil before being re-assigned")
	}
	p.typecheckHtmlNodeDependencies = make(map[string]*ast.HTMLNode)
	p.typecheckStatements(htmlDefinition, scope)
	htmlDefinition.Dependencies = p.typecheckHtmlNodeDependencies
	p.typecheckHtmlNodeDependencies = nil
}

func (p *Parser) typecheckStatements(topNode ast.Node, scope *Scope) {
	nodeStack := make([]ast.Node, 0, 50)
	nodes := topNode.Nodes()
	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		nodeStack = append(nodeStack, node)
	}

	//Loop:
	for len(nodeStack) > 0 {
		itNode := nodeStack[len(nodeStack)-1]
		nodeStack = nodeStack[:len(nodeStack)-1]
		avoidNestingScopeThisIteration := false

		if itNode == nil {
			scope = scope.parent
			continue
		}

		switch node := itNode.(type) {
		case *ast.CSSDefinition,
			*ast.CSSConfigDefinition,
			*ast.HTMLComponentDefinition,
			*ast.StructDefinition,
			*ast.ProcedureDefinition:
			// Skip nodes and child nodes
			continue
		case *ast.Call:
			p.typecheckCall(scope, node, nil)
		case *ast.HTMLBlock:
			panic("todo(Jake): Remove below commented out line if this is unused.")
			//p.typecheckHTMLBlock(node, scope)
		case *ast.HTMLNode:
			p.typecheckExpression(scope, &node.IfExpression)

			for i, _ := range node.Parameters {
				p.typecheckExpression(scope, &node.Parameters[i].Expression)
			}

			name := node.Name.String()
			isValidHTML5TagName := util.IsValidHTML5TagName(name)
			if isValidHTML5TagName {
				continue
			}
			htmlComponentDefinition, ok := scope.GetHTMLDefinition(name)
			if !ok {
				if name != strings.ToLower(name) {
					// NOTE(Jake): 2017-12-28
					//			   Hit this error while developing and decided that if the convention is that non-HTML5 spec
					//			   elements
					p.addErrorToken(fmt.Errorf("\"%s\" is an undefined component. If you want to use a standard HTML5 element, the name must all be in lowercase.", name), node.Name)
					continue
				}
				p.addErrorToken(fmt.Errorf("\"%s\" is not a valid HTML5 element or defined component.", name), node.Name)
				continue
			}
			//fmt.Printf("%s -- %d\n", htmlComponentDefinition.Name.String(), len(p.typecheckHtmlDefinitionStack))
			//for _, itHtmlDefinition := range p.typecheckHtmlDefinitionStack {
			//	if htmlComponentDefinition == itHtmlDefinition {
			//		p.addErrorLine(fmt.Errorf("Cannot reference self in \"%s :: html\".", htmlComponentDefinition.Name.String()), node.Name.Line)
			//		//continue Loop
			//		return
			//	}
			//}
			if p.typecheckHtmlNodeDependencies != nil {
				p.typecheckHtmlNodeDependencies[name] = node
			}
			node.HTMLDefinition = htmlComponentDefinition
			structDef := node.HTMLDefinition.Struct
			if structDef != nil && len(structDef.Fields) > 0 {
				// Check if parameters exist
			ParameterCheckLoop:
				for i, _ := range node.Parameters {
					parameterNode := node.Parameters[i]
					paramName := parameterNode.Name.String()
					for _, field := range structDef.Fields {
						if paramName == field.Name.String() {
							parameterType := parameterNode.TypeInfo
							componentStructType := field.TypeInfo
							if parameterType != componentStructType {
								if field.TypeInfo == nil {
									p.fatalError(fmt.Errorf("Struct field \"%s\" is missing type info.", paramName))
									return
								}
								p.addErrorToken(fmt.Errorf("\"%s\" must be of type %s, not %s", paramName, componentStructType.String(), parameterType.String()), parameterNode.Name)
							}
							continue ParameterCheckLoop
						}
					}
					p.addErrorToken(fmt.Errorf("\"%s\" is not a property on \"%s :: html\"", paramName, name), parameterNode.Name)
					continue
				}
			}
		case *ast.Return:
			p.typecheckExpression(scope, &node.Expression)
			continue
		case *ast.ArrayAppendStatement:
			nameToken := node.LeftHandSide[0]
			name := nameToken.String()
			arrayTypeInfo := p.getTypeFromLeftHandSide(node.LeftHandSide, scope)
			if arrayTypeInfo == nil {
				continue
			}
			variableTypeInfo, isArray := arrayTypeInfo.(*TypeInfo_Array)
			if !isArray {
				// todo(Jake): 2017-12-25, Retrieve the whole string for the token list and use instead of "name"
				p.addErrorToken(fmt.Errorf("Cannot do \"%s []=\" as it's not an array type.", name), nameToken)
				continue
			}
			p.typecheckExpression(scope, &node.Expression)
			resultTypeInfo := node.Expression.TypeInfo
			if !TypeEquals(variableTypeInfo.Underlying(), resultTypeInfo) {
				// todo(Jake): 2017-12-25, test these... as the error messages are untested
				p.addErrorToken(fmt.Errorf("Cannot change \"%s\" from %s to %s", name, variableTypeInfo, resultTypeInfo.String()), nameToken)
				p.addErrorToken(fmt.Errorf("Cannot change \"%s\" from %s to %s", name, variableTypeInfo, resultTypeInfo.String()), nameToken)
			}
			continue
		case *ast.OpStatement:
			variableTypeInfo := p.getTypeFromLeftHandSide(node.LeftHandSide, scope)
			if variableTypeInfo == nil {
				continue
			}
			p.typecheckExpression(scope, &node.Expression)
			resultTypeInfo := node.Expression.TypeInfo
			if !TypeEquals(variableTypeInfo, resultTypeInfo) {
				nameToken := node.LeftHandSide[0]
				if variableTypeInfo == nil {
					p.fatalErrorToken(fmt.Errorf("\"variableTypeInfo\" is nil, this should have a value."), nameToken)
					continue
				}
				if resultTypeInfo == nil {
					p.fatalErrorToken(fmt.Errorf("\"resultTypeInfo\" is nil, this should have a value."), nameToken)
					continue
				}
				name := nameToken.String()
				for i := 1; i < len(node.LeftHandSide); i++ {
					name += "." + node.LeftHandSide[i].String()
				}
				p.addErrorToken(fmt.Errorf("Cannot change \"%s\" from %s to %s", name, variableTypeInfo, resultTypeInfo.String()), nameToken)
			}
			continue
		case *ast.DeclareStatement:
			expr := &node.Expression
			p.typecheckExpression(scope, expr)
			name := node.Name.String()
			_, ok := scope.GetFromThisScope(name)
			if ok {
				p.addErrorToken(fmt.Errorf("Cannot redeclare \"%s\".", name), node.Name)
				continue
			}
			scope.Set(name, expr.TypeInfo)
			continue
		case *ast.Expression:
			p.typecheckExpression(scope, node)
			continue
		case *ast.If:
			expr := &node.Condition
			//expr.TypeInfo = types.Bool()
			p.typecheckExpression(scope, expr)

			scope = NewScope(scope)
			nodeStack = append(nodeStack, nil)
			avoidNestingScopeThisIteration = true

			// Add if true children
			{
				nodes := node.Nodes()
				for i := len(nodes) - 1; i >= 0; i-- {
					nodeStack = append(nodeStack, nodes[i])
				}
			}
			{
				// Add else children
				nodes := node.ElseNodes
				for i := len(nodes) - 1; i >= 0; i-- {
					nodeStack = append(nodeStack, nodes[i])
				}
			}
			continue
		case *ast.Block:
			// no-op, will jump to adding child nodes / new scope below
		case *ast.For:
			if !node.IsDeclareSet {
				panic("todo(Jake): handle array without declare set")
			}
			p.typecheckExpression(scope, &node.Array)
			iTypeInfo := node.Array.TypeInfo
			typeInfo, ok := iTypeInfo.(*TypeInfo_Array)
			if !ok {
				p.addErrorToken(fmt.Errorf("Cannot use type %s as array.", iTypeInfo.String()), node.RecordName)
				continue
			}
			if node.IsDeclareSet {
				// Nest scope
				// - Earlier nesting so we declare variables in the `for` line rather
				//	 then only after the {
				//
				// WARNING: Ensure nothing else appends to `nodeStack` after this.
				//
				scope = NewScope(scope)
				nodeStack = append(nodeStack, nil)
				avoidNestingScopeThisIteration = true
			}

			// Add "i" to scope if used
			if node.IndexName.Kind != token.Unknown {
				indexName := node.IndexName.String()
				_, ok = scope.GetFromThisScope(indexName)
				if ok {
					p.addErrorToken(fmt.Errorf("Cannot redeclare \"%s\" in for-loop.", indexName), node.IndexName)
					continue
				}
				scope.Set(indexName, p.typeinfo.NewTypeInfoInt())
			}

			// Set left-hand value type
			name := node.RecordName.String()
			_, ok = scope.GetFromThisScope(name)
			if ok {
				p.addErrorToken(fmt.Errorf("Cannot redeclare \"%s\" in for-loop.", name), node.RecordName)
				continue
			}
			scope.Set(name, typeInfo.Underlying())
		default:
			panic(fmt.Sprintf("TypecheckStatements: Unknown type %T", node))
		}

		// Nest scope
		if !avoidNestingScopeThisIteration {
			scope = NewScope(scope)
			nodeStack = append(nodeStack, nil)
		}

		// Add children
		nodes := itNode.Nodes()
		for i := len(nodes) - 1; i >= 0; i-- {
			nodeStack = append(nodeStack, nodes[i])
		}
	}
}

func (p *Parser) typecheckStruct(node *ast.StructDefinition, scope *Scope) {
	// Add typeinfo to each struct field
	for i := 0; i < len(node.Fields); i++ {
		structField := &node.Fields[i]
		typeIdent := structField.TypeIdentifier.Name
		if typeIdent.Kind == token.Unknown {
			p.addErrorToken(fmt.Errorf("Missing type identifier on \"%s :: struct\" field \"%s\"", structField.Name, node.Name.String()), structField.Name)
			continue
		}
		typeIdentString := typeIdent.String()
		resultTypeInfo := p.DetermineType(&structField.TypeIdentifier)
		if resultTypeInfo == nil {
			p.addErrorToken(fmt.Errorf("Undeclared type %s", typeIdentString), typeIdent)
			return
		}
		structField.TypeInfo = resultTypeInfo
	}
}

func (p *Parser) typecheckProcedureDefinition(node *ast.ProcedureDefinition, scope *Scope) {
	errorCount := 0
	for i := 0; i < len(node.Parameters); i++ {
		parameter := &node.Parameters[i]
		typeinfo := p.DetermineType(&parameter.TypeIdentifier)
		if typeinfo == nil {
			p.addErrorToken(fmt.Errorf("Unknown type %s on parameter %s", parameter.TypeIdentifier.String(), parameter.Name), parameter.TypeIdentifier.Name)
			continue
		}
		parameter.TypeInfo = typeinfo
		scope.Set(parameter.Name.String(), typeinfo)
	}
	if errorCount > 0 {
		return
	}
	p.typecheckStatements(node, scope)

	var returnType types.TypeInfo
	if node.TypeIdentifier.Name.Kind != token.Unknown {
		returnType = p.DetermineType(&node.TypeIdentifier)
		node.TypeInfo = returnType
	}
	// Check return statements
	// NOTE(Jake): 2017-12-30
	//
	// The code below is probably super slow compared
	// to if we just added a flag to `typecheckStatements`
	//
	nodes := node.ChildNodes
	nodeStack := make([]ast.Node, 0, len(nodes))
	for i := len(nodes) - 1; i >= 0; i-- {
		nodeStack = append(nodeStack, nodes[i])
	}
	for len(nodeStack) > 0 {
		node := nodeStack[len(nodeStack)-1]
		nodeStack = nodeStack[:len(nodeStack)-1]

		returnNode, ok := node.(*ast.Return)
		if ok {
			if TypeEquals(returnType, returnNode.TypeInfo) {
				continue
			}
			t := returnNode.TypeIdentifier.Name
			if returnType == nil {
				p.addErrorToken(fmt.Errorf("Return statement %s doesn't match procedure type void", returnNode.TypeInfo.String()), t)
				continue
			}
			if returnNode.TypeInfo == nil {
				p.addErrorToken(fmt.Errorf("Return statement void doesn't match procedure type %s", returnType.String()), t)
				continue
			}
			p.addErrorToken(fmt.Errorf("Return statement %s doesn't match procedure type %s", returnNode.TypeInfo.String(), returnType.String()), t)
			continue
		}

		nodes := node.Nodes()
		for i := len(nodes) - 1; i >= 0; i-- {
			nodeStack = append(nodeStack, nodes[i])
		}
	}

	functionType := p.typeinfo.NewProcedureInfo(node)
	p.typeinfo.register(node.Name.String(), functionType)
}

func (p *Parser) TypecheckFile(file *ast.File, globalScope *Scope) {
	scope := NewScope(globalScope)
	p.typecheckStatements(file, scope)
}

func (p *Parser) TypecheckAndFinalize(files []*ast.File) {
	globalScope := NewScope(nil)

	// Get all global/top-level identifiers
	for _, file := range files {
		scope := globalScope
		for _, itNode := range file.ChildNodes {
			switch node := itNode.(type) {
			case *ast.HTMLNode,
				*ast.DeclareStatement,
				*ast.OpStatement,
				*ast.ArrayAppendStatement,
				*ast.Expression,
				*ast.If,
				*ast.For,
				*ast.Block,
				*ast.HTMLBlock,
				*ast.Call:
				// no-op, these are checked in TypecheckFile()
			case *ast.ProcedureDefinition:
				if node == nil {
					p.fatalError(fmt.Errorf("Found nil top-level %T.", node))
					continue
				}
				p.typecheckProcedureDefinition(node, NewScope(scope))
			case *ast.StructDefinition:
				if node == nil {
					p.fatalError(fmt.Errorf("Found nil top-level %T.", node))
					continue
				}
				if node.Name.Kind == token.Unknown {
					p.addErrorToken(fmt.Errorf("Cannot declare anonymous \":: struct\" block."), node.Name)
					continue
				}
				name := node.Name.String()
				existingNode, ok := scope.structDefinitions[name]
				if ok {
					errorMessage := fmt.Errorf("Cannot declare \"%s :: struct\" more than once in global scope.", name)
					p.addErrorToken(errorMessage, existingNode.Name)
					p.addErrorToken(errorMessage, node.Name)
					continue
				}

				p.typecheckStruct(node, scope)

				scope.structDefinitions[name] = node
				typeinfo := p.typeinfo.NewStructInfo(node)
				p.typeinfo.register(name, typeinfo)
			case *ast.HTMLComponentDefinition:
				if node == nil {
					p.fatalError(fmt.Errorf("Found nil top-level %T.", node))
					continue
				}
				if node.Name.Kind == token.Unknown {
					p.addErrorToken(fmt.Errorf("Cannot declare anonymous \":: html\" block."), node.Name)
					continue
				}
				name := node.Name.String()
				existingNode, ok := scope.htmlDefinitions[name]
				if ok {
					errorMessage := fmt.Errorf("Cannot declare \"%s :: html\" more than once in global scope.", name)
					p.addErrorToken(errorMessage, existingNode.Name)
					p.addErrorToken(errorMessage, node.Name)
					continue
				}
				scope.htmlDefinitions[name] = node
			case *ast.CSSDefinition:
				if node == nil {
					p.fatalError(fmt.Errorf("Found nil top-level %T.", node))
					continue
				}
				if node.Name.Kind == token.Unknown {
					continue
				}
				name := node.Name.String()
				existingNode, ok := scope.cssDefinitions[name]
				if ok {
					errorMessage := fmt.Errorf("Cannot declare \"%s :: css\" more than once in global scope.", name)
					p.addErrorToken(errorMessage, existingNode.Name)
					p.addErrorToken(errorMessage, node.Name)
					continue
				}
				scope.cssDefinitions[name] = node
			case *ast.CSSConfigDefinition:
				if node == nil {
					p.fatalError(fmt.Errorf("Found nil top-level %T.", node))
					continue
				}
				if node.Name.Kind == token.Unknown {
					p.addErrorToken(fmt.Errorf("Cannot declare anonymous \":: css_config\" block."), node.Name)
					continue
				}
				name := node.Name.String()
				existingNode, ok := scope.cssConfigDefinitions[name]
				if ok {
					errorMessage := fmt.Errorf("Cannot declare \"%s :: css_config\" more than once in global scope.", name)
					p.addErrorToken(errorMessage, existingNode.Name)
					p.addErrorToken(errorMessage, node.Name)
					continue
				}
				scope.cssConfigDefinitions[name] = node
			default:
				panic(fmt.Sprintf("TypecheckAndFinalize: Unknown type %T", node))
			}
		}
	}

	//
	for _, htmlDefinition := range globalScope.htmlDefinitions {
		p.typecheckHTMLDefinition(htmlDefinition, globalScope)
	}

	// Check if CSS config matches a HTML or CSS component. If not, throw error.
	for name, cssConfigDefinition := range globalScope.cssConfigDefinitions {
		_, ok := globalScope.GetCSSDefinition(name)
		if ok {
			continue
		}
		_, ok = globalScope.GetHTMLDefinition(name)
		if ok {
			continue
		}
		p.addErrorToken(fmt.Errorf("\"%s :: css_config\" has no matching \":: css\" or \":: html\" block.", name), cssConfigDefinition.Name)
	}

	// Get nested dependencies
	for _, htmlDefinition := range globalScope.htmlDefinitions {
		nodeStack := make([]*ast.HTMLNode, 0, 50)
		for _, subNode := range htmlDefinition.Dependencies {
			nodeStack = append(nodeStack, subNode)
		}
		for len(nodeStack) > 0 {
			node := nodeStack[len(nodeStack)-1]
			nodeStack = nodeStack[:len(nodeStack)-1]

			// Add child dependencies
			for _, subNode := range node.HTMLDefinition.Dependencies {
				name := subNode.Name.String()
				_, ok := htmlDefinition.Dependencies[name]
				if ok {
					continue
				}
				htmlDefinition.Dependencies[name] = subNode
				nodeStack = append(nodeStack, subNode)
			}
		}

		// Print deps
		// fmt.Printf("\n\nDependencies of %s\n", htmlDefinition.Name.String())
		// for name, _ := range htmlDefinition.Dependencies {
		// 	fmt.Printf("- %s\n", name)
		// }
	}

	// Lookup if component depends on itself
	for _, htmlDefinition := range globalScope.htmlDefinitions {
		name := htmlDefinition.Name.String()
		node, ok := htmlDefinition.Dependencies[name]
		if !ok {
			continue
		}
		p.addErrorToken(fmt.Errorf("Cannot use \"%s\". Cyclic references are not allowed.", name), node.Name)
	}

	// Typecheck
	for _, file := range files {
		p.TypecheckFile(file, globalScope)
	}
}
