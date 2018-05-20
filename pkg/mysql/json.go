package mysql

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/pkg/errors"
)

// RowsToJSON converts sql rows to a generic representation
// that can be marshalled to JSON
// NOTE: only supports JSON primitive types (string, int64, boolean, null)
// and will return nil values for unsupported fields
func RowsToJSON(rows *sql.Rows) ([]map[string]interface{}, error) {
	// get the columns of the result
	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "getting columns of rows")
	}

	// create a list of values
	values := make([]interface{}, len(columns))
	// create a list of pointers to the values
	valPtrs := make([]interface{}, len(values))
	for i := range values {
		valPtrs[i] = &values[i]
	}

	// make a list of results
	var results []map[string]interface{}
	for rows.Next() {
		// create a row
		row := make(map[string]interface{})
		// scan the row
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, err
		}
		// try to parse the different JSON types from the bytes of the values
		for i, value := range values {
			if value == nil {
				row[columns[i]] = nil
				continue
			}
			valueBytes := value.([]byte)
			if float, ok := strconv.ParseFloat(string(valueBytes), 64); ok == nil {
				row[columns[i]] = float
			} else if str := string(valueBytes); "string" == fmt.Sprintf("%T", str) {
				row[columns[i]] = str
			} else if boolean, ok := strconv.ParseBool(string(valueBytes)); ok == nil {
				row[columns[i]] = boolean
			} else {
				log.Printf("Unsupported column type %T of %v\n", valueBytes, valueBytes)
				row[columns[i]] = nil
			}
		}
		results = append(results, row)
	}
	return results, nil
}
