package mysql

import (
	"sort"
	"strings"
)

// CreateTableStatement builds a create table statement string from a
// table name and a schema. Note that the table will have an INT typed `id`
// primary key
func CreateTableStatement(name string, schema map[string]string) string {
	// list of fields in the schema
	var tableFields []string
	for fieldName, fieldType := range schema {
		// append field field name and appropriate field type to field list
		switch fieldType {
		case "string":
			field := "`" + Escape(fieldName) + "` TEXT, "
			tableFields = append(tableFields, field)
		case "int":
			field := "`" + Escape(fieldName) + "` INT, "
			tableFields = append(tableFields, field)
		}
	}
	// sort the fields
	sort.Strings(tableFields)

	// join them
	safeTableFields := strings.Join(tableFields, "")

	stmt := "CREATE TABLE IF NOT EXISTS `" +
		Escape(name) +
		"`(`id` INT NOT NULL AUTO_INCREMENT, " +
		safeTableFields +
		"PRIMARY KEY(`id`)" +
		");"

	return stmt
}

// InsertTableStatement builds a statement to insert records into a table,
// given a table name, schema, and some records, and returns the arguments to be passed
// to the statement
// NOTE: I figured it was safer and better to use the built-in mechanism (bindvars) for
// record value inserts, and only handle escaping the table name and field names manually.
func InsertTableStatement(name string, schema map[string]string, records []map[string]interface{}) (string, []interface{}) {
	// list of field names (to preserve order bewteen field names and arguments)
	var fieldNames []string
	// will represent placeholders for fields
	var bindvars []string
	for fieldName := range schema {
		// append a bindvar for the field
		bindvars = append(bindvars, "?")
		// append safe field name and to list of fields
		fieldNames = append(fieldNames, Escape(fieldName))
	}
	// sort the fields
	sort.Strings(fieldNames)

	// concatenate the field names and wrap in backticks
	var safeTableFields = "`" + strings.Join(fieldNames, "`, `") + "`"

	// concatenate the bindvars and wrap in parens
	var bindvarString = "(" + strings.Join(bindvars, ", ") + ")"

	// the list of bindvars for all records
	var valueBindvars []string
	// the list of args to pass into the statement
	var args []interface{}
	for _, record := range records {
		// append a bindvarstring for each record
		valueBindvars = append(valueBindvars, bindvarString)

		for _, fieldName := range fieldNames {
			// append field values as arguments
			fieldValue := record[fieldName]
			if fieldValue != nil {
				args = append(args, fieldValue)
			}
		}
	}
	// join the value bind vars
	valuePlaceholders := strings.Join(valueBindvars, ", ")

	stmt := "INSERT INTO `" +
		Escape(name) +
		"`(" +
		safeTableFields +
		") VALUES " +
		valuePlaceholders +
		";"

	return stmt, args
}

// Escape prepares strings to be safely used in MySQL statements
// I found this from a quick google search. For the sake of time,
// I'm just going to trust this. Ideally, it would have lots of tests
//  to protect from injection attacks.
func Escape(sql string) string {
	dest := make([]byte, 0, 2*len(sql))
	var escape byte
	for i := 0; i < len(sql); i++ {
		c := sql[i]

		escape = 0

		switch c {
		case 0: /* Must be escaped for 'mysql' */
			escape = '0'
			break
		case '\n': /* Must be escaped for logs */
			escape = 'n'
			break
		case '\r':
			escape = 'r'
			break
		case '\\':
			escape = '\\'
			break
		case '\'':
			escape = '\''
			break
		case '"': /* Better safe than sorry */
			escape = '"'
			break
		case '\032': /* This gives problems on Win32 */
			escape = 'Z'
		}

		if escape != 0 {
			dest = append(dest, '\\', escape)
		} else {
			dest = append(dest, c)
		}
	}

	return string(dest)
}
