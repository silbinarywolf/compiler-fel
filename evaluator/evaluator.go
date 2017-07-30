package evaluator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/silbinarywolf/compiler-fel/ast"
	"github.com/silbinarywolf/compiler-fel/parser"
)

type Program struct {
	globalScope *Scope
}

func New() *Program {
	p := new(Program)
	p.globalScope = new(Scope)
	return p
}

func (program *Program) RunProject(projectDirpath string) error {
	// Get all files in folder recursively with *.fel
	filepathSet := make([]string, 0, 50)
	err := filepath.Walk(projectDirpath, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".fel" {
			filepathSet = append(filepathSet, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("An error occurred reading: %v, Error Message: %v", projectDirpath, err)
	}

	if len(filepathSet) == 0 {
		return fmt.Errorf("No *.fel files found in your project directory: %v", projectDirpath)
	}

	// Parse files
	astFiles := make([]*ast.File, 0, 50)
	p := parser.New()
	for _, filepath := range filepathSet {

		// -----------------------------------------------------------
		// todo(Jake): Remove this and add parsing logic for Page.fel
		// -----------------------------------------------------------
		if filepath == "..\\testdata\\sampleproject\\fel\\templates\\Page.fel" {
			continue
		}

		filecontentsAsBytes, err := ioutil.ReadFile(filepath)
		if err != nil {
			return fmt.Errorf("An error occurred reading file: %v, Error message: %v", filepath, err)
			//continue
		}
		astFile := p.Parse(filecontentsAsBytes, filepath)
		if astFile == nil {
			panic("Unexpected parse error (Parse() returned a nil ast.File node)")
		}
		if errors := p.GetErrors(); len(errors) > 0 {
			errorOrErrors := "errors"
			if len(errors) == 1 {
				errorOrErrors = "error"
			}
			fmt.Printf("Found %d %s in \"%s\"\n", len(errors), errorOrErrors, filepath)
			for _, err := range errors {
				fmt.Printf("- %v \n\n", err)
			}
			return fmt.Errorf("Error parsing file: %v", filepath)
		}
		astFiles = append(astFiles, astFile)
	}

	// Find config first
	var configAstFile *ast.File
	for _, astFile := range astFiles {
		baseFilename := astFile.Filepath[len(projectDirpath):len(astFile.Filepath)]
		if baseFilename == "\\config.fel" {
			configAstFile = astFile
			break
		}
	}
	if configAstFile == nil {
		return fmt.Errorf("Cannot find config.fel in root of project directory: %v", projectDirpath)
	}

	program.evaluateBlock(configAstFile, program.globalScope)

	// Evaluate template files

	panic("Evaluator::RunProject(): todo(Jake): The rest of the function")
	return nil
}

func (program *Program) evaluateBlock(blockNode ast.Node, parentScope *Scope) {
	scope := &Scope{
		parent: parentScope,
	}

	nodesQueue := blockNode.Nodes()

	// DEBUG
	/*json, _ := json.MarshalIndent(nodesQueue, "", "   ")
	fmt.Printf("%s", string(json))
	panic("evaluateBlock")*/

	for len(nodesQueue) > 0 {
		currentNode := nodesQueue[0]
		nodesQueue = nodesQueue[1:]

		switch node := currentNode.(type) {
		case *ast.DeclareStatement:
			name := node.Name.String()
			if _, exists := scope.GetCurrentScope(name); exists {
				panic(fmt.Sprintf("Cannot redeclare %v", name))
			}
			//scope.Set(name,
			panic("todo(Jake): scope.Set() code")
		default:
			panic(fmt.Sprintf("Unhandled type: %T", node))
		}

		// Add children
		//nodesQueue = append(nodesQueue, currentNode.Nodes()...)
	}
	panic("Evaluator::evaluateBlock(): todo(Jake): The rest of the function")
}
