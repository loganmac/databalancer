package main

import (
	"flag"
	"log"

	"github.com/kolide/databalancer-logan/pkg/logs"
	"github.com/kolide/databalancer-logan/pkg/mysql"
	"github.com/kolide/databalancer-logan/pkg/server"
)

func main() {
	// Key variables are set as command-line flags
	dbUsername := flag.String("mysql_username", "dbuser", "The MySQL user account username")
	dbPassword := flag.String("mysql_password", "dbpassword", "The MySQL user account password")
	dbAddress := flag.String("mysql_address", "localhost:3306", "The MySQL server address")
	dbName := flag.String("mysql_database", "databalancer", "The MySQL database to use")
	serverAddress := flag.String("server_address", ":8080", "The address and port to serve the local HTTP server")

	flag.Parse()

	// Using data from command-line flags, we create a MySQL client
	dbClient, err := mysql.NewClient(*dbUsername, *dbPassword, *dbAddress, *dbName)
	if err != nil {
		log.Fatalf("Failed connecting to MySQL: %+v", err)
	}

	// create the logs service with the database client
	logSvc := logs.CreateService(dbClient)

	// Now that we have performed all required flag parsing and state
	// initialization, we create and launch our HTTP web server for our
	// micro-service
	if err := server.HTTP(*serverAddress, logSvc); err != nil {
		log.Fatalf("Failed to start server: %+v", err)
	}
}
