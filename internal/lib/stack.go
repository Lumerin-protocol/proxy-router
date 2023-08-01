package lib

type Stack[T any] []T

func (s *Stack[T]) IsEmpty() bool {
	return len(*s) == 0
}

func (s *Stack[T]) Push(str T) {
	*s = append(*s, str)
}

func (s *Stack[T]) Pop() (T, bool) {
	if s.IsEmpty() {
		return *new(T), false
	} else {
		index := len(*s) - 1
		element := (*s)[index]
		*s = (*s)[:index]
		return element, true
	}
}

func (s *Stack[T]) Peek() (T, bool) {
	if s.IsEmpty() {
		return *new(T), false
	} else {
		index := len(*s) - 1
		element := (*s)[index]
		return element, true
	}
}

func (s *Stack[T]) Size() int {
	return len(*s)
}

func (s *Stack[T]) Clear() {
	*s = *new(Stack[T])
}