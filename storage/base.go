package storage

// Storage is an interface that provides storing and retrieval of file like things
type Storage interface {
	// Name is the name of the storage implementation
	Name() string

	// Test verifies this storage is functioning and returns an error if not
	Test() error

	// Get retrieves the file from the given path
	Get(path string) (string, []byte, error)

	// Put stores the given file at the given path
	Put(path string, contentType string, contents []byte) (string, error)
}
