package evaluator

import (
	"bytes"
	//"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/silbinarywolf/compiler-fel/ast"
	"github.com/silbinarywolf/compiler-fel/data"
	"github.com/silbinarywolf/compiler-fel/generate"
	"github.com/silbinarywolf/compiler-fel/parser"
	"github.com/silbinarywolf/compiler-fel/token"
)

type TemplateFile struct {
	Filepath string
	Content  string
}

type Program struct {
	Filepath                    string
	globalScope                 *Scope
	htmlDefinitionUsed          map[string]*ast.HTMLComponentDefinition
	anonymousCSSDefinitionsUsed []*ast.CSSDefinition
	debugLevel                  int
}

func New() *Program {
	p := new(Program)
	p.globalScope = NewScope(nil)
	p.htmlDefinitionUsed = make(map[string]*ast.HTMLComponentDefinition)
	return p
}

func (program *Program) CreateDataType(t token.Token) data.Type {
	typename := t.String()
	switch typename {
	case "string":
		return new(data.String)
	default:
		panic(fmt.Sprintf("Unknown type name: %s", typename))
	}
}

func (program *Program) GetConfigString(configName string) (string, error) {
	value, ok := program.globalScope.Get(configName)
	if !ok {
		return "", fmt.Errorf("%s is undefined in config.fel. This definition is required.", configName)
	}
	if value.Kind() != data.KindString {
		return "", fmt.Errorf("%s is expected to be a string.", configName)
	}
	return value.String(), nil
}

func (program *Program) RunProject(projectDirpath string) error {
	totalTimeStart := time.Now()

	configFilepath := projectDirpath + "/config.fel"
	if _, err := os.Stat(configFilepath); os.IsNotExist(err) {
		return fmt.Errorf("Cannot find config.fel in root of project directory: %v", configFilepath)
	}

	// Find and parse config.fel
	var configAstFile *ast.File
	var readFileTime time.Duration

	{
		filepath := configFilepath

		p := parser.New()

		fileReadStart := time.Now()
		filecontentsAsBytes, err := ioutil.ReadFile(filepath)
		readFileTime += time.Since(fileReadStart)

		if err != nil {
			return fmt.Errorf("An error occurred reading file: %v, Error message: %v", filepath, err)
		}
		astFile := p.Parse(filecontentsAsBytes, filepath)
		if astFile == nil {
			panic("Unexpected parse error (Parse() returned a nil ast.File node)")
		}
		configAstFile = astFile
		p.TypecheckFile(configAstFile, nil)
		if p.HasErrors() {
			p.PrintErrors()
			return fmt.Errorf("Parse errors in config.fel in root of project directory")
		}
		if configAstFile == nil {
			return fmt.Errorf("Cannot find config.fel in root of project directory: %v", projectDirpath)
		}
	}

	// Evaluate config file
	for _, node := range configAstFile.Nodes() {
		program.evaluateStatement(node, program.globalScope)
	}
	//panic("Finished evaluating config file")

	// Get config variables
	templateOutputDirectory, err := program.GetConfigString("template_output_directory")
	if err != nil {
		return err
	}
	templateOutputDirectory = fmt.Sprintf("%s/%s", projectDirpath, templateOutputDirectory)
	cssOutputDirectory, err := program.GetConfigString("css_output_directory")
	if err != nil {
		return err
	}
	cssOutputDirectory = fmt.Sprintf("%s/%s", projectDirpath, cssOutputDirectory)

	//templateInputDirectory, err := program.GetConfigString("template_input_directory")
	//if err != nil {
	//	return err
	//}
	//templateInputDirectory = fmt.Sprintf("%s/%s", projectDirpath, templateInputDirectory)
	templateInputDirectory := projectDirpath + "/templates"
	// Check if input templates directory exists
	{
		_, err := os.Stat(templateInputDirectory)
		if err != nil {
			return fmt.Errorf("Error with directory \"templates\" directory in project directory: %v", err)
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("Expected to find \"templates\" directory in: %s", projectDirpath)
		}
	}

	// Check if output templates directory exists
	{
		_, err := os.Stat(templateOutputDirectory)
		if err != nil {
			return fmt.Errorf("Error with directory: %v", err)
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("template_output_directory specified does not exist: %s", templateOutputDirectory)
		}
	}

	// Get all files in folder recursively with *.fel
	filepathSet := make([]string, 0, 50)
	//templateFilepathSet := make([]string, 0, 50)
	{
		fileReadStart := time.Now()
		err := filepath.Walk(projectDirpath, func(path string, f os.FileInfo, _ error) error {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".fel" {
				filepathSet = append(filepathSet, path)
				//if strings.HasPrefix(path, templateInputDirectory) {
				//	templateFilepathSet = append(templateFilepathSet, path)
				//}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("An error occurred reading: %v, Error Message: %v", templateInputDirectory, err)
		}

		if len(filepathSet) == 0 {
			return fmt.Errorf("No *.fel files found in your project's \"templates\" directory: %v", templateInputDirectory)
		}
		readFileTime += time.Since(fileReadStart)
	}

	// Parse files
	astFiles := make([]*ast.File, 0, 50)
	p := parser.New()
	parsingStart := time.Now()
	for _, filepath := range filepathSet {
		fileReadStart := time.Now()
		filecontentsAsBytes, err := ioutil.ReadFile(filepath)
		readFileTime += time.Since(fileReadStart)
		if err != nil {
			return fmt.Errorf("An error occurred reading file: %v, Error message: %v", filepath, err)
			//continue
		}
		astFile := p.Parse(filecontentsAsBytes, filepath)
		if astFile == nil {
			panic("Unexpected parse error (Parse() returned a nil ast.File node)")
		}
		astFiles = append(astFiles, astFile)
	}
	p.TypecheckAndFinalize(astFiles)
	parsingElapsed := time.Since(parsingStart)

	if p.HasErrors() {
		p.PrintErrors()
		return fmt.Errorf("Stopping due to parsing errors.")
	}

	//fmt.Printf("File read time: %s\n", readFileTime)
	//fmt.Printf("Parsing time: %s\n", parsingElapsed)
	//panic("TESTING TYPECHECKER: Finished Typecheck.")

	/*{
		json, _ := json.MarshalIndent(astFiles, "", "   ")
		fmt.Printf("%s", string(json))
	}*/

	outputTemplateFileSet := make([]TemplateFile, 0, len(astFiles))
	outputCSSDefinitionSet := make([]*data.CSSDefinition, 0, 3)

	// Execute template
	executionStart := time.Now()
	for _, astFile := range astFiles {
		if !strings.HasPrefix(astFile.Filepath, templateInputDirectory) {
			continue
		}
		program.globalScope = NewScope(nil)

		globalScope := program.globalScope
		htmlNode := program.evaluateTemplate(astFile, globalScope)

		if len(htmlNode.ChildNodes) == 0 {
			return fmt.Errorf("No top level HTMLNode or HTMLText found in %s.", astFile.Filepath)
		}
		if htmlNode == nil {
			panic(fmt.Sprintf("No html node found in %s.", astFile.Filepath))
		}

		// Queue up to-be-printed CSS definitions
		//cssDefinitionList := globalScope.cssDefinitions
		//if len(cssDefinitionList) > 0 {
		//	for _, cssDefinition := range cssDefinitionList {
		//		// Unnamed ":: css {" blocks only
		//		if cssDefinition.Name.Kind == token.Unknown {
		//			outputCSSDefinitionSet = append(outputCSSDefinitionSet, cssDefinition)
		//		}
		//	}
		//}

		baseFilename := astFile.Filepath[len(templateInputDirectory) : len(astFile.Filepath)-4]
		outputFilepath := filepath.Clean(fmt.Sprintf("%s%s.html", templateOutputDirectory, baseFilename))
		result := TemplateFile{
			Filepath: outputFilepath,
			Content:  generate.PrettyHTML(htmlNode),
		}
		outputTemplateFileSet = append(outputTemplateFileSet, result)
	}
	executionElapsed := time.Since(executionStart)

	// Output named "MyComponent :: css" blocks
	for _, htmlDefinition := range program.htmlDefinitionUsed {
		cssDefinition := htmlDefinition.CSSDefinition
		if cssDefinition == nil {
			continue
		}
		dataCSSDefinition := program.evaluateCSSDefinition(cssDefinition, program.globalScope)
		outputCSSDefinitionSet = append(outputCSSDefinitionSet, dataCSSDefinition)
	}

	// Output anonymous ":: css" blocks
	for _, cssDefinition := range program.anonymousCSSDefinitionsUsed {
		dataCSSDefinition := program.evaluateCSSDefinition(cssDefinition, program.globalScope)
		outputCSSDefinitionSet = append(outputCSSDefinitionSet, dataCSSDefinition)
	}

	// Output
	var generateTimeElapsed time.Duration
	{
		generateStartTime := time.Now()
		// Output CSS definitions
		{
			var cssOutput bytes.Buffer
			for _, cssDefinition := range outputCSSDefinitionSet {
				name := cssDefinition.Name
				if len(name) == 0 {
					name = "<anonymous>"
				}
				cssOutput.WriteString(fmt.Sprintf("/* Name: %s */\n", name))
				cssOutput.WriteString(generate.PrettyCSS(cssDefinition))
			}
			outputFilepath := filepath.Clean(fmt.Sprintf("%s/%s.css", cssOutputDirectory, "main"))
			fmt.Printf("%s\n", outputFilepath)
			err := ioutil.WriteFile(
				outputFilepath,
				cssOutput.Bytes(),
				0644,
			)
			if err != nil {
				panic(err)
			}
		}

		// Write to file
		for _, outputTemplateFile := range outputTemplateFileSet {
			err := ioutil.WriteFile(
				outputTemplateFile.Filepath,
				[]byte(outputTemplateFile.Content),
				0644,
			)
			if err != nil {
				panic(err)
			}
		}
		generateTimeElapsed = time.Since(generateStartTime)
	}

	//fmt.Printf("templateOutputDirectory: %s\n", templateOutputDirectory)
	fmt.Printf("File read time: %s\n", readFileTime)
	fmt.Printf("Parsing time: %s\n", parsingElapsed)
	fmt.Printf("Execution time: %s\n", executionElapsed)
	fmt.Printf("Generate/File write time: %s\n", generateTimeElapsed)
	totalTimeElapsed := time.Since(totalTimeStart)
	fmt.Printf("Total time: %s\n", totalTimeElapsed)
	panic("Evaluator::RunProject(): todo(Jake): The rest of the function")
	return nil
}
