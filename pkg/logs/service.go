package logs

import (
	"log"

	"github.com/pkg/errors"
)

// DBClient is the interface that defines methods for creating tables in a database
type DBClient interface {
	CreateTable(family Family, schema Schema) (Table, error)
}

// Table is an interface for inserting records into a table
type Table interface {
	Insert(records Raw) error
}

// Service contains the databases to ingest logs into
type Service struct {
	db DBClient
}

// Family is the table name for a group of logs
type Family string

// Schema describes the structure of a log type
type Schema map[string]string

// Raw is the raw JSON of logs
type Raw []map[string]interface{}

// CreateService returns a `Service`, backed by a `DB`
func CreateService(db DBClient) *Service {
	return &Service{db: db}
}

// Ingest is a method which parses and stores logs into the database.
// It validates the logs match the schema, creates the database table,
// and then writes the logs to it.
func (s *Service) Ingest(family Family, schema Schema, logs Raw) error {
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
	// but this allows you to make requests with just the schema to create
	// tables
	if len(logs) == 0 {
		return nil
	}

	if err := table.Insert(logs); err != nil {
		// TODO: check and convert errors
		return err
	}
	return nil
}

// checkLogSchema validates that all logs match the given schema
func checkLogSchema(schema Schema, logs Raw) error {
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
