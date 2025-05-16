package samples

// We need an implementation file so the tests can run

// Add is implemented in test1.go but we need it here too
func Add(a, b int) int {
	return a + b
}

// Multiply is implemented in test2.go but we need it here too
func Multiply(a, b int) int {
	return a * b
}