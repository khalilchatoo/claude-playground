package samples

import "testing"

// TestAddition tests the Add function
func TestAddition(t *testing.T) {
	tests := map[string]struct {
		a		int
		b		int
		expected	int
	}{
		"simple sum":		{1, 2, 3},
		"zero value":		{0, 5, 5},
		"negative numbers":	{-2, -3, -5},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := Add(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

