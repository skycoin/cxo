package encoder

import "fmt"

type ReflectionField struct {
	Name []byte
	Type []byte
	Tag  []byte
}

func (s *ReflectionField) String() string {
	return fmt.Sprintln(s.Name, s.Type, s.Tag)
}
