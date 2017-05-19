package skyobject

// A MissingObjectError represents ...
type MissingObjectError struct {
	ref Reference
}

// Error implements error interface
func (m *MissingObjectError) Error() string {
	return "missing object " + m.ref.String()
}

// Reference returns reference of missing object
func (m *MissingObjectError) Reference() Reference {
	return m.ref
}

// A MissingRegistryError represents ...
type MissingRegistryError struct {
	rr RegistryReference
}

// Error implements error interface
func (m *MissingRegistryError) Error() string {
	return "missing registry " + m.rr.String()
}

// Reference returns reference of missing registry
func (m *MissingRegistryError) Reference() RegistryReference {
	return m.rr
}
