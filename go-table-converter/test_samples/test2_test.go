package samples

import "testing"

// TestMultiplication tests the Multiply function
func TestMultiplication(t *testing.T) {
	testCases := map[string]struct {
		a		int
		b		int
		expected	int
	}{
		"simple multiply":	{2, 3, 6},
		"multiply by zero":	{5, 0, 0},
		"negative values":	{-2, 4, -8},
	}

	for name, tc := range testCases {
		result := Multiply(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("%s: expected %d, got %d", name, tc.expected, result)
		}
	}
}

