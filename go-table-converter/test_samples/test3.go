package samples

import "testing"

// This file doesn't use table tests
func TestDivision(t *testing.T) {
	if Divide(6, 2) != 3 {
		t.Error("6 / 2 should equal 3")
	}

	if Divide(5, 0) != 0 {
		t.Error("Division by zero should return 0")
	}
}

func Divide(a, b int) int {
	if b == 0 {
		return 0
	}
	return a / b
}