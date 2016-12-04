package encoder

import "fmt"

type ReflectionField struct {
	Name string	`json:"name"`
	Type string	`json:"type"`
	Tag  string	`json:"tag"`
}

func (s *ReflectionField) String() string {
	return fmt.Sprintln(s.Name, s.Type, s.Tag)
}
