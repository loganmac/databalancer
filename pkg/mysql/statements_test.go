package mysql_test

import (
	"testing"

	"github.com/kolide/databalancer-logan/pkg/mysql"
	"github.com/stretchr/testify/assert"
)

// utility types to clean up tests
type schema map[string]string
type record map[string]interface{}
type records []map[string]interface{}

// describes a test case for CreateTableStatement
type createCase struct {
	name      string
	tableName string
	schema    schema
	statement string
}

func TestCreateTableStatement(t *testing.T) {
	cases := []createCase{
		{
			name:      "can construct a create statement from a schema",
			tableName: "dog_registry",
			schema:    schema{"name": "string", "breed": "string", "weight": "int"},
			statement: "CREATE TABLE IF NOT EXISTS `dog_registry`(`id` INT NOT NULL AUTO_INCREMENT, `breed` TEXT, `name` TEXT, `weight` INT, PRIMARY KEY(`id`));",
		},
		// NOTE: not sure if this is even desirable
		{
			name:      "can construct a create statement from an empty schema",
			tableName: "cat_registry",
			schema:    schema{},
			statement: "CREATE TABLE IF NOT EXISTS `cat_registry`(`id` INT NOT NULL AUTO_INCREMENT, PRIMARY KEY(`id`));",
		},
		{
			name:      "escapes attempts to inject sql",
			tableName: "criminal_registry",
			schema:    schema{"name": "string", "test; DROP TABLE users": "test; DROP TABLE users"},
			statement: "CREATE TABLE IF NOT EXISTS `criminal_registry`(`id` INT NOT NULL AUTO_INCREMENT, `name` TEXT, PRIMARY KEY(`id`));",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.statement, mysql.CreateTableStatement(tt.tableName, tt.schema))
		})
	}
}

// describes a test case for InsertTableStatement
type insertCase struct {
	name      string
	tableName string
	schema    schema
	records   records
	statement string
	args      []interface{}
}

func TestInsertTableStatement(t *testing.T) {
	cases := []insertCase{
		{
			name:      "can construct an insert statement from a schema and logs",
			tableName: "dog_registry",
			schema:    schema{"name": "string", "breed": "string", "weight": "int"},
			records: records{
				record{"name": "max", "breed": "chihuahua", "weight": float64(3)},
				record{"name": "spot", "breed": "husky", "weight": float64(130)},
				record{"name": "spike", "breed": "bulldog", "weight": float64(80)},
			},
			statement: "INSERT INTO `dog_registry`(`breed`, `name`, `weight`) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?);",
			args: []interface{}{
				"chihuahua", "max", float64(3),
				"husky", "spot", float64(130),
				"bulldog", "spike", float64(80),
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stmt, args := mysql.InsertTableStatement(tt.tableName, tt.schema, tt.records)
			assert.Equal(t, tt.statement, stmt)
			assert.Equal(t, tt.args, args)
		})
	}
}
