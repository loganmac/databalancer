package mysql

import (
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql" //mysql driver
	"github.com/jmoiron/sqlx"
	"github.com/kolide/databalancer-logan/pkg/logs"
	"github.com/pkg/errors"
)

// Client is a connection to a MySQL database
type Client struct {
	*sqlx.DB // underlying database
}

// Table defines methods for inserting and querying logs for that table
type Table struct {
	*sqlx.DB                   // database for table
	Name     string            // table name
	Schema   map[string]string // schema of the table from request
}

// NewClient makes a new MySQL database client and ensures that it's connected
func NewClient(username, password, address, name string) (*Client, error) {
	connectionString := fmt.Sprintf(
		"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		username,
		password,
		address,
		name,
	)
	// Using our connection string, we attempt to open a MySQL connection
	db, err := sqlx.Open("mysql", connectionString)
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

	return &Table{DB: c.DB, Name: name.String(), Schema: schema}, nil
}

// Insert creates new logs in the supplied table
func (t *Table) Insert(logs logs.JSON) error {
	// construct insert statement
	insert, args := InsertTableStatement(t.Name, t.Schema, logs)

	log.Printf("Insert statment: \n%+v\n", insert)
	log.Printf("Args: \n%+v\n", args)

	// insert the data
	_, err := t.Exec(insert, args...)
	if err != nil {
		return errors.Wrapf(err, "inserting records for %s table", t.Name)
	}

	return nil
}

// QueryJSON returns rows as a representation that can be marshalled to JSON
func (c *Client) QueryJSON(query string) (logs.JSON, error) {
	_, err := c.DescribeDatabase()
	if err != nil {
		return nil, errors.Wrap(err, "crap")
	}
	// make the query. we use a prepared statement here because mysql
	// only returns column type info if the statement is prepared,
	// otherwise everything will be typed as []byte
	stmt, err := c.Preparex(query)
	if err != nil {
		return nil, errors.Wrapf(err, "querying database with query '%s'", query)
	}
	defer stmt.Close()

	// execute the query
	rows, err := stmt.Queryx()
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving rows of query '%s'", query)
	}
	defer rows.Close()

	// scan the rows into a JSON representation
	var results []map[string]interface{}
	for rows.Next() {
		// create a row
		row := make(map[string]interface{})
		// scan the row
		if err := rows.MapScan(row); err != nil {
			return nil, errors.Wrapf(err, "scanning row of query '%s'", query)
		}
		// the mysql driver returns text fields as []byte,
		// so cast to string if any fields have that type
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				row[k] = string(b)
			}
		}
		results = append(results, row)
	}
	return results, nil
}

// DescribeDatabase returns the table names, columns, and types
func (c *Client) DescribeDatabase() (logs.JSON, error) {
	var tableDescriptions []struct {
		Schema   string // not used yet
		Name     string // table name
		Column   string // column name
		Nullable string // YES/NO if column nullable
		Datatype string // column data type
	}
	err := c.Select(&tableDescriptions,
		"SELECT `TABLE_SCHEMA` as `schema`, "+
			"`TABLE_NAME` as `name`, "+
			"`COLUMN_NAME` as `column`, "+
			"`IS_NULLABLE` as `nullable`, "+
			"`DATA_TYPE` as `datatype` "+
			"FROM information_schema.columns "+
			"WHERE table_schema <> 'information_schema'")
	if err != nil {
		return nil, errors.Wrap(err, "describing databse")
	}

	var tables logs.JSON
	for i, tableDescription := range tableDescriptions {
		_ = i
		_ = tableDescription
		_ = tables
	}

	log.Printf("%+v\n", tableDescriptions)

	return nil, nil
}
