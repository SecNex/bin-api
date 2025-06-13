package logger

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// HTTPLogEntry repräsentiert einen einzelnen HTTP-Log-Eintrag
type HTTPLogEntry struct {
	Host         string
	RemoteAddr   string
	RequestTime  time.Time
	Method       string
	Path         string
	Protocol     string
	StatusCode   int
	ResponseSize int64
	ResponseTime time.Duration
	UserAgent    string
	Referer      string
}

// FormatHTTPLog formatiert einen HTTP-Log-Eintrag im Nginx-ähnlichen Format
func FormatHTTPLog(entry HTTPLogEntry) string {
	return fmt.Sprintf("%s - - \"%s %s %s\" %d %d \"%s\" \"%s\" %.3f (%s)",
		entry.RemoteAddr,
		entry.Method,
		entry.Path,
		entry.Protocol,
		entry.StatusCode,
		entry.ResponseSize,
		entry.Referer,
		entry.UserAgent,
		entry.ResponseTime.Seconds(),
		entry.Host,
	)
}

// LogHTTPRequest ist ein Middleware-Handler für HTTP-Logging
func LogHTTPRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// ResponseWriter wrapper um die Größe der Antwort zu erfassen
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Handler aufrufen
		next.ServeHTTP(rw, r)

		// Log-Eintrag erstellen
		entry := HTTPLogEntry{
			RemoteAddr:   r.RemoteAddr,
			Method:       r.Method,
			Path:         r.URL.Path,
			Protocol:     r.Proto,
			StatusCode:   rw.statusCode,
			ResponseSize: rw.size,
			ResponseTime: time.Since(start),
			UserAgent:    r.UserAgent(),
			Referer:      r.Referer(),
			Host:         r.Host,
		}

		// Log ausgeben
		log.Println(FormatHTTPLog(entry))
	})
}

// responseWriter ist ein Wrapper für http.ResponseWriter
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

// WriteHeader überschreibt die WriteHeader-Methode
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write überschreibt die Write-Methode
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += int64(size)
	return size, err
}
