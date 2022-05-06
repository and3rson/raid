package raid

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
		return fmt.Errorf("sse: encode event data: %w", err)
	}

	if _, err := e.writer.Write([]byte(fmt.Sprintf("event: %s\r\ndata: %s\r\n\r\n", event, encoded))); err != nil {
		return fmt.Errorf("sse: write event data: %w", err)
	}

	e.writer.(http.Flusher).Flush()

	return nil
}
