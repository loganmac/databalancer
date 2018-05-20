package logs_test

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/kolide/databalancer-logan/pkg/logs"
	"github.com/stretchr/testify/assert"
)

// MOCKS
type mockDB struct{}
type mockTable struct{}

func (m *mockDB) CreateTable(family logs.Family, schema logs.Schema) (logs.Table, error) {
	return &mockTable{}, nil
}

func (m *mockDB) QueryJSON(query string) (logs.JSON, error) {
	return logs.JSON{}, nil
}

func (m *mockDB) DescribeDatabase() (logs.JSON, error) {
	panic("not implemented")
}

func (m *mockTable) Insert(records logs.JSON) error {
	return nil
}

// describes a test case for Ingest
type ingestCase struct {
	name   string
	family logs.Family
	schema logs.Schema
	logs   logs.JSON
	result error
}

// utility type to clean up tests
type rawLog map[string]interface{}

func TestIngest(t *testing.T) {
	// disable logging
	log.SetOutput(ioutil.Discard)

	// GIVEN
	service := logs.CreateService(&mockDB{})

	// THEN
	successCases := []ingestCase{
		{
			name:   "a correct schema should insert without problems",
			family: "dog_registry",
			schema: logs.Schema{"name": "string", "breed": "string", "weight": "int"},
			logs: logs.JSON{
				rawLog{"name": "max", "breed": "chihuahua", "weight": float64(3)},
				rawLog{"name": "spot", "breed": "husky", "weight": float64(130)},
				rawLog{"name": "spike", "breed": "bulldog", "weight": float64(80)},
			},
		},
		{
			name:   "a schema with more fields than the logs should insert without problems",
			family: "dog_registry",
			schema: logs.Schema{"name": "string", "breed": "string", "weight": "int", "age": "int"},
			logs: logs.JSON{
				rawLog{"name": "max", "breed": "chihuahua", "weight": float64(3)},
				rawLog{"name": "spot", "breed": "husky", "weight": float64(130)},
				rawLog{"name": "spike", "breed": "bulldog", "weight": float64(80)},
			},
		},
	}

	for _, tt := range successCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, service.Ingest(tt.family, tt.schema, tt.logs))
		})
	}

	failureCases := []ingestCase{
		{
			name:   "a schema with an unknown type should return an error",
			family: "dog_registry",
			schema: logs.Schema{"name": "float", "breed": "string", "weight": "int"},
			logs: logs.JSON{
				rawLog{"name": "max", "breed": "chihuahua", "weight": float64(3)},
				rawLog{"name": "spot", "breed": "husky", "weight": float64(130)},
				rawLog{"name": "spike", "breed": "bulldog", "weight": float64(80)},
			},
		},
		{
			name:   "a schema that doesn't match the logs should return an error",
			family: "dog_registry",
			schema: logs.Schema{"name": "string", "breed": "string", "age": "int"},
			logs: logs.JSON{
				rawLog{"name": "max", "breed": "chihuahua", "weight": float64(3)},
				rawLog{"name": "spot", "breed": "husky", "weight": float64(130)},
				rawLog{"name": "spike", "breed": "bulldog", "weight": float64(80)},
			},
		},
		{
			name:   "heterogenous logs that contain more fields than the schema return an error",
			family: "dog_registry",
			schema: logs.Schema{"name": "string", "breed": "string", "weight": "int"},
			logs: logs.JSON{
				rawLog{"name": "max", "breed": "chihuahua", "weight": float64(3), "age": float64(10)},
				rawLog{"name": "spot", "breed": "husky", "weight": float64(130)},
				rawLog{"name": "spike", "breed": "bulldog", "weight": float64(80)},
			},
		},
	}

	for _, tt := range failureCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, service.Ingest(tt.family, tt.schema, tt.logs))
		})
	}
}

// describes a test case for queryCase
type queryCase struct {
	name   string
	query  string
	result error
}

func TestQuery(t *testing.T) {
	// GIVEN
	service := logs.CreateService(&mockDB{})

	// THEN
	successCases := []queryCase{
		{
			name:  "a select query should be executed without problems",
			query: "SELECT * FROM `dog_registry`;",
		},
	}

	for _, tt := range successCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Query(tt.query)
			assert.NoError(t, err)
		})
	}

	failureCases := []queryCase{
		{
			name:  "multiple select statements should return an error",
			query: "SELECT * FROM `dog_registry`; SELECT * FROM `users`;",
		},
		{
			name:  "invalid statements should return an error",
			query: "SELECT * FROMa;",
		},
	}

	for _, tt := range failureCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Query(tt.query)
			assert.Error(t, err)
		})
	}

	errCases := []queryCase{
		{
			name:   "an INSERT statement should return a read only error",
			query:  "INSERT INTO `dog_registry`(`name`, `weight`) VALUES('spot', 130);",
			result: logs.ErrReadOnly,
		},
		{
			name:   "a DROP statement should return a read only error",
			query:  "DROP TABLE `dog_registry`;",
			result: logs.ErrReadOnly,
		},
	}

	for _, tt := range errCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Query(tt.query)
			assert.Equal(t, tt.result, err)
		})
	}
}
