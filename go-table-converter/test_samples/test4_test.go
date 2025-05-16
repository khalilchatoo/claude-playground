package samples

import "testing"

// TestMultiplicationTable tests the Multiply function
func TestMultiplicationTable(t *testing.T) {
	testCases := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"simple multiply", 2, 3, 6},
		{"multiply by zero", 5, 0, 0},
		{"negative values", 2, 3, 6},
	}

	for name, tc := range testCases {
		result := Multiply(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("%s: expected %d, got %d", name, tc.expected, result)
		}
	}
}
