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

// InitSentry initializes Sentry with configuration
func InitSentry(config SentryConfig) error {
	return sentry.Init(sentry.ClientOptions{
		Dsn:              config.DSN,
		Environment:      config.Environment,
		Release:          config.Release,
		SampleRate:       config.SampleRate,
		TracesSampleRate: config.TracesSampleRate,
		Debug:            config.Debug,
		AttachStacktrace: true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Filter out or modify events before sending to Sentry
			if event.Level == sentry.LevelInfo {
				return nil // Don't send info level events
			}
			return event
		},
	})
}

// InitSentryFromEnv initializes Sentry from environment variables
func InitSentryFromEnv() error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Println("SENTRY_DSN not set, Sentry disabled")
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

	tracesSampleRate := 0.1
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

				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelError)
					scope.SetTag("panic", "true")
					scope.SetContext("request", map[string]interface{}{
						"method":     r.Method,
						"url":        r.URL.String(),
						"user_agent": r.UserAgent(),
						"remote_ip":  r.RemoteAddr,
					})
					hub.CaptureException(requestError)
				})

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

		// Send error responses to Sentry
		if rw.statusCode >= 500 {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.LevelError)
				scope.SetTag("http.status_code", strconv.Itoa(rw.statusCode))
				scope.SetContext("request", map[string]interface{}{
					"method":        r.Method,
					"url":           r.URL.String(),
					"user_agent":    r.UserAgent(),
					"remote_ip":     r.RemoteAddr,
					"response_time": entry.ResponseTime.Milliseconds(),
					"response_size": entry.ResponseSize,
				})

				if requestError != nil {
					hub.CaptureException(requestError)
				} else {
					hub.CaptureMessage(fmt.Sprintf("HTTP %d: %s %s", rw.statusCode, r.Method, r.URL.Path))
				}
			})
		} else if rw.statusCode >= 400 {
			// Log client errors as warnings
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetLevel(sentry.LevelWarning)
				scope.SetTag("http.status_code", strconv.Itoa(rw.statusCode))
				scope.SetContext("request", map[string]interface{}{
					"method":     r.Method,
					"url":        r.URL.String(),
					"user_agent": r.UserAgent(),
					"remote_ip":  r.RemoteAddr,
				})
				hub.CaptureMessage(fmt.Sprintf("HTTP %d: %s %s", rw.statusCode, r.Method, r.URL.Path))
			})
		}

		// Log ausgeben
		log.Println(FormatHTTPLog(entry))
	}))
}

// LogError logs an error to both standard logger and Sentry
func LogError(err error, message string, tags map[string]string, extra map[string]interface{}) {
	log.Printf("ERROR: %s: %v", message, err)

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)

		for key, value := range tags {
			scope.SetTag(key, value)
		}

		if extra != nil {
			scope.SetContext("extra", extra)
		}

		if message != "" {
			scope.SetTag("message", message)
		}

		sentry.CaptureException(err)
	})
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

// LogInfo logs an info message to standard logger (not sent to Sentry by default)
func LogInfo(message string) {
	log.Printf("INFO: %s", message)
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
