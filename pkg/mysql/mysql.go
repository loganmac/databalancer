package mysql

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kolide/databalancer-logan/pkg/logs"
	"github.com/pkg/errors"
)

// Table defines methods for inserting and querying logs for that table
type Table struct {
	*sql.DB // selected db for table
	family  logs.Family
	schema  logs.Schema
}

// Client is a connection to a MySQL database
type Client struct {
	*sql.DB                        // underlying database
	tables  map[logs.Family]*Table // a kind of local cache for tables
}

// Create makes a new MySQL database client and ensures that it's connected
func Create(username, password, address, name string) (*Client, error) {
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

// FindOrCreateTable finds or creates the table based on the given
// attributes with the client, caching it in the process
func (c *Client) FindOrCreateTable(family logs.Family, schema logs.Schema) (logs.Table, error) {
	if table, ok := c.tables[family]; ok == true {
		return table, nil
	}

	create, err := c.Prepare(
		"CREATE TABLE IF NOT EXISTS `raw_logs` (" +
			"`id` INT NOT NULL AUTO_INCREMENT," +
			"`family` TEXT," +
			"`log` TEXT," +
			"PRIMARY KEY(`id`)" +
			");",
	)
	if err != nil {
		return nil, errors.Wrap(err, "preparing create statement")
	}

	_, err = create.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "executing create statement")
	}

	// construct the table
	table := &Table{DB: c.DB, family: family, schema: schema}

	// cache the table
	c.tables[family] = table

	return table, nil
}

// Insert creates a new record in the supplied table
func (t *Table) Insert(log []byte) error {
	stmt, err := t.Prepare("INSERT INTO `raw_logs` (`family`, `log`) VALUES(?, ?);")
	if err != nil {
		return errors.Wrap(err, "creating insert statement")
	}

	res, err := stmt.Exec(string(t.family), log)
	if err != nil {
		return errors.Wrap(err, "executing insert statement")
	}

	// last id is not currently used
	if _, err := res.LastInsertId(); err != nil {
		return errors.Wrap(err, "inserting record")
	}

	return nil
}
