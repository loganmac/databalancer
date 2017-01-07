package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dbUsername    = flag.String("mysql_username", "dbuser", "The MySQL user account username")
	dbPassword    = flag.String("mysql_password", "dbpassword", "The MySQL user account password")
	dbAddress     = flag.String("mysql_address", "localhost:3306", "The MySQL server address")
	dbName        = flag.String("mysql_database", "databalancer", "The MySQL database to use")
	serverAddress = flag.String("server_address", ":8080", "The address and port to serve the local HTTP server")
)

// ctxKey is the type which we use to identify values stored in contexts
type ctxKey int

// dbCtxKey is the database context key. objects stored in a context using this
// key should be of type *sql.DB
var dbCtxKey ctxKey

// RawLog is an example struct which is used to store raw logs in the database
type RawLog struct {
	ID     int64
	Family string
	Log    string
}

// Create issues the SQL CREATE TABLE statement to the supplied DB for RawLog
func (rl *RawLog) Create(db *sql.DB) error {
	stmt, err := db.Prepare(
		"CREATE TABLE `raw_logs` (" +
			"`id` INT NOT NULL AUTO_INCREMENT," +
			"`family` TEXT," +
			"`log` TEXT," +
			"PRIMARY KEY(`id`)" +
			");",
	)
	if err != nil {
		return err
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

// Drop issues the SQL DROP TABLE statement to the supplied DB for RawLog
func (rl *RawLog) Drop(db *sql.DB) error {
	stmt, err := db.Prepare("DROP TABLE IF EXISTS `raw_logs`;")
	if err != nil {
		return err
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

// Name is the name of the RawLog table in MySQL
func (rl *RawLog) Name() string {
	return "raw_log"
}

// NewRawLog creates a new RawLog record in using the supplied db
func NewRawLog(db *sql.DB, log *RawLog) (*RawLog, error) {
	stmt, err := db.Prepare("INSERT INTO `raw_logs`(`family`, `log`) VALUES(?, ?);")
	if err != nil {
		return nil, err
	}

	res, err := stmt.Exec(log.Family, log.Log)
	if err != nil {
		return nil, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	log.ID = lastID

	return log, nil
}

// databaseTable is an interface which defines a few core methods used for
// database management. All database tables should comply to this interface and
// added to the databaseTables array
type databaseTable interface {
	Create(*sql.DB) error
	Drop(*sql.DB) error
	Name() string
}

// Any tables in this array will automatically be dropped and re-created every
// time the binary starts. This may become undesired behavior eventually.
var databaseTables = [...]databaseTable{
	&RawLog{},
}

// handler is an internal wrapper around HTTP handlers that allows us to pass a
// context with our HTTP handlers.
type handler struct {
	ctx              context.Context
	IngestLogHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request)
}

// ServeHTTP implements the HandlerFunc interface in the net/http package.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/log" && r.Method == "PUT" {
		h.IngestLogHandler(h.ctx, w, r)
		return
	}

	http.Error(w, "Route not found: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
}

// IngestLogBody is the format of the JSON required in the body of a request to
// the IngestLogHandler
type IngestLogBody struct {
	Family string                   `json:"family"`
	Schema map[string]string        `json:"schema"`
	Logs   []map[string]interface{} `json:"logs"`
}

// IngestLogHandler is an HTTP handler which ingests logs from the network
func IngestLogHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var body IngestLogBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "An error occured parsing JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = IngestLog(ctx, body)
	if err != nil {
		http.Error(w, "An error occured ingesting logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// IngestLog is a method which ingests logs into the database
func IngestLog(ctx context.Context, body IngestLogBody) error {
	log.Printf("Received logs for the %s log family\n", body.Family)
	for column, columnType := range body.Schema {
		log.Printf("Log values for the field %s of the %s log will be of type %s\n", column, body.Family, columnType)
	}

	for _, logEvent := range body.Logs {
		log.Printf("Handling a new log event for the %s log family\n", body.Family)
		for field, value := range logEvent {
			columnType, ok := body.Schema[field]
			if !ok {
				return fmt.Errorf(
					"Data type for the field %s was not specified in the %s schema map\n",
					field,
					body.Family,
				)
			}
			switch columnType {
			case "string":
				log.Printf("The value of the %s field in the %s log event is %s\n", field, body.Family, value.(string))
			case "int":
				log.Printf("The value of the %s field in the %s log event is %d\n", field, body.Family, int(value.(float64)))
			default:
				return fmt.Errorf("Unsupported data type in %s log for the field %s: %s\n", body.Family, field, columnType)
			}
		}

		// Marshal the log event back into JSON to store it in the database
		rawLogContent, err := json.Marshal(logEvent)
		if err != nil {
			return err
		}
		rawLog := &RawLog{
			Family: body.Family,
			Log:    string(rawLogContent),
		}

		db, ok := ctx.Value(dbCtxKey).(*sql.DB)
		if !ok {
			return fmt.Errorf("DB not set in context")
		}

		rawLog, err = NewRawLog(db, rawLog)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// Key variables are set as command-line flags
	flag.Parse()

	// Using data from command-line parameters, we create a MySQL connection
	// string
	connectionString := fmt.Sprintf(
		"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		*dbUsername,
		*dbPassword,
		*dbAddress,
		*dbName,
	)
	// Using our connection string, we attempt to open a MySQL connection
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatalf("Could not open the database: %s\n", err)
	}

	// Now, we ensure that can communicate with the database
	err = db.Ping()
	if err != nil {
		log.Fatalf("Could not communicate with the database: %s\n", err)
	}
	log.Printf("Connected to MySQL as %s at %s\n", *dbUsername, *dbAddress)

	for _, table := range databaseTables {
		err = table.Drop(db)
		if err != nil {
			log.Fatalf("Failed dropping %s: %s\n", table.Name(), err)
		}

		err = table.Create(db)
		if err != nil {
			log.Fatalf("Failed creating %s: %s\n", table.Name(), err)
		}
	}

	log.Printf("Starting HTTP server on %s\n", *serverAddress)

	// Now that we have performed all required flag parsing and state
	// initialization, we create and launch our HTTP web server for our
	// micro-service
	http.ListenAndServe(*serverAddress,
		handler{
			ctx:              context.WithValue(context.Background(), dbCtxKey, db),
			IngestLogHandler: IngestLogHandler,
		},
	)
}
