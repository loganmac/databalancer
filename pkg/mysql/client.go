package mysql

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql" //mysql driver
	"github.com/kolide/databalancer-logan/pkg/logs"
	"github.com/pkg/errors"
)

// Client is a connection to a MySQL database
type Client struct {
	*sql.DB // underlying database
}

// Table defines methods for inserting and querying logs for that table
type Table struct {
	*sql.DB                   // database for table
	name    string            // table name
	schema  map[string]string // schema of the table from request
}

// CreateClient makes a new MySQL database client and ensures that it's connected
func CreateClient(username, password, address, name string) (*Client, error) {
	connectionString := fmt.Sprintf(
		"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		username,
		password,
		address,
		name,
	)
	// Using our connection string, we attempt to open a MySQL connection
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, errors.Wrap(err, "opening database")
	}

	// Now, we ensure that can communicate with the database
	if err = db.Ping(); err != nil {
		return nil, errors.Wrap(err, "pinging database")
	}

	log.Printf("Connected to MySQL as %s at %s\n", username, address)
	return &Client{DB: db}, nil
}

// CreateTable creates the the table (if it doesn't exist) based on the given
// attributes with the client and creates an Insert method.
func (c *Client) CreateTable(name logs.Family, schema logs.Schema) (logs.Table, error) {
	// construct create table statement
	create := CreateTableStatement(name.String(), schema)

	// create the table
	_, err := c.Exec(create)
	if err != nil {
		return nil, errors.Wrapf(err, "creating %s table", name)
	}

	return &Table{DB: c.DB, name: name.String(), schema: schema}, nil
}

// Insert creates new logs in the supplied table
func (t *Table) Insert(logs logs.JSON) error {
	// construct insert statement
	insert, args := InsertTableStatement(t.name, t.schema, logs)

	// insert the data
	_, err := t.Exec(insert, args...)
	if err != nil {
		return errors.Wrapf(err, "inserting records for %s table", t.name)
	}
	return nil
}
