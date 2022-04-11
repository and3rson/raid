package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	_ "embed"

	"github.com/didip/tollbooth/v6"
	"github.com/didip/tollbooth/v6/limiter"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

//go:generate make -C assets
//go:embed assets/index.html
var indexContent []byte
//go:embed assets/index.en.html
var indexEnContent []byte

type StatesResponse struct {
	States     []State   `json:"states"`
	LastUpdate time.Time `json:"last_update"`
}

type PollResponse struct {
	State State `json:"state"`
}

type APIServer struct {
	apiKeys    []string
	apiKeysMap map[string]bool
	polls      chan time.Time
	updates    chan *State
	lastUpdate time.Time
}

func NewAPIServer(apiKeys []string, polls chan time.Time, updates chan *State) *APIServer {
	apiKeysMap := make(map[string]bool)
	for _, key := range apiKeys {
		apiKeysMap[key] = true
	}

	return &APIServer{
		apiKeys:    apiKeys,
		apiKeysMap: apiKeysMap,
		polls:      polls,
		updates:    updates,
		lastUpdate: time.Time{},
	}
}

func (a *APIServer) CreateRouter(ctx context.Context) *mux.Router {
	listeners := make(map[chan *State]bool)

	go func() {
		for {
			select {
			case poll := <-a.polls:
				a.lastUpdate = poll
			case state := <-a.updates:
				for ch := range listeners {
					ch <- state
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	tooManyRequestsBody := []byte("{\"error\": \"Too many requests\"}")

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
	lmt.SetHeader("X-API-Key", a.apiKeys)

	// First middleware will be applied last.
	apiMux.Use(func(h http.Handler) http.Handler {
		return tollbooth.LimitHandler(lmt, h)
	})
	apiMux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("x-api-key")
			if _, ok := a.apiKeysMap[key]; !ok {
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
			a.lastUpdate,
		}
		_ = enc.Encode(response)
	})
	apiMux.HandleFunc("/states/live", func(rw http.ResponseWriter, r *http.Request) {
		log.Infof("api: subscribe to events")
		events := make(chan *State)
		listeners[events] = true
		defer func() {
			log.Infof("api: unsubscribe from events")
			delete(listeners, events)
		}()
		rw.Header().Set("Content-Type", "text/event-stream")
		sse := NewSSEEncoder(rw)
		if err := sse.Write("hello", nil); err != nil {
			log.Errorf("api: send SSE hello: %s", err)
		}
		for {
			select {
			case state := <-events:
				if err := sse.Write("update", PollResponse{*state}); err != nil {
					log.Errorf("api: send SSE update: %s", err)

					return
				}
			case <-time.After(15 * time.Second):
				if err := sse.Write("ping", nil); err != nil {
					log.Errorf("api: send SSE ping: %s", err)

					return
				}
			case <-ctx.Done():
				return
			}
		}
	})
	webMux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(200)
		_, _ = rw.Write(indexContent)
	})
	webMux.HandleFunc("/en", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(200)
		_, _ = rw.Write(indexEnContent)
	})

	return webMux
}

func (a *APIServer) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer wg.Done()
	wg.Add(1)

	server := &http.Server{
		Addr:    "0.0.0.0:10101",
		Handler: a.CreateRouter(ctx),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				errch <- fmt.Errorf("api: server stopped: %w", err)

				return
			}
		}
	}()

	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		log.Errorf("api: shutdown: %s", err)
	}
}
