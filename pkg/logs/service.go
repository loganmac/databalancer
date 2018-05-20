package logs

import (
	"log"

	"github.com/pkg/errors"
	"github.com/xwb1989/sqlparser"
)

// DBClient is the interface that defines methods for creating tables in a database
type DBClient interface {
	CreateTable(family Family, schema Schema) (Table, error)
	QueryJSON(query string) (JSON, error)
	DescribeDatabase() (JSON, error)
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

// JSON represents data that can be marshalled to JSON
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

	if err := table.Insert(logs); err != nil {
		// TODO: check and convert errors
		return err
	}
	return nil
}

// Query receives a SQL query that it sends to the database
// as long as it is a SELECT
func (s *Service) Query(query string) (JSON, error) {
	// parse the query, also verifies that it's a valid
	// single statement query
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing query '%s'", query)
	}
	switch stmt.(type) {
	case *sqlparser.Select:
		// statement is good, and a select, so pass it through
		results, err := s.db.QueryJSON(query)
		if err != nil {
			return nil, errors.Wrap(err, "querying database client")
		}
		return results, nil
	default:
		// query wasn't really a query, so return readonly error
		return nil, ErrReadOnly
	}
}

// DescribeLogs describes the database tables and columns as JSON
func (s *Service) DescribeLogs() (JSON, error) {
	// TODO: right now this just returns the same format as the database,
	// but it would be better if this service defined a structure that
	// the databases should use describe their data, in the same
	// language that the ingestion uses for schema and family etc
	results, err := s.db.DescribeDatabase()
	if err != nil {
		return nil, errors.Wrap(err, "describing logs")
	}
	return results, nil
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
