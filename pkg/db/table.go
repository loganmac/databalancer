package db

// Table is an interface for inserting logs into a table
type Table interface {
	Insert(log []byte) error
}
