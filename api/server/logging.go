package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/photoview/photoview/api/graphql/auth"
	"github.com/wsxiaoys/terminal/color"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		statusWriter := newStatusResponseWriter(&w)
		next.ServeHTTP(statusWriter, r)

		elapsed := time.Since(start)
		date := time.Now().Format("2006/01/02 15:04:05")

		statusColor := getStatusColor(statusWriter.status)
		methodColor := getMethodColor(r.Method)

		user := auth.UserFromContext(r.Context())
		userText := "unauthenticated"
		if user != nil {
			userText = color.Sprintf("@ruser: %s", user.Username)
		}

		statusText := color.Sprintf("%s%s %s%d", methodColor, r.Method, statusColor, statusWriter.status)
		requestText := fmt.Sprintf("%s%s", r.Host, r.URL.Path)
		durationText := color.Sprintf("@c%s", elapsed)

		fmt.Printf("%s %s %s %s %s\n", date, statusText, requestText, durationText, userText)

	})
}

func getStatusColor(status int) string {
	switch {
	case status < 200:
		return color.Colorize("b")
	case status < 300:
		return color.Colorize("g")
	case status < 400:
		return color.Colorize("c")
	case status < 500:
		return color.Colorize("y")
	default:
		return color.Colorize("r")
	}
}

func getMethodColor(method string) string {
	switch {
	case method == http.MethodGet:
		return color.Colorize("b")
	case method == http.MethodPost:
		return color.Colorize("g")
	case method == http.MethodOptions:
		return color.Colorize("y")
	default:
		return color.Colorize("r")
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	status   int
	hijacker http.Hijacker
}

func newStatusResponseWriter(w *http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: *w,
		hijacker:       (*w).(http.Hijacker),
	}
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.hijacker == nil {
		return nil, nil, errors.New("http.Hijacker not implemented by underlying http.ResponseWriter")
	}
	return w.hijacker.Hijack()
}