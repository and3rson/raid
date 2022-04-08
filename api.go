package main

import (
	"encoding/json"
	"net/http"
	"time"

	_ "embed"

	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

//go:generate pandoc -f markdown -s assets/index.md -c assets/style.css --metadata "title=Air Raid Alert API (Ukraine)" --self-contained --highlight-style breezedark -o assets/index.html
//go:embed assets/index.html
var indexContent []byte

type StatesResponse struct {
	States     []State   `json:"states"`
	LastUpdate time.Time `json:"last_update"`
}

type PollResponse struct {
	State State `json:"state"`
}

func CreateWebRouter(apiKeys []string) *mux.Router {
	apiKeysMap := make(map[string]bool)
	for _, key := range apiKeys {
		apiKeysMap[key] = true
	}

	webMux := mux.NewRouter()
	apiMux := webMux.PathPrefix("/api").Subrouter()
	lmt := tollbooth.NewLimiter(10, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
	lmt.SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"})
	lmt.SetOverrideDefaultResponseWriter(true)
	lmt.SetOnLimitReached(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(429)
		enc := json.NewEncoder(w)
		_ = enc.Encode(map[string]string{"error": "Too many requests"})
	})
	lmt.SetHeader("X-API-Key", apiKeys)

	// First middleware will be applied last.
	apiMux.Use(func(h http.Handler) http.Handler {
		return tollbooth.LimitHandler(lmt, h)
	})
	apiMux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("x-api-key")
			if _, ok := apiKeysMap[key]; !ok {
				rw.Header().Add("Content-Type", "application/json")
				rw.WriteHeader(403)
				enc := json.NewEncoder(rw)
				_ = enc.Encode(map[string]string{"error": "Unknown or missing X-API-Key value"})
				return
			}
			next.ServeHTTP(rw, r)
		})
	})

	apiMux.HandleFunc("/states", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "application/json")
		rw.WriteHeader(200)
		enc := json.NewEncoder(rw)
		response := StatesResponse{
			States,
			LastUpdate,
		}
		_ = enc.Encode(response)
	})
	apiMux.HandleFunc("/states/live", func(rw http.ResponseWriter, r *http.Request) {
		log.Infof("api: subscribe to default topic")
		events := DefaultTopic.Subscribe()
		defer func() {
			log.Infof("api: unsubscribe from default topic")
			DefaultTopic.Unsubscribe(events)
		}()
		rw.Header().Set("Content-Type", "text/event-stream")
		sse := NewSSEEncoder(rw)
		if err := sse.Write("hello", nil); err != nil {
			log.Errorf("api: send SSE hello: %s", err)
		}
		for {
			select {
			case ev := <-events:
				if state, ok := ev.(*State); ok {
					if err := sse.Write("update", PollResponse{*state}); err != nil {
						log.Errorf("api: send SSE update: %s", err)
						return
					}
				} else {
					log.Errorf("api: cannot cast event payload to *State")
				}
			case <-time.After(5 * time.Second):
				if err := sse.Write("ping", nil); err != nil {
					log.Errorf("api: send SSE ping: %s", err)
					return
				}
			}
		}
	})
	webMux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(200)
		_, _ = rw.Write(indexContent)
	})
	return webMux
}

func CreateHTTPServer(apiKeys []string) *http.Server {
	return &http.Server{
		Addr:    "0.0.0.0:10101",
		Handler: CreateWebRouter(apiKeys),
	}
}
