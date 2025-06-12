package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Server struct {
	Host string
	Port int
}

func NewServer(host string, port int) *Server {
	return &Server{
		Host: host,
		Port: port,
	}
}

func (s *Server) String() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Alle Anfragen, die an den Server gesendet werden, sollen als als JSON-Objekt verarbeitet werden und der Body, die Parameter und Headers sollen anschließend als JSON zurückgegeben werden.
func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]interface{})

	// Body verarbeiten, wenn vorhanden
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(body) > 0 {
			var bodyData interface{}
			if err := json.Unmarshal(body, &bodyData); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			response["body"] = bodyData
		} else {
			response["body"] = map[string]interface{}{}
		}
	} else {
		response["body"] = map[string]interface{}{}
	}

	// Header verarbeiten
	headers := make(map[string]interface{})
	for key, values := range r.Header {
		if len(values) == 1 {
			headers[key] = values[0]
		} else {
			headers[key] = values
		}
	}
	response["headers"] = headers

	// Query-Parameter verarbeiten
	queries := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			queries[key] = values[0]
		} else {
			queries[key] = values
		}
	}
	response["queries"] = queries

	// URL-Parameter verarbeiten
	response["params"] = map[string]interface{}{}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) Start() {
	http.HandleFunc("/", s.HandleRequest)
	http.ListenAndServe(s.String(), nil)
}
