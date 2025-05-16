package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ConversionResult holds statistics about the conversion process
type ConversionResult struct {
	FilesProcessed  int
	FilesModified   int
	TablesConverted int
	Errors          []string
}

// TableTestConverter converts slice-based table tests to map-based table tests
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run tabletests.go <directory_path>")
		os.Exit(1)
	}

	directoryPath := os.Args[1]
	result, err := ConvertTableTests(directoryPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Conversion complete:\n")
	fmt.Printf("  Files processed: %d\n", result.FilesProcessed)
	fmt.Printf("  Files modified: %d\n", result.FilesModified)
	fmt.Printf("  Tables converted: %d\n", result.TablesConverted)

	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
}

// ConvertTableTests converts all slice-based table tests to map-based tables in a directory
func ConvertTableTests(directory string) (ConversionResult, error) {
	result := ConversionResult{}

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error accessing %s: %v", path, err))
			return nil // Continue processing
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Process Go file
		fileResult, err := processFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error processing %s: %v", path, err))
			return nil // Continue with next file
		}

		result.FilesProcessed++
		if fileResult.Modified {
			result.FilesModified++
			result.TablesConverted += fileResult.TablesConverted
		}

		return nil
	})

	if err != nil {
		return result, fmt.Errorf("error walking directory: %v", err)
	}

	return result, nil
}

// FileResult holds information about the conversion of a single file
type FileResult struct {
	Modified        bool
	TablesConverted int
}

// processFile processes a single Go file and converts its table tests
func processFile(filePath string) (FileResult, error) {
	result := FileResult{}

	// Parse the Go file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return result, fmt.Errorf("error parsing file: %v", err)
	}

	// Find and convert table tests
	modified := false
	tablesConverted := 0

	ast.Inspect(node, func(n ast.Node) bool {
		// Look for variable declarations
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.VAR && decl.Tok != token.CONST {
			return true
		}

		// Process each spec in the declaration
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Check if the declaration is a potential table test
			for i, value := range valueSpec.Values {
				// We're looking for a slice of struct literals
				arrayType, ok := valueSpec.Type.(*ast.ArrayType)
				if !ok {
					continue
				}

				// Check if it's a slice (no length specified)
				if arrayType.Len != nil {
					continue
				}

				// Check if it's a struct type
				structType, ok := arrayType.Elt.(*ast.StructType)
				if !ok {
					continue
				}

				// Find the name field (usually first field)
				nameField, nameFieldIndex := findNameField(structType)
				if nameField == "" {
					continue
				}

				// Convert the slice expression to a map expression
				compLit, ok := value.(*ast.CompositeLit)
				if !ok {
					continue
				}

				// Convert to map-based table test
				mapType := &ast.MapType{
					Key:   &ast.Ident{Name: "string"},
					Value: structType,
				}

				// Create a new struct type without the name field
				newStructType := createStructTypeWithoutField(structType, nameFieldIndex)

				// Create new map composite literal
				newCompLit := &ast.CompositeLit{
					Type: mapType,
					Elts: convertElementsToMapEntries(compLit.Elts, nameFieldIndex),
				}

				// Update the AST
				valueSpec.Type = mapType
				valueSpec.Values[i] = newCompLit
				arrayType.Elt = newStructType

				modified = true
				tablesConverted++
			}
		}

		return true
	})

	// Update loops where the table tests are used
	ast.Inspect(node, func(n ast.Node) bool {
		// Look for range statements
		rangeStmt, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}

		// Check if this is a range over a variable (potential table test)
		ident, ok := rangeStmt.X.(*ast.Ident)
		if !ok {
			return true
		}

		// Find declarations to determine if this is a table test
		obj := lookupObject(node, ident.Name)
		if obj == nil {
			return true
		}

		// Check if it's a map type
		if isMapType(obj) {
			// Update loop variables
			// For map based tests: for name, tc := range tests
			if rangeStmt.Key != nil && rangeStmt.Value != nil {
				// Already has both key and value - no need to change
				return true
			}

			// If only using index (for i := range tests)
			// or if only using value (for _, tc := range tests),
			// update to use both name and value
			if isBlankIdent(rangeStmt.Key) || rangeStmt.Key == nil {
				// Create a new key identifier "name"
				rangeStmt.Key = &ast.Ident{Name: "name"}
			}

			modified = true
		}

		return true
	})

	// Also update t.Run calls to use 'name' instead of 'tc.name'
	ast.Inspect(node, func(n ast.Node) bool {
		// Look for t.Run calls
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if it's a t.Run call
		selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		receiverIdent, ok := selectorExpr.X.(*ast.Ident)
		if !ok || receiverIdent.Name != "t" || selectorExpr.Sel.Name != "Run" {
			return true
		}

		// Check first argument - should be tc.name
		if len(callExpr.Args) < 1 {
			return true
		}

		// Check if the first argument is tc.name
		arg0, ok := callExpr.Args[0].(*ast.SelectorExpr)
		if !ok {
			return true
		}

		x, ok := arg0.X.(*ast.Ident)
		if !ok {
			return true
		}

		// If it's tc.name, replace with just "name"
		if x.Name == "tc" && arg0.Sel.Name == "name" {
			callExpr.Args[0] = &ast.Ident{Name: "name"}
			modified = true
		}

		return true
	})

	if modified {
		// Write the modified AST back to the file
		f, err := os.Create(filePath)
		if err != nil {
			return result, fmt.Errorf("error creating file: %v", err)
		}
		defer f.Close()

		err = printer.Fprint(f, fset, node)
		if err != nil {
			return result, fmt.Errorf("error writing to file: %v", err)
		}

		result.Modified = true
		result.TablesConverted = tablesConverted
	}

	return result, nil
}

// findNameField tries to find the name field in a struct type
func findNameField(structType *ast.StructType) (string, int) {
	if structType.Fields == nil || structType.Fields.List == nil {
		return "", -1
	}

	// Check for name field (usually first field)
	for i, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		if fieldName == "name" || fieldName == "desc" || fieldName == "description" {
			return fieldName, i
		}
	}

	return "", -1
}

// createStructTypeWithoutField creates a new struct type without the specified field
func createStructTypeWithoutField(structType *ast.StructType, fieldIndex int) *ast.StructType {
	if fieldIndex < 0 {
		return structType
	}

	newFields := &ast.FieldList{
		List: make([]*ast.Field, 0, len(structType.Fields.List)-1),
	}

	for i, field := range structType.Fields.List {
		if i != fieldIndex {
			newFields.List = append(newFields.List, field)
		}
	}

	return &ast.StructType{
		Fields: newFields,
	}
}

// convertElementsToMapEntries converts slice elements to map entries
func convertElementsToMapEntries(elements []ast.Expr, nameFieldIndex int) []ast.Expr {
	if nameFieldIndex < 0 {
		return elements
	}

	newElements := make([]ast.Expr, 0, len(elements))

	for _, elt := range elements {
		compLit, ok := elt.(*ast.CompositeLit)
		if !ok || nameFieldIndex >= len(compLit.Elts) {
			continue
		}

		// Extract the name value to be used as key
		var nameValue string
		if basicLit, ok := compLit.Elts[nameFieldIndex].(*ast.BasicLit); ok {
			nameValue = basicLit.Value
		} else {
			// Skip if cannot determine name
			continue
		}

		// Create a new composite literal without the name field
		newElts := make([]ast.Expr, 0, len(compLit.Elts)-1)
		for i, fieldValue := range compLit.Elts {
			if i != nameFieldIndex {
				newElts = append(newElts, fieldValue)
			}
		}

		// Create a new key-value entry
		newElement := &ast.KeyValueExpr{
			Key:   &ast.BasicLit{Kind: token.STRING, Value: nameValue},
			Value: &ast.CompositeLit{Elts: newElts},
		}

		newElements = append(newElements, newElement)
	}

	return newElements
}

// lookupObject finds the declaration of a variable
func lookupObject(file *ast.File, name string) *ast.Object {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR && genDecl.Tok != token.CONST {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, ident := range valueSpec.Names {
				if ident.Name == name {
					return ident.Obj
				}
			}
		}
	}

	return nil
}

// isMapType checks if a variable is a map type
func isMapType(obj *ast.Object) bool {
	if obj == nil || obj.Decl == nil {
		return false
	}

	spec, ok := obj.Decl.(*ast.ValueSpec)
	if !ok {
		return false
	}

	// Check if the type is a map
	_, ok = spec.Type.(*ast.MapType)
	return ok
}

// isBlankIdent checks if an expression is a blank identifier (_)
func isBlankIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "_"
}