package samples

import "testing"

func TestAddition(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"simple sum", 1, 2, 3},
		{"zero value", 0, 5, 5},
		{"negative numbers", -2, -3, -5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Add(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func Add(a, b int) int {
	return a + b
}