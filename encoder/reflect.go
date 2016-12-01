package encoder

import "fmt"

type ReflectionField struct {
	Name string
	Type string
	Tag  string
}

func (s *ReflectionField) String() string {
	return fmt.Sprintln(s.Name, s.Type, s.Tag)
}
