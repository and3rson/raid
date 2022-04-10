package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
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

func CreateWebRouter(apiKeys []string, topic *Topic, sharedStatus *Status) *mux.Router {
	apiKeysMap := make(map[string]bool)
	for _, key := range apiKeys {
		apiKeysMap[key] = true
	}

	var tooManyRequestsBody []byte

	tooManyRequestsBody, err := json.Marshal(map[string]string{"error": "Too many requests"})
	if err != nil {
		log.Fatalf("api: prepare marshalled tooManyRequestsBody: %s", err)
	}

	webMux := mux.NewRouter()
	apiMux := webMux.PathPrefix("/api").Subrouter()
	lmt := tollbooth.NewLimiter(10, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
	lmt.SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"})
	lmt.SetOverrideDefaultResponseWriter(true)
	lmt.SetOnLimitReached(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(429)
		_, _ = w.Write(tooManyRequestsBody)
		key := r.Header.Get("x-api-key")
		// addr := r.Header.Get("x-forwarder-for")
		// if addr == "" {
		// 	addr = r.RemoteAddr
		// }
		log.Warnf("api: throttle for key %s", key)
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
			sharedStatus.LastUpdate,
		}
		_ = enc.Encode(response)
	})
	apiMux.HandleFunc("/states/live", func(rw http.ResponseWriter, r *http.Request) {
		log.Infof("api: subscribe to default topic")
		events := topic.Subscribe()
		defer func() {
			log.Infof("api: unsubscribe from default topic")
			topic.Unsubscribe(events)
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
			case <-time.After(15 * time.Second):
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

func RunHTTPServer(ctx context.Context, wg *sync.WaitGroup, apiKeys []string, topic *Topic, sharedStatus *Status) {
	defer wg.Done()
	wg.Add(1)

	server := &http.Server{
		Addr:    "0.0.0.0:10101",
		Handler: CreateWebRouter(apiKeys, topic, sharedStatus),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				log.Fatalf("api: serve: %s", err)
			}
		}
	}()

	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("api: shutdown: %s", err)
	}
}
