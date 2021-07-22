package utils

var void struct{}

type Set struct {
	Map map[string]struct{}
}

func (s *Set) Add(element string) {
	s.Map[element] = void
}

func (s *Set) Contains(element string) bool {
	_, isPresent := s.Map[element]
	return isPresent 
}

func EmptySet() *Set {
	return &Set{make(map[string]struct{})}
}

func SetFromArray(arr []string) *Set {
	set := EmptySet()
	for _, element := range arr {
		set.Add(element)
	}

	return set
}
