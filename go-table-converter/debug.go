package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

func main() {
	directoryPath := "test_samples"
	filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		// Skip directories and non-Go files
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		
		// Skip the debug file itself
		if filepath.Base(path) == "debug.go" {
			return nil
		}
		
		analyzeFile(path)
		return nil
	})
}

func analyzeFile(filePath string) {
	fmt.Printf("\n===== Analyzing %s =====\n", filePath)
	
	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		return
	}
	
	// Find table tests
	foundSliceOfStructs := false
	
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok != token.VAR && x.Tok != token.CONST {
				return true
			}
			
			fmt.Printf("Found %s declaration\n", x.Tok)
			
			// Look at each spec
			for i, spec := range x.Specs {
				fmt.Printf("Checking spec %d (%T)\n", i, spec)
				
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					fmt.Printf("  - Not a ValueSpec\n")
					continue
				}
				
				// Print each variable name
				for j, name := range vs.Names {
					fmt.Printf("  Variable: %s\n", name.Name)
					
					// Print type info
					if vs.Type != nil {
						fmt.Printf("    Type: %T\n", vs.Type)
						
						// If it's a composite type, check more
						switch t := vs.Type.(type) {
						case *ast.ArrayType:
							fmt.Printf("    Array/Slice type with element type: %T\n", t.Elt)
							
							if t.Len == nil {
								fmt.Printf("    It's a slice (no length)\n")
							} else {
								fmt.Printf("    It's an array with length\n")
							}
							
							// Check if element is a struct
							if st, ok := t.Elt.(*ast.StructType); ok {
								fmt.Printf("    Element is a struct\n")
								foundSliceOfStructs = true
								
								// Check fields
								if st.Fields != nil && st.Fields.List != nil {
									fmt.Printf("    Struct has %d fields:\n", len(st.Fields.List))
									
									for k, field := range st.Fields.List {
										if len(field.Names) > 0 {
											fmt.Printf("      Field %d: %s\n", k, field.Names[0].Name)
										} else {
											fmt.Printf("      Field %d: (anonymous)\n", k)
										}
									}
								}
							}
						case *ast.StructType:
							fmt.Printf("    Direct struct type\n")
						case *ast.MapType:
							fmt.Printf("    Map type with key %T and value %T\n", t.Key, t.Value)
						}
					}
					
					// Check the value
					if j < len(vs.Values) {
						fmt.Printf("    Value type: %T\n", vs.Values[j])
						
						// For composite literals, show more info
						if cl, ok := vs.Values[j].(*ast.CompositeLit); ok {
							fmt.Printf("    Composite literal with %d elements\n", len(cl.Elts))
							
							// Print the first few elements
							for k := 0; k < len(cl.Elts) && k < 3; k++ {
								fmt.Printf("      Element %d: %T\n", k, cl.Elts[k])
							}
						}
					}
				}
			}
		}
		
		return true
	})
	
	// Check range statements
	fmt.Println("\nLoop statements:")
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.RangeStmt:
			fmt.Println("- Found range statement")
			
			// What is being ranged over
			fmt.Printf("  Ranging over: %T\n", x.X)
			if ident, ok := x.X.(*ast.Ident); ok {
				fmt.Printf("  Ranging over variable: %s\n", ident.Name)
			}
			
			// Key variable
			if x.Key != nil {
				fmt.Printf("  Key variable: %T\n", x.Key)
				if ident, ok := x.Key.(*ast.Ident); ok {
					fmt.Printf("  Key variable name: %s\n", ident.Name)
				}
			}
			
			// Value variable
			if x.Value != nil {
				fmt.Printf("  Value variable: %T\n", x.Value)
				if ident, ok := x.Value.(*ast.Ident); ok {
					fmt.Printf("  Value variable name: %s\n", ident.Name)
				}
			}
		}
		
		return true
	})
	
	// Check t.Run calls
	fmt.Println("\nt.Run calls:")
	foundTestRun := false
	ast.Inspect(node, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		
		selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		
		receiverIdent, ok := selectorExpr.X.(*ast.Ident)
		if !ok || receiverIdent.Name != "t" || selectorExpr.Sel.Name != "Run" {
			return true
		}
		
		foundTestRun = true
		fmt.Println("- Found t.Run call")
		
		if len(callExpr.Args) > 0 {
			fmt.Printf("  First argument: %T\n", callExpr.Args[0])
			
			switch a := callExpr.Args[0].(type) {
			case *ast.SelectorExpr:
				x, ok := a.X.(*ast.Ident)
				if ok {
					fmt.Printf("  Selector expression: %s.%s\n", x.Name, a.Sel.Name)
				}
			case *ast.Ident:
				fmt.Printf("  Identifier: %s\n", a.Name)
			case *ast.BasicLit:
				fmt.Printf("  Basic literal: %s\n", a.Value)
			}
		}
		
		return true
	})
	
	if !foundTestRun {
		fmt.Println("No t.Run calls found")
	}
	
	if !foundSliceOfStructs {
		fmt.Println("\nNo slice of structs found in this file")
	} else {
		fmt.Println("\nFound slice of structs in this file")
	}
}