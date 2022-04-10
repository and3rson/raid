package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	_ "time/tzdata"

	log "github.com/sirupsen/logrus"
)

type Status struct {
	LastUpdate time.Time
}

func main() {
	settings := MustLoadSettings()

	if settings.Debug {
		log.SetLevel(log.DebugLevel)
	}

	topic := NewTopic()
	ctx, cancel := context.WithCancel(context.Background())

	wg := &sync.WaitGroup{}
	sharedStatus := &Status{}

	go Updater(ctx, wg, settings.Timezone, topic, sharedStatus)
	go RunHTTPServer(ctx, wg, settings.APIKeys, topic, sharedStatus)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Warnf("main: receive %v", <-c)
	cancel()
	log.Warnf("main: waiting for all children to terminate")
	wg.Wait()
	log.Warnf("main: finished")
}
