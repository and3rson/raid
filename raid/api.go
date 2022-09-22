package raid

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "embed"

	"github.com/google/uuid"
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

type ShortState struct {
	ID    int  `json:"id"`
	Alert bool `json:"alert"`
}

type StatesShortResponse struct {
	States     []ShortState `json:"states"`
	LastUpdate time.Time    `json:"last_update"`
}

type StateResponse struct {
	*State     `json:"state"`
	LastUpdate time.Time `json:"last_update"`
}

type PollResponse struct {
	State          State     `json:"state"`
	NotificationID uuid.UUID `json:"notification_id"`
}

type APIServer struct {
	port              uint16
	apiKeys           []string
	apiKeysMap        map[string]bool
	updaterState      *UpdaterState
	updates           *Topic[Update]
	mapData           *MapData
	listRecordsFunc   func() ([]Record, error)
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

func NewAPIServer(
	port uint16, apiKeys []string, updaterState *UpdaterState, updates *Topic[Update], mapData *MapData,
	listRecordsFunc func() ([]Record, error),
) *APIServer {
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
		mapData:           mapData,
		listRecordsFunc:   listRecordsFunc,
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
			Custom: func(r *http.Request) string {
				realAddr := r.RemoteAddr
				forwardedAddr := r.Header.Get("X-Forwarded-For")
				if forwardedAddr != "" {
					realAddr = forwardedAddr

					ips := strings.Split(realAddr, " ")
					if len(ips) > 1 {
						realAddr = ips[0]
					}
				}

				return realAddr
			},
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

	statesHandleFunc := func(rw http.ResponseWriter, r *http.Request) {
		id := 0
		if idStr, ok := mux.Vars(r)["id"]; ok {
			id, _ = strconv.Atoi(idStr)
		}

		short := r.URL.Query().Has("short")

		rw.Header().Add("Content-Type", "application/json")
		rw.WriteHeader(200)
		enc := json.NewEncoder(rw)

		if id != 0 {
			for i, state := range a.updaterState.States {
				if state.ID == id {
					_ = enc.Encode(StateResponse{
						&a.updaterState.States[i],
						a.updaterState.LastUpdate,
					})

					return
				}
			}

			_ = enc.Encode(StateResponse{
				nil,
				a.updaterState.LastUpdate,
			})
		} else {
			if short {
				shortStates := []ShortState{}
				for _, state := range a.updaterState.States {
					shortStates = append(shortStates, ShortState{ID: state.ID, Alert: state.Alert})
				}
				_ = enc.Encode(StatesShortResponse{
					shortStates,
					a.updaterState.LastUpdate,
				})
			} else {
				_ = enc.Encode(StatesResponse{
					a.updaterState.States,
					a.updaterState.LastUpdate,
				})
			}
		}
	}
	apiMux.HandleFunc("/states", statesHandleFunc)
	apiMux.HandleFunc("/states/{id:[0-9]+}", statesHandleFunc)

	liveHandleFunc := func(rw http.ResponseWriter, r *http.Request) {
		id := 0
		if idStr, ok := mux.Vars(r)["id"]; ok {
			id, _ = strconv.Atoi(idStr)
		}

		if id == 0 {
			log.Info("api: subscribe to events")
		} else {
			log.Infof("api: subscribe to events for state %d", id)
		}

		events := a.updates.Subscribe("api-"+r.RemoteAddr, func(u Update) bool {
			return u.IsFresh && (id == 0 || id == u.State.ID)
		})
		defer func() {
			log.Infof("api: unsubscribe from events")
			a.updates.Unsubscribe(events)
		}()
		rw.Header().Set("Content-Type", "text/event-stream")
		rw.Header().Set("Cache-Control", "no-cache")
		sse := NewSSEEncoder(rw)

		if err := sse.Write("hello", nil); err != nil {
			log.Errorf("api: send SSE hello: %s", err)
		}

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}

				uuid1, err := uuid.NewUUID()
				if err != nil {
					return
				}

				if err := sse.Write("update", PollResponse{event.State, uuid1}); err != nil {
					log.Errorf("api: send SSE update: %s", err)

					return
				}
			case <-time.After(5 * time.Second):
				if err := sse.Write("ping", nil); err != nil {
					log.Errorf("api: send SSE ping: %s", err)

					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
	apiMux.HandleFunc("/states/live", liveHandleFunc)
	apiMux.HandleFunc("/states/live/{id:[0-9]+}", liveHandleFunc)

	const historyCooldown = 60 * time.Second

	historyCallsPerKey := make(map[string]time.Time)

	apiMux.HandleFunc("/history", func(rw http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-api-key")
		lastCall := historyCallsPerKey[key]
		if time.Since(lastCall) < historyCooldown {
			rw.Header().Add("Content-Type", "application/json")
			rw.WriteHeader(429)
			enc := json.NewEncoder(rw)
			_ = enc.Encode(map[string]string{
				"error": fmt.Sprintf("Please wait %d seconds", int(time.Until(lastCall.Add(historyCooldown)).Seconds())),
			})

			return
		}
		historyCallsPerKey[key] = time.Now()

		records, err := a.listRecordsFunc()
		if err != nil {
			log.Errorf("api: list records: %v", err)
			rw.Header().Add("Content-Type", "text/html; charset=utf-8")
			rw.WriteHeader(500)
			enc := json.NewEncoder(rw)
			_ = enc.Encode(map[string]string{"error": "Internal server error while fetching data from DB"})

			return
		}

		rw.Header().Add("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(200)
		enc := json.NewEncoder(rw)
		_ = enc.Encode(records)
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
	webMux.HandleFunc("/map.png", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", a.mapData.ContentType)
		rw.WriteHeader(200)
		_, _ = rw.Write(a.mapData.Bytes)
	})
	webMux.PathPrefix("/").Handler(http.FileServer(http.FS(a.staticDirFS)))

	return webMux
}

func (a *APIServer) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer log.Debug("api: exit")

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
