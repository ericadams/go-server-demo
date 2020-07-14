package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

const serviceName = "go-server-demo"
const countHeader = "X-REQUEST-COUNT"
const reasonHeader = "X-REASON"

var (
	logger       log.Logger
	requestCount int
)

type Nestable struct {
	NestedObject Nested    `json:"nested-object,omitempty"`
	Name         string    `json:"name"`
	ID           uuid.UUID `json:"unique-identifier,omitempty"`
	Number       int       `json:"number"`
}

type Nested struct {
	List []string          `json:"nested-list,omitempty"`
	Dict map[string]string `json:"data-bag,omitempty"`
}

type QueryError struct {
	Reason    string
	Timestamp time.Time
}

func countRequest(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestCount++
	w.Header().Set(countHeader, strconv.Itoa(requestCount))
}

// Index is the handler for requests to '/'
func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Infoln("handling index")
	fmt.Fprint(w, "Welcome!\n")
}

// Hello is the handler which is a lightly modified version of the trivial httprouter example
func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Infoln("handling hello")
	w.Header().Set(countHeader, strconv.Itoa(requestCount))
	fmt.Fprintf(w, "hello, %s!\n\tYour RequestCount is: %d\n", ps.ByName("name"), requestCount)
}

//QueryParamDemo is the handler for /query and requires a valid UUID v4 to be passed as a query parameter
func QueryParamDemo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	qry := r.URL.Query()
	log.WithField("query-params", qry).Debugf("we got %d params, buddy!\n", len(qry))

	if len(qry) == 0 {
		reason := "EMPTY_PARAMS"
		writeBadRequest(w, errors.New(reason))
		return
	}

	var id uuid.UUID
	id, err := uuid.Parse(qry.Get("id"))
	if err != nil {
		writeBadRequest(w, err)
		return
	}

	bytes, err := json.Marshal(Nestable{
		ID:     id,
		Name:   "QueryHandler",
		Number: requestCount,
	})
	if err != nil {
		writeInternalServerError(w, err)
	}
	w.Write(bytes)
}

//
func writeInternalServerError(w http.ResponseWriter, err error) {
	log.WithError(err)
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "Bewarror: %s\n", err.Error())
}

func writeBadRequest(w http.ResponseWriter, err error) {
	log.Warn(err)
	qe := &QueryError{
		Reason:    err.Error(),
		Timestamp: time.Now().UTC(),
	}
	bytes, marshalErr := json.Marshal(qe)
	if marshalErr != nil {
		writeInternalServerError(w, marshalErr)
		return
	}

	w.Header().Set(reasonHeader, err.Error())
	w.WriteHeader(http.StatusBadRequest)
	w.Write(bytes)
}

func chain(handles ...httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		for _, h := range handles {
			h(w, r, ps)
		}
	}

}

// specialty function that runs on startup; do not abuse
func init() {
	// panic if uuid is unusable
	fmt.Println(uuid.New())

	// set logrus defaults
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func main() {
	log.Debugf("%s service initiated\n", serviceName)
	router := httprouter.New()
	router.GET("/", chain(countRequest, Index))
	router.GET("/hello/:name", chain(countRequest, Hello))
	router.GET("/query", chain(countRequest, QueryParamDemo))

	log.Fatal(http.ListenAndServe(":8080", router))
}
