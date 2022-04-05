package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type SSEEncoder struct {
	writer http.ResponseWriter
}

func NewSSEEncoder(w http.ResponseWriter) *SSEEncoder {
	return &SSEEncoder{w}
}

func (e *SSEEncoder) Write(event string, data interface{}) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := e.writer.Write([]byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, encoded))); err != nil {
		return err
	}
	e.writer.(http.Flusher).Flush()
	return nil
}
