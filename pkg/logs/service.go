package logs

import (
	"log"

	"github.com/pkg/errors"
	"github.com/xwb1989/sqlparser"
)

// DBClient is the interface that defines methods for creating tables in a database
type DBClient interface {
	CreateTable(family Family, schema Schema) (Table, error)
	Query(query string) (JSON, error)
}

// Table is an interface for inserting records into a table
type Table interface {
	Insert(records JSON) error
}

// Service contains the databases to ingest logs into
type Service struct {
	db DBClient
}

// Family is the table name for a group of logs
type Family string

// Schema describes the structure of a log type
type Schema map[string]string

// JSON is the JSON formatted logs
type JSON []map[string]interface{}

// ErrReadOnly is returned when valid SQL other than a SELECT is sent
var ErrReadOnly = errors.New("service can only be used to query records")

// CreateService returns a `Service`, backed by a `DB`
func CreateService(db DBClient) *Service {
	return &Service{db: db}
}

// Ingest parses and stores logs into the database.
// It validates the logs match the schema, creates the database table,
// and then writes the logs to it.
func (s *Service) Ingest(family Family, schema Schema, logs JSON) error {
	// validate that the logs match the given schema and contain valid types
	if err := checkLogSchema(schema, logs); err != nil {
		// TODO: check for specific error types, wrap in error type that
		// any exposing interface can use to create nicer error messaging
		return errors.Wrapf(err, "validating %s logs against schema", family)
	}

	table, err := s.db.CreateTable(family, schema)
	if err != nil {
		// TODO: check and convert errors
		return errors.Wrapf(err, "creating table %s", family)
	}

	// return if there are no logs to insert
	// NOTE: I'm not sure this is desirable from a product perspective,
	// but this allows you to make requests with just the schema to create tables
	if len(logs) == 0 {
		return nil
	}

	if err := table.Insert(logs); err != nil {
		// TODO: check and convert errors
		return err
	}
	return nil
}

// Query receives a SQL query that it
func (s *Service) Query(query string) (JSON, error) {

	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing query '%s'", query)
	}
	switch stmt.(type) {
	case *sqlparser.Select:
		// statement is good, and a select, so pass it through
		results, err := s.db.Query(query)
		if err != nil {
			return nil, errors.Wrap(err, "querying database client")
		}
		return results, nil
	default:
		return nil, ErrReadOnly
	}
}

// checkLogSchema validates that all logs match the given schema
func checkLogSchema(schema Schema, logs JSON) error {
	for _, logEvent := range logs {
		for field, value := range logEvent {
			columnType, ok := schema[field]
			if !ok {
				return errors.Errorf("field %s was not specified in the schema", field)
			}
			switch columnType {
			case "string":
				log.Printf("The value of the %s field is %s\n", field, value.(string))
			case "int":
				log.Printf("The value of the %s field is %d\n", field, int(value.(float64)))
			default:
				// TODO: convert to error that can be used to convery more information to
				// any exposing interfaces (http, grpc, etc)
				return errors.Errorf("Unsupported data type in log for the field %s: %s\n", field, columnType)
			}
		}
	}
	return nil
}

// String method for Family in case underlying type changes
func (f Family) String() string {
	return string(f)
}
