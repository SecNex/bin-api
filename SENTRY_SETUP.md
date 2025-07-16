# 📊 Sentry Performance-Monitoring Einrichtung

## 🚀 Schnellstart

### 1. Umgebungsvariablen setzen

```bash
# Erforderlich: Ihr Sentry DSN
export SENTRY_DSN="https://your-public-key@your-org.ingest.sentry.io/your-project-id"

# Optional: Weitere Konfiguration
export SENTRY_ENVIRONMENT="development"
export SENTRY_RELEASE="1.0.0"
export SENTRY_SAMPLE_RATE="1.0"
export SENTRY_TRACES_SAMPLE_RATE="1.0"
export SENTRY_DEBUG="true"
```

### 2. Server starten

```bash
go run main.go
```

## 📈 Was wird gesendet?

### Nur Server-Fehler (Errors only)

- 🔴 **Server-Fehler** (HTTP 500+) als Errors
- 💥 **Panics** mit vollständigen Stack-Traces
- ❌ **Keine Performance-Daten** (200-499)
- ❌ **Keine Warnings** (4xx, langsame Requests)
- ❌ **Keine Info-Events**

### Lokales Logging (nicht an Sentry)

- 📊 **HTTP-Requests** werden lokal geloggt (Konsole/Logs)
- ⏱️ **Performance-Metriken** nur lokal verfügbar
- 📝 **Info-Messages** nur in lokalen Logs

### Verfügbare Logging-Funktionen (nur lokal)

```go
// Error-Logging (wird an Sentry gesendet)
logger.LogError(err, "Database connection failed",
    map[string]string{"component": "database"},
    map[string]interface{}{"retry_count": 3})

// Warning-Logging (nur lokal, nicht an Sentry)
logger.LogWarning("High memory usage detected",
    map[string]string{"service": "api"},
    map[string]interface{}{"memory_mb": 512})

// Performance-Messung (nur lokal, nicht an Sentry)
logger.LogPerformance("database_query", 150*time.Millisecond,
    map[string]string{"table": "users"},
    map[string]interface{}{"rows": 100})

// Metriken (nur lokal, nicht an Sentry)
logger.LogMetric("memory_usage", 512.5, "MB",
    map[string]string{"service": "api"})
```

## 🔍 Testing

Der Server initialisiert Sentry beim Start:

```bash
🔧 Initializing Sentry...
✅ Sentry initialized successfully
🐛 Sentry debug mode enabled
🚀 Starting server on localhost:8081
📊 Server ready - Only server errors (500+) will be sent to Sentry
```

**Keine automatischen Test-Events** - nur echte Server-Fehler werden gesendet.

## 📊 Sentry Dashboard - Was zu erwarten ist

### Sentry Events (nur Errors):

- `issue_type: server_error` - Server-Fehler (HTTP 500+)
- `http.status_code: 500+` - Fehler-Status-Codes
- `http.method` - HTTP-Methode (GET, POST, etc.)
- `http.path` - Request-Pfad

### Events-Typen:

- **Messages** mit Level "Error" nur für Server-Fehler (500+)
- **Exceptions** für Panics mit Stack-Traces

### Filter-Beispiele:

```
level:error                                        # Alle Errors
issue_type:server_error                           # Nur Server-Fehler
http.status_code:500                              # Nur HTTP 500 Fehler
http.path:/api/*                                  # Nur API-Fehler
```

### Context-Daten:

- `performance` - Timing und Performance-Metriken
- `database` - Datenbank-Query-Details
- `api_call` - Externe API-Aufruf-Details
- `metric` - Custom-Metrik-Werte

## 🐛 Troubleshooting

### Keine Errors in Sentry?

1. **DSN prüfen**:

   ```bash
   echo $SENTRY_DSN
   ```

2. **Debug-Modus aktivieren**:

   ```bash
   export SENTRY_DEBUG=true
   ```

3. **500-Fehler simulieren**:

   ```bash
   # Einen 500-Fehler provozieren (Server-interne Fehler)
   curl http://localhost:8081/cause-error
   # Oder mit ungültigen Daten (JSON-Parsing-Fehler)
   curl -X POST http://localhost:8081/ -d 'invalid-json'
   ```

4. **Logs überprüfen**:
   - Achten Sie auf "Sending error event to Sentry" Nachrichten
   - Nur bei HTTP 500+ oder Panics werden Events gesendet

### Häufige Probleme:

- **"SENTRY_DSN not set"** → Umgebungsvariable setzen
- **Keine Events in Sentry** → Nur 500+ Fehler werden gesendet
- **Normale Requests (200-499)** → Werden NICHT an Sentry gesendet

## 📝 Beispiel .env Datei

```bash
# .env
SENTRY_DSN=https://your-key@your-org.ingest.sentry.io/your-project
SENTRY_ENVIRONMENT=development
SENTRY_SAMPLE_RATE=1.0
SENTRY_TRACES_SAMPLE_RATE=1.0
SENTRY_DEBUG=true
```

Laden Sie die .env Datei mit:

```bash
source .env
go run main.go
```
