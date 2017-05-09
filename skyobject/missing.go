package skyobject

type MissingObjectError struct {
	ref Reference
}

func (m *MissingObjectError) Error() string {
	return "missing object " + m.ref.String()
}

func (m *MissingObjectError) Reference() Reference {
	return m.ref
}
