package main

import (
	"context"
	_ "time/tzdata"

	log "github.com/sirupsen/logrus"
)

func main() {
	settings := MustLoadSettings()

	if settings.Debug {
		log.SetLevel(log.DebugLevel)
	}

	topic := NewTopic()
	ctx, cancel := context.WithCancel(context.Background())

	go Updater(ctx, settings.Timezone, topic)

	httpServer := CreateHTTPServer(settings.APIKeys, topic)
	if err := httpServer.ListenAndServe(); err != nil {
		cancel()
		log.Fatalf("main: %s", err)
	}
}
