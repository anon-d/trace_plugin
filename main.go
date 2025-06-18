package trace_plugin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Config структура конфигурации плагина
type Config struct {
	HeaderName string `json:"headerName,omitempty"`
}

// CreateConfig создает конфигурацию по умолчанию
func CreateConfig() *Config {
	return &Config{
		HeaderName: "X-Trace-Id",
	}
}

// TraceIDPlugin основная структура плагина
type TraceIDPlugin struct {
	next       http.Handler
	name       string
	headerName string
}

// New создает новый экземпляр плагина
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.HeaderName == "" {
		config.HeaderName = "X-Trace-Id"
	}

	return &TraceIDPlugin{
		next:       next,
		name:       name,
		headerName: config.HeaderName,
	}, nil
}

// ServeHTTP обрабатывает HTTP запросы
func (t *TraceIDPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	startTime := time.Now()
	// Проверяем, есть ли уже trace_id в заголовках
	traceID := req.Header.Get(t.headerName)

	// Если нет, генерируем новый
	if traceID == "" {
		traceID = generateTraceID()
		req.Header.Set(t.headerName, traceID)
	}

	// Добавляем trace_id в response headers
	// rw.Header().Set(t.headerName, traceID)

	// Добавляем trace_id в контекст для использования в приложении
	ctx := context.WithValue(req.Context(), "trace_id", traceID)
	req = req.WithContext(ctx)

	logData := map[string]interface{}{
		"trace_id":        traceID,
		"method":          req.Method,
		"path":            req.URL.Path,
		"remote_addr":     req.RemoteAddr,
		"user_agent":      req.Header.Get("User-Agent"),
		"x_forwarded_for": req.Header.Get("X-Forwarded-For"),  // ip клиента
		"timestamp":       startTime.Format(time.RFC3339Nano), // Корректная временная метка
	}
	fmt.Printf("%s\n", formatJSON(logData))

	// Передаем управление следующему middleware
	t.next.ServeHTTP(rw, req)

	endTime := time.Now()
	logData["timestamp"] = endTime.Format(time.RFC3339Nano)
	logData["response_time"] = strconv.Itoa(int(endTime.Sub(startTime).Milliseconds())) + " ms"
	fmt.Printf("%s\n", formatJSON(logData))
}

func generateTraceID() string {
	// Используем crypto/rand для безопасной генерации
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback - используем timestamp если crypto/rand недоступен
		return fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func formatJSON(data map[string]interface{}) string {
	var s string
	s = "{"
	first := true
	for k, v := range data {
		if !first {
			s += ","
		}
		s += fmt.Sprintf(`"%s":"%v"`, k, v)
		first = false
	}
	s += "}"
	return s
}
