package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	_ "embed"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

//go:generate make -C assets
//go:embed assets/index.html
var indexContent []byte

//go:embed static/*
var staticFS embed.FS

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
	port              uint16
	apiKeys           []string
	apiKeysMap        map[string]bool
	updaterState      *UpdaterState
	updates           *Topic[*State]
	addrRateLimiter   throttled.RateLimiter
	apiKeyRateLimiter throttled.RateLimiter
	staticDirFS       fs.FS
}

func CreateRateLimiter(perSec int, burst int) throttled.RateLimiter {
	store, err := memstore.New(16384)
	if err != nil {
		log.Fatalf("api: create memstore: %s", err)
	}

	rateLimiter, err := throttled.NewGCRARateLimiter(store, throttled.RateQuota{
		MaxRate:  throttled.PerSec(perSec),
		MaxBurst: burst,
	})
	if err != nil {
		log.Fatalf("api: create rate limiter: %s", err)
	}

	return rateLimiter
}

func NewAPIServer(port uint16, apiKeys []string, updaterState *UpdaterState, updates *Topic[*State]) *APIServer {
	apiKeysMap := make(map[string]bool)
	for _, key := range apiKeys {
		apiKeysMap[key] = true
	}

	staticDirFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("api: create sub fs: %v", err)
	}

	return &APIServer{
		port:              port,
		apiKeys:           apiKeys,
		apiKeysMap:        apiKeysMap,
		updaterState:      updaterState,
		updates:           updates,
		addrRateLimiter:   CreateRateLimiter(10, 10),
		apiKeyRateLimiter: CreateRateLimiter(100, 100),
		staticDirFS:       staticDirFS,
	}
}

func (a *APIServer) CreateRouter(ctx context.Context) *mux.Router {
	webMux := mux.NewRouter()
	apiMux := webMux.PathPrefix("/api").Subrouter()
	httpAPIKeyRateLimiter := throttled.HTTPRateLimiter{
		DeniedHandler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(429)
			enc := json.NewEncoder(rw)
			_ = enc.Encode(map[string]string{"error": "Too many requests using your API key"})
		}),
		RateLimiter: a.apiKeyRateLimiter,
		VaryBy: &throttled.VaryBy{
			Headers: []string{"X-API-Key"},
		},
	}
	httpAddrRateLimiter := throttled.HTTPRateLimiter{
		DeniedHandler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(429)
			enc := json.NewEncoder(rw)
			_ = enc.Encode(map[string]string{"error": "Too many requests from your address"})
		}),
		RateLimiter: a.addrRateLimiter,
		VaryBy: &throttled.VaryBy{
			RemoteAddr: true,
		},
	}

	apiMux.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"X-API-Key", "Content-Type", "Cache-Control"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"*"}),
	))
	apiMux.Use(httpAPIKeyRateLimiter.RateLimit)
	apiMux.Use(httpAddrRateLimiter.RateLimit)
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
			a.updaterState.States,
			a.updaterState.LastUpdate,
		}
		_ = enc.Encode(response)
	})
	apiMux.HandleFunc("/states/live", func(rw http.ResponseWriter, r *http.Request) {
		log.Infof("api: subscribe to events")
		events := a.updates.Subscribe()
		defer func() {
			log.Infof("api: unsubscribe from events")
			a.updates.Unsubscribe(events)
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
	webMux.HandleFunc("/en", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(200)
		_, _ = rw.Write(indexEnContent)
	})
	webMux.PathPrefix("/").Handler(http.FileServer(http.FS(a.staticDirFS)))

	return webMux
}

func (a *APIServer) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer wg.Done()
	wg.Add(1)

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", a.port),
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
