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

//go:generate pandoc -f markdown -s index.md -c style.css --metadata "title=Air Raid Alert API (Ukraine)" --self-contained -o index.html
//go:embed index.html
var indexContent []byte

type StatesResponse struct {
	States []State `json:"states"`
	LastUpdate time.Time `json:"last_update"`
}

type PollResponse struct {
	State State `json:"state"`
}

func CreateWebRouter() *mux.Router {
	webMux := mux.NewRouter()
	apiMux := webMux.PathPrefix("/api").Subrouter()
	lmt := tollbooth.NewLimiter(4, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
	lmt.SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"})
	lmt.SetOverrideDefaultResponseWriter(true)
	lmt.SetOnLimitReached(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(429)
		enc := json.NewEncoder(w)
		_ = enc.Encode(map[string]string{"error": "Too many requests"})
	})
	apiMux.Handle("/states", tollbooth.LimitFuncHandler(lmt, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		enc := json.NewEncoder(w)
		response := StatesResponse{
			States,
			LastUpdate,
		}
		_ = enc.Encode(response)
	}))
	apiMux.Handle("/states/live", tollbooth.LimitFuncHandler(lmt, func(w http.ResponseWriter, r *http.Request) {
		log.Infof("api: subscribe to default topic")
		events := DefaultTopic.Subscribe()
		defer func() {
			log.Infof("api: unsubscribe from default topic")
			DefaultTopic.Unsubscribe(events)
		}()
		w.Header().Set("Content-Type", "text/event-stream")
		sse := NewSSEEncoder(w)
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
	}))
	webMux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html; charset=utf-8")
		rw.WriteHeader(200)
		_, _ = rw.Write(indexContent)
	})
	return webMux
}

func CreateHTTPServer() *http.Server {
	return &http.Server{
		Addr:    "0.0.0.0:10101",
		Handler: CreateWebRouter(),
	}
}
