package logger

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

// SentryConfig holds Sentry configuration
type SentryConfig struct {
	DSN              string
	Environment      string
	Release          string
	SampleRate       float64
	TracesSampleRate float64
	Debug            bool
}

// Performance threshold configuration
var (
	// SlowRequestThresholdMs defines the response time threshold in milliseconds
	// above which requests are considered slow and reported as Issues
	SlowRequestThresholdMs float64 = 5.0
)

// SetSlowRequestThreshold allows customizing the slow request threshold
func SetSlowRequestThreshold(thresholdMs float64) {
	SlowRequestThresholdMs = thresholdMs
	log.Printf("🎯 Slow request threshold set to %.1fms", thresholdMs)
}

// InitSentry initializes Sentry with configuration
func InitSentry(config SentryConfig) error {
	return sentry.Init(sentry.ClientOptions{
		Dsn:              config.DSN,
		Environment:      config.Environment,
		Release:          config.Release,
		SampleRate:       config.SampleRate,
		EnableTracing:    true, // Enable performance monitoring
		TracesSampleRate: config.TracesSampleRate,
		Debug:            config.Debug,
		AttachStacktrace: true,
		SendDefaultPII:   true, // Send user information for better context
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Send all events including info level for performance monitoring
			if config.Debug {
				log.Printf("Sending event to Sentry: Level=%s, Message=%s", event.Level, event.Message)
			}
			return event
		},
	})
}

// InitSentryFromEnv initializes Sentry from environment variables
func InitSentryFromEnv() error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return nil
	}

	environment := os.Getenv("SENTRY_ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	release := os.Getenv("SENTRY_RELEASE")

	sampleRate := 1.0
	if sr := os.Getenv("SENTRY_SAMPLE_RATE"); sr != "" {
		if parsed, err := strconv.ParseFloat(sr, 64); err == nil {
			sampleRate = parsed
		}
	}

	tracesSampleRate := 1.0
	if tsr := os.Getenv("SENTRY_TRACES_SAMPLE_RATE"); tsr != "" {
		if parsed, err := strconv.ParseFloat(tsr, 64); err == nil {
			tracesSampleRate = parsed
		}
	}

	debug := false
	if d := os.Getenv("SENTRY_DEBUG"); d == "true" {
		debug = true
	}

	config := SentryConfig{
		DSN:              dsn,
		Environment:      environment,
		Release:          release,
		SampleRate:       sampleRate,
		TracesSampleRate: tracesSampleRate,
		Debug:            debug,
	}

	return InitSentry(config)
}

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
	Error        error
}

// FormatHTTPLog formatiert einen HTTP-Log-Eintrag im Nginx-ähnlichen Format
func FormatHTTPLog(entry HTTPLogEntry) string {
	errorStr := ""
	if entry.Error != nil {
		errorStr = fmt.Sprintf(" error=\"%s\"", entry.Error.Error())
	}

	return fmt.Sprintf("%s - - \"%s %s %s\" %d %d \"%s\" \"%s\" %.3f (%s)%s",
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
		errorStr,
	)
}

// LogHTTPRequest ist ein Middleware-Handler für HTTP-Logging mit Sentry-Integration
func LogHTTPRequest(next http.Handler) http.Handler {
	// Wrap with Sentry HTTP middleware
	sentryHandler := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	})

	return sentryHandler.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get Sentry hub from context
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			hub = sentry.CurrentHub()
		}

		// Start a transaction for performance monitoring
		transaction := sentry.StartTransaction(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		transaction.SetTag("http.method", r.Method)
		transaction.SetTag("http.url", r.URL.String())
		transaction.SetTag("user_agent", r.UserAgent())
		defer transaction.Finish()

		// Update request context with transaction
		r = r.WithContext(transaction.Context())

		// ResponseWriter wrapper um die Größe der Antwort zu erfassen
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		var requestError error

		// Recover from panics and send to Sentry
		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok {
					requestError = err
				} else {
					requestError = fmt.Errorf("panic: %v", rec)
				}

				var panicEventID *sentry.EventID
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelError)
					scope.SetTag("panic", "true")
					scope.SetContext("request", map[string]interface{}{
						"method":     r.Method,
						"url":        r.URL.String(),
						"user_agent": r.UserAgent(),
						"remote_ip":  r.RemoteAddr,
					})
					panicEventID = hub.CaptureException(requestError)
				})

				if panicEventID != nil && *panicEventID != "" {
					log.Printf("SENTRY: Panic captured with ID: %s", *panicEventID)
				}

				// Set error status and re-panic to let HTTP server handle it
				rw.statusCode = http.StatusInternalServerError
				panic(rec)
			}
		}()

		// Handler aufrufen
		next.ServeHTTP(rw, r)

		// Log-Eintrag erstellen
		entry := HTTPLogEntry{
			RemoteAddr:   r.RemoteAddr,
			RequestTime:  start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Protocol:     r.Proto,
			StatusCode:   rw.statusCode,
			ResponseSize: rw.size,
			ResponseTime: time.Since(start),
			UserAgent:    r.UserAgent(),
			Referer:      r.Referer(),
			Host:         r.Host,
			Error:        requestError,
		}

		// Set transaction status based on HTTP status code
		if rw.statusCode >= 400 {
			transaction.Status = sentry.HTTPtoSpanStatus(rw.statusCode)
		}

		// Only send Server Errors (500+) to Sentry if no alert was already sent
		if rw.statusCode >= 500 {
			// Check if alert was already sent via LogError
			alertAlreadySent := false

			// Simple check: if we can access the hub and it has our marker
			if hub != nil {
				// We'll assume alert was sent if this is application error (not panic)
				alertAlreadySent = (requestError == nil) // No panic error = likely application error
			}

			if !alertAlreadySent {
				responseTimeMs := float64(entry.ResponseTime.Nanoseconds()) / 1000000

				var eventID *sentry.EventID
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelError)
					scope.SetTag("issue_type", "server_error")
					scope.SetTag("source", "http_handler")
					scope.SetTag("http.status_code", strconv.Itoa(rw.statusCode))
					scope.SetTag("http.method", r.Method)
					scope.SetTag("http.path", r.URL.Path)
					scope.SetTag("response_time_ms", fmt.Sprintf("%.1f", responseTimeMs))

					scope.SetContext("request", map[string]interface{}{
						"method":           r.Method,
						"url":              r.URL.String(),
						"user_agent":       r.UserAgent(),
						"remote_ip":        r.RemoteAddr,
						"response_time_ms": responseTimeMs,
						"response_time_ns": entry.ResponseTime.Nanoseconds(),
						"response_size":    entry.ResponseSize,
						"host":             r.Host,
						"status_code":      rw.statusCode,
					})

					if requestError != nil {
						eventID = hub.CaptureException(requestError)
					} else {
						eventID = hub.CaptureMessage(fmt.Sprintf("Server Error: %s %s → %d (%.1fms)",
							r.Method, r.URL.Path, rw.statusCode, responseTimeMs))
					}
				})

				if eventID != nil && *eventID != "" {
					log.Printf("SENTRY: Server Error captured with ID: %s", *eventID)
				}
			}
		}
		// All other requests (200-499) are not sent to Sentry

		// Log ausgeben
		log.Println(FormatHTTPLog(entry))
	}))
}

// ErrorDetails holds error information for HTTP context
type ErrorDetails struct {
	Error   error
	Message string
	Tags    map[string]string
	Extra   map[string]interface{}
}

// contextKey type for context keys
type contextKey string

const errorDetailsKey contextKey = "errorDetails"

// LogError logs an error and sends alert to monitoring system
func LogError(err error, message string, tags map[string]string, extra map[string]interface{}) {
	// Send alert to monitoring system with all details
	var eventID *sentry.EventID
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetTag("source", "application")
		scope.SetTag("error_message", message)

		for key, value := range tags {
			scope.SetTag(key, value)
		}

		if extra != nil {
			scope.SetContext("extra", extra)
		}

		eventID = sentry.CaptureException(err)
	})

	if eventID != nil && *eventID != "" {
		log.Printf("SENTRY: Alert sent to monitoring system with ID: %s", *eventID)

		// Mark that alert was already sent for this request
		hub := sentry.CurrentHub()
		if hub != nil {
			hub.Scope().SetTag("alert_sent", "true")
		}
	}
}

// LogWarning logs a warning to both standard logger and Sentry
func LogWarning(message string, tags map[string]string, extra map[string]interface{}) {
	log.Printf("WARNING: %s", message)

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelWarning)

		for key, value := range tags {
			scope.SetTag(key, value)
		}

		if extra != nil {
			scope.SetContext("extra", extra)
		}

		sentry.CaptureMessage(message)
	})
}

// LogInfo logs an info message to standard logger only (not sent to Sentry)
func LogInfo(message string, tags map[string]string, extra map[string]interface{}) {
	log.Printf("INFO: %s", message)
	// Info messages are not sent to Sentry - only local logging
}

// LogPerformance logs performance metrics to standard logger only (not sent to Sentry)
func LogPerformance(operation string, duration time.Duration, tags map[string]string, extra map[string]interface{}) {
	message := fmt.Sprintf("Performance: %s took %v", operation, duration)
	log.Printf("PERFORMANCE: %s", message)
	// Performance metrics are not sent to Sentry - only local logging
}

// LogMetric logs custom metrics to standard logger only (not sent to Sentry)
func LogMetric(name string, value interface{}, unit string, tags map[string]string) {
	message := fmt.Sprintf("Metric: %s = %v %s", name, value, unit)
	log.Printf("METRIC: %s", message)
	// Metrics are not sent to Sentry - only local logging
}

// LogDatabaseQuery logs database performance to standard logger only (not sent to Sentry)
func LogDatabaseQuery(query string, duration time.Duration, rowsAffected int64, tags map[string]string) {
	message := fmt.Sprintf("DB Query took %v, affected %d rows", duration, rowsAffected)
	log.Printf("DB_PERFORMANCE: %s", message)
	// Database performance is not sent to Sentry - only local logging
}

// LogAPICall logs external API call performance to standard logger only (not sent to Sentry)
func LogAPICall(endpoint string, method string, statusCode int, duration time.Duration, tags map[string]string) {
	message := fmt.Sprintf("API Call: %s %s returned %d in %v", method, endpoint, statusCode, duration)
	log.Printf("API_PERFORMANCE: %s", message)
	// API call performance is not sent to Sentry - only local logging
}

// Flush flushes any pending Sentry events
func Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
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
