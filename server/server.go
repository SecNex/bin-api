package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/secnex/bin-api/logger"
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

// initSentry initializes Sentry for performance monitoring
func (s *Server) initSentry() {
	logger.InitSentryFromEnv()
}

// Alle Anfragen, die an den Server gesendet werden, sollen als als JSON-Objekt verarbeitet werden und der Body, die Parameter und Headers sollen anschließend als JSON zurückgegeben werden.
func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Performance wird automatisch durch logger.LogHTTPRequest erfasst

	response := make(map[string]interface{})

	// Body verarbeiten, wenn vorhanden
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			// Log error to Sentry
			logger.LogError(err, "Failed to read request body",
				map[string]string{"endpoint": "handle_request"},
				map[string]interface{}{"url": r.URL.String()})
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(body) > 0 {
			var bodyData interface{}
			if err := json.Unmarshal(body, &bodyData); err != nil {
				// Log JSON parsing error to Sentry
				logger.LogError(err, "Failed to parse JSON body",
					map[string]string{"endpoint": "handle_request"},
					map[string]interface{}{"body": string(body)})
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response)
}

func (s *Server) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) Start() {
	// Initialize Sentry first
	s.initSentry()

	log.Printf("Starting server on %s", s.String())

	// Setup graceful shutdown
	defer func() {
		logger.Flush(10 * time.Second)
	}()

	router := http.NewServeMux()
	router.HandleFunc("/", s.HandleRequest)
	router.HandleFunc("/healthz", s.Healthz)
	handler := logger.LogHTTPRequest(router)

	if err := http.ListenAndServe(s.String(), handler); err != nil {
		logger.LogError(err, "Server failed to start",
			map[string]string{"host": s.Host, "port": fmt.Sprintf("%d", s.Port)},
			nil)
	}
}
