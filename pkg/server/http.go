package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kolide/databalancer-logan/pkg/logs"
	"github.com/pkg/errors"
)

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

// handler is an internal wrapper around HTTP handlers that allows us to pass
// some services for our handlers
type handler struct {
	logSvc LogService
}

// LogService contains the methods for the log processing service
type LogService interface {
	Ingest(family logs.Family, schema logs.Schema, logs logs.JSON) error
	Query(query string) (logs.JSON, error)
}

// ServeHTTP implements the HandlerFunc interface in the net/http package.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// PUT /api/log
	if r.URL.Path == "/api/log" && r.Method == "PUT" {
		h.ingestLogHandler(w, r)
		return
	}

	// POST /api/query
	if r.URL.Path == "/api/query" && r.Method == "POST" {
		h.queryHandler(w, r)
		return
	}

	// handle route not found
	http.Error(w, "Route not found: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
}

// ingestLogBody is the format of the JSON required in the body of a request to
// the IngestLogHandler
type ingestLogBody struct {
	Family logs.Family `json:"family"`
	Schema logs.Schema `json:"schema"`
	Logs   logs.JSON   `json:"logs"`
}

// ingestLogHandler is an HTTP handler which ingests logs from the network
func (h *handler) ingestLogHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// decode the request
	var body ingestLogBody
	err := json.NewDecoder(r.Body).Decode(&body)
	// TODO: Add validation, responding about how the request was invalid with a 400 request
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

// queryBody is the format of the JSON required in the body of a request to
// perform a query
type queryBody struct {
	Query string `json:"query"`
}

// query response is the format of the response to a query
type queryResponse struct {
	Results logs.JSON `json:"results"`
}

// queryHandler is an HTTP handler which ingests logs from the network
func (h *handler) queryHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// decode the request
	var body queryBody
	err := json.NewDecoder(r.Body).Decode(&body)
	// TODO: Add validation, responding about how the request was invalid with a 400 request
	if err != nil {
		http.Error(w, "An error occured parsing JSON: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error parsing json of log: %+v\n", err)
		return
	}

	// ingest the logs through the service
	results, err := h.logSvc.Query(body.Query)
	if err != nil {
		http.Error(w, "An error occured querying logs: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error querying log: %+v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err := json.NewEncoder(w).Encode(queryResponse{Results: results}); err != nil {
		http.Error(w, "An error occured encoding the results: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error encoding results: %+v\n", err)
		return
	}
}
