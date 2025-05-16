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

		fmt.Printf("Processing file: %s\n", path)

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
			fmt.Printf("Modified file: %s, Tables converted: %d\n", path, fileResult.TablesConverted)
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
	
	// First, identify all table test variables
	tableTestVars := make(map[string]bool)
	
	// Step 1: Find all slice of struct declarations (table tests)
	ast.Inspect(node, func(n ast.Node) bool {
		// Look for assignment statements like 'tests := []struct{ ... }{ ... }'
		if assign, ok := n.(*ast.AssignStmt); ok {
			// Only interested in := or = assignments
			if assign.Tok != token.DEFINE && assign.Tok != token.ASSIGN {
				return true
			}
			
			// Check each right-hand side value
			for i, rhs := range assign.Rhs {
				// Check if it's a composite literal
				compLit, ok := rhs.(*ast.CompositeLit)
				if !ok {
					continue
				}
				
				// Check if it's a slice of structs
				arrayType, ok := compLit.Type.(*ast.ArrayType)
				if !ok {
					continue
				}
				
				// Check if it's a slice (no length)
				if arrayType.Len != nil {
					continue
				}
				
				// Check if element type is a struct
				structType, ok := arrayType.Elt.(*ast.StructType)
				if !ok {
					continue
				}
				
				// Check for name/description field
				nameField, nameFieldIndex := findNameField(structType)
				if nameField == "" {
					continue
				}
				
				// Found a table test - get the variable name
				if i < len(assign.Lhs) {
					if ident, ok := assign.Lhs[i].(*ast.Ident); ok {
						tableTestVars[ident.Name] = true
						fmt.Printf("Found table test variable: %s\n", ident.Name)
						
						// Convert the slice of structs to a map
						mapType := &ast.MapType{
							Key:   &ast.Ident{Name: "string"},
							Value: createStructTypeWithoutField(structType, nameFieldIndex),
						}
						
						// Create new map entries from the slice elements
						entries := make([]ast.Expr, 0, len(compLit.Elts))
						for _, elt := range compLit.Elts {
							if sliceElt, ok := elt.(*ast.CompositeLit); ok && nameFieldIndex < len(sliceElt.Elts) {
								// Extract name field value for map key
								var nameValue ast.Expr
								if basicLit, ok := sliceElt.Elts[nameFieldIndex].(*ast.BasicLit); ok {
									nameValue = basicLit
								} else {
									continue
								}
								
								// Create a new struct literal without the name field
								newElts := make([]ast.Expr, 0, len(sliceElt.Elts)-1)
								for j, val := range sliceElt.Elts {
									if j != nameFieldIndex {
										newElts = append(newElts, val)
									}
								}
								
								// Create map entry
								entry := &ast.KeyValueExpr{
									Key:   nameValue,
									Value: &ast.CompositeLit{Elts: newElts},
								}
								
								entries = append(entries, entry)
							}
						}
						
						// Replace the original slice with the new map
						compLit.Type = mapType
						compLit.Elts = entries
						
						modified = true
						tablesConverted++
					}
				}
			}
		}
		
		return true
	})
	
	// Step 2: Update range loops over table tests
	ast.Inspect(node, func(n ast.Node) bool {
		if rangeStmt, ok := n.(*ast.RangeStmt); ok {
			// Check if the range is over a table test variable
			if ident, ok := rangeStmt.X.(*ast.Ident); ok && tableTestVars[ident.Name] {
				fmt.Printf("Found range over table test: %s\n", ident.Name)
				
				// Update loop variables for map-based iteration
				// Change from: for _, tc := range tests
				// To:         for name, tc := range tests
				if isBlankIdent(rangeStmt.Key) || rangeStmt.Key == nil {
					rangeStmt.Key = &ast.Ident{Name: "name"}
					modified = true
				}
			}
		}
		
		return true
	})
	
	// Step 3: Update t.Run calls and other references to use the map key instead of tc.name/tc.desc
	ast.Inspect(node, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			// Check if it's a t.Run call
			if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok && ident.Name == "t" && selectorExpr.Sel.Name == "Run" {
					// Check if the first argument is tc.name
					if len(callExpr.Args) > 0 {
						if arg, ok := callExpr.Args[0].(*ast.SelectorExpr); ok {
							if x, ok := arg.X.(*ast.Ident); ok && x.Name == "tc" && 
							   (arg.Sel.Name == "name" || arg.Sel.Name == "desc" || arg.Sel.Name == "description") {
								// Replace tc.name with name
								callExpr.Args[0] = &ast.Ident{Name: "name"}
								modified = true
							}
						}
					}
				}
			}
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
		fmt.Println("Struct has no fields")
		return "", -1
	}
	
	// Check for name field
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

// isBlankIdent checks if an expression is a blank identifier (_)
func isBlankIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "_"
}