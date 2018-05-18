package logs

import (
	"encoding/json"
	"log"

	"github.com/kolide/databalancer-logan/pkg/db"
	"github.com/pkg/errors"
)

// DBClient is the interface that defines methods for creating tables in a database
type DBClient interface {
	FindOrCreateTable(family Family, schema Schema) (db.Table, error)
}

// Service contains the databases to ingest logs into
type Service struct {
	db DBClient
}

// Family is the table name for a group of logs
type Family string

// Schema describes the structure of a log type
type Schema map[string]string

// Raw is the raw JSON of a log
type Raw map[string]interface{}

// CreateService returns a `Service`, backed by a `DB`
func CreateService(db DBClient) *Service {
	return &Service{db: db}
}

// Ingest is a method which ingests logs into the database.
// It finds or creates the database table,  parses logs,
// and writes the logs to the selected table.
func (s *Service) Ingest(family Family, schema Schema, logs []Raw) error {
	table, err := s.db.FindOrCreateTable(family, schema)
	if err != nil {
		return errors.Wrapf(err, "finding or creating table '%s'", family)
	}

	log.Printf("Received logs for the %s log family\n", family)
	for column, columnType := range schema {
		log.Printf("Log values for the field %s of the %s log will be of type %s\n", column, family, columnType)
	}

	for _, logEvent := range logs {
		log.Printf("Handling a new log event for the %s log family\n", family)
		for field, value := range logEvent {
			columnType, ok := schema[field]
			if !ok {
				return errors.Errorf(
					"Data type for the field %s was not specified in the %s schema map\n",
					field,
					family,
				)
			}
			switch columnType {
			case "string":
				log.Printf("The value of the %s field in the %s log event is %s\n", field, family, value.(string))
			case "int":
				log.Printf("The value of the %s field in the %s log event is %d\n", field, family, int(value.(float64)))
			default:
				return errors.Errorf("Unsupported data type in %s log for the field %s: %s\n", family, field, columnType)
			}
		}

		// Marshal the log event back into JSON to store it in the database
		logJSON, err := json.Marshal(logEvent)
		if err != nil {
			return err
		}

		if err := table.Insert(logJSON); err != nil {
			return err
		}
	}

	return nil
}

// String method for Family in case underlying type changes
func (f Family) String() string {
	return string(f)
}
