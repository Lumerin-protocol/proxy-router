package lib

type Set map[string]struct{}

func NewSet() Set {
	s := make(map[string]struct{})
	return Set(s)
}

func NewSetFromSlice(slice []string) Set {
	s := make(map[string]struct{})
	for _, v := range slice {
		s[v] = struct{}{}
	}
	return Set(s)
}

func (s Set) Add(value ...string) {
	for _, v := range value {
		s[v] = struct{}{}
	}
}

func (s Set) Remove(value string) bool {
	_, c := s[value]
	delete(s, value)
	return c
}

func (s Set) Contains(value string) bool {
	_, c := s[value]
	return c
}

func (s Set) Len() int {
	return len(s)
}

func (s Set) ToSlice() []string {
	var keys []string
	for k := range s {
		keys = append(keys, k)
	}
	return keys
}

func (s Set) Clear() {
	for k := range s {
		delete(s, k)
	}
}
