package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
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
	// Проверяем, есть ли уже trace_id в заголовках
	traceID := req.Header.Get(t.headerName)

	// Если нет, генерируем новый
	if traceID == "" {
		traceID = generateTraceID()
		req.Header.Set(t.headerName, traceID)
	}

	// Добавляем trace_id в response headers
	rw.Header().Set(t.headerName, traceID)

	// Добавляем trace_id в контекст для использования в приложении
	ctx := context.WithValue(req.Context(), "trace_id", traceID)
	req = req.WithContext(ctx)

	// Структурированное логирование
	fmt.Printf(`{"trace_id":"%s","method":"%s","path":"%s","remote_addr":"%s","user_agent":"%s","timestamp":"%s"}`,
		traceID, req.Method, req.URL.Path, req.RemoteAddr,
		req.Header.Get("User-Agent"), req.Header.Get("X-Forwarded-For"))
	fmt.Println()

	// Передаем управление следующему middleware
	t.next.ServeHTTP(rw, req)
}

// generateTraceID генерирует уникальный trace ID
func generateTraceID() string {
	// Используем crypto/rand для безопасной генерации
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback - используем timestamp если crypto/rand недоступен
		return fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
