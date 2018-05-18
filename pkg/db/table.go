package db

// Table is an interface for inserting logs into a table
// Note: This package with a single interface may seem strange.
// It is because of a limitation of the golang reflection laws,
// which does not allow methods of an interface to return a type
// that implements that interface, they must return that interface directly.
// This interface allows that to happen without circular dependencies.
type Table interface {
	Insert(log []byte) error
}
