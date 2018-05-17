package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kolide/databalancer-logan/pkg/logs"
	"github.com/pkg/errors"
)

// LogService contains the methods for the log processing service
type LogService interface {
	Ingest(family logs.Family, schema logs.Schema, logs []logs.Raw) error
}

// handler is an internal wrapper around HTTP handlers that allows us to pass
// some services for our handlers
type handler struct {
	logSvc LogService
}

// IngestLogBody is the format of the JSON required in the body of a request to
// the IngestLogHandler
type IngestLogBody struct {
	Family logs.Family `json:"family"`
	Schema logs.Schema `json:"schema"`
	Logs   []logs.Raw  `json:"logs"`
}

// ServeHTTP implements the HandlerFunc interface in the net/http package.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// PUT /api/log
	if r.URL.Path == "/api/log" && r.Method == "PUT" {
		h.ingestLogHandler(w, r)
		return
	}

	// handle route not found
	http.Error(w, "Route not found: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
}

// ingestLogHandler is an HTTP handler which ingests logs from the network
func (h *handler) ingestLogHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// decode the request
	var body IngestLogBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "An error occured parsing JSON: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error parsing json of log: %+v\n", err)
		return
	}

	// ingest the logs through the service
	if err = h.logSvc.Ingest(body.Family, body.Schema, body.Logs); err != nil {
		http.Error(w, "An error occured ingesting logs: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error ingesting log: %+v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// HTTP creates a new HTTP server to handle requests
func HTTP(address string, logs LogService) error {
	log.Printf("Starting HTTP server on %s\n", address)

	if err := http.ListenAndServe(address,
		handler{
			logSvc: logs,
		},
	); err != nil {
		return errors.Wrapf(err, "starting server at address '%s'", address)
	}

	return nil
}