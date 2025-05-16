# Go Table Test Converter

This tool converts slice-based table tests in Go to map-based table tests. It analyzes Go source code, identifies slice-based table tests, and transforms them to use maps with the test name as the key.

## Project Structure

- `tabletests.go`: Main Go tool that handles the conversion logic
- `run_conversion.sh`: Shell script to easily build and run the converter
- `test_samples/`: Directory with example test files to demonstrate the tool
  - `test1_test.go`: Table test with t.Run subtests (converted)
  - `test2_test.go`: Table test with direct iteration (converted)
  - `test3_test.go`: Regular test (not a table test)
  - `main.go`: Implementation file for the test functions

## How Conversion Works

The tool:
1. Recursively walks through a directory and identifies Go files
2. Parses each file to create an Abstract Syntax Tree (AST)
3. Identifies slice-based table tests by looking for:
   - Assignment statements with slice of anonymous structs
   - Structs that have a "name", "desc", or "description" field
4. Converts these to map-based table tests:
   - Changes slice type to map[string]struct
   - Moves the name/description field to be the map key
   - Updates loop variables to use the map key for test names
   - Replaces t.Run(tc.name, ...) with t.Run(name, ...)
   - Replaces other references to the removed name field (like in error messages)
5. Only modifies files that actually contain slice-based table tests

## Example Conversion

### Before:
```go
tests := []struct {
    name     string
    input    string
    expected string
}{
    {"simple case", "input1", "expected1"},
    {"edge case", "input2", "expected2"},
}

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // Test logic
    })
}
```

### After:
```go
tests := map[string]struct {
    input    string
    expected string
}{
    "simple case": {input: "input1", expected: "expected1"},
    "edge case":   {input: "input2", expected: "expected2"},
}

for name, tc := range tests {
    t.Run(name, func(t *testing.T) {
        // Test logic
    })
}
```

## Usage

1. To run the converter:
   ```
   go run tabletests.go <directory_path>
   ```

2. Using the shell script:
   ```
   ./run_conversion.sh <directory_path>
   ```

## Benefits of Map-Based Table Tests

- Test names are more clearly decoupled from test data
- Random test execution helps catch unintended dependencies
- Map keys make the test name's purpose more explicit
- Better IDE collapsibility for complex test cases
- References to test names are consistent throughout the code

## Technical Details

The tool uses Go's standard library packages:
- `go/parser`: For parsing Go source code
- `go/ast`: For manipulating the Abstract Syntax Tree
- `go/printer`: For writing modified AST back to file
- `go/token`: For token handling and position information

## Implementation Notes

The converter handles several edge cases:
- It identifies table test variables by looking for slice declarations with struct elements
- It updates both t.Run calls and other references to the removed name field
- It preserves code formatting and comments
- It detects different naming patterns for the test name field (name, desc, description)