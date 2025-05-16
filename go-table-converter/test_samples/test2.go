package samples

import "testing"

func TestMultiplication(t *testing.T) {
	testCases := []struct {
		desc     string
		a        int
		b        int
		expected int
	}{
		{"simple multiply", 2, 3, 6},
		{"multiply by zero", 5, 0, 0},
		{"negative values", -2, 4, -8},
	}

	for _, tc := range testCases {
		result := Multiply(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("%s: expected %d, got %d", tc.desc, tc.expected, result)
		}
	}
}

func Multiply(a, b int) int {
	return a * b
}