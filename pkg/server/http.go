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
	DescribeLogs() (logs.JSON, error)
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

	// GET /api/describe
	if r.URL.Path == "/api/describe" && r.Method == "GET" {
		h.describeHandler(w, r)
		return
	}

	// handle route not found
	http.Error(w, "Route not found: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
}

// ingestLogHandler is an HTTP handler which ingests logs from the network
func (h *handler) ingestLogHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// decode the request
	var body struct {
		Family logs.Family `json:"family"`
		Schema logs.Schema `json:"schema"`
		Logs   logs.JSON   `json:"logs"`
	}
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

// queryHandler is an HTTP handler which ingests logs from the network
func (h *handler) queryHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// decode the request
	var body struct {
		Query string `json:"query"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	// TODO: Add validation, responding about how the request was invalid with a 400 request
	if err != nil {
		http.Error(w, "An error occured parsing JSON: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error parsing json of log: %+v\n", err)
		return
	}

	// query the logs service
	results, err := h.logSvc.Query(body.Query)
	if err != nil {
		http.Error(w, "An error occured querying logs: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error querying log: %+v\n", err)
		return
	}

	// set json content-type
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// format the response as JSON with a results field that's a list of results
	var queryResponse struct {
		Results logs.JSON `json:"results"`
	}
	queryResponse.Results = results

	if err := json.NewEncoder(w).Encode(queryResponse); err != nil {
		http.Error(w, "An error occured encoding the results: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error encoding results: %+v\n", err)
		return
	}
}

// describeHandler is an HTTP handler which ingests logs from the network
func (h *handler) describeHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// describe the logs of the log service
	tables, err := h.logSvc.DescribeLogs()
	if err != nil {
		http.Error(w, "An error occured describing logs: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error describing logs: %+v\n", err)
		return
	}

	// set json content-type
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// format the response as JSON with a results field that's a list of results
	var describeResponse struct {
		Tables logs.JSON `json:"tables"`
	}
	describeResponse.Tables = tables

	if err := json.NewEncoder(w).Encode(describeResponse); err != nil {
		http.Error(w, "An error occured encoding the results: "+err.Error(), http.StatusInternalServerError)
		// TODO: change to structured logger and use debug level logging, or report to error aggregation service
		log.Printf("error encoding results: %+v\n", err)
		return
	}
}
