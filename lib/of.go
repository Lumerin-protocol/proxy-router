package lib

// Of converts any value into pointer. Useful with constants
func Of[E any](e E) *E {
	return &e
}
