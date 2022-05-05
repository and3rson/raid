package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	_ "time/tzdata"

	log "github.com/sirupsen/logrus"
)

func main() {
	settings := MustLoadSettings()

	if settings.Debug {
		log.SetLevel(log.DebugLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	errch := make(chan error, 3)

	updaterState := &UpdaterState{}

	persistence, err := NewPersistence(updaterState, "./data/app_state.json")
	if err != nil {
		log.Fatalf("main: create app state persistence: %v", err)
	}

	updater := NewUpdater(settings.Timezone, settings.BacklogSize, updaterState)
	mapGenerator := NewMapGenerator(updaterState, updater.Updates)
	apiServer := NewAPIServer(10101, settings.APIKeys, updaterState, updater.Updates, mapGenerator.MapData)
	tcpServer := NewTCPServer(1024, settings.APIKeys, updaterState, updater.Updates)

	go updater.Run(ctx, wg, errch)
	go apiServer.Run(ctx, wg, errch)
	go tcpServer.Run(ctx, wg, errch)
	go mapGenerator.Run(ctx, wg, errch)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case sig := <-c:
		log.Warnf("main: receive %v", sig)
	case err := <-errch:
		log.Warnf("main: child crashed: %v", err)
	}
	cancel()
	log.Warnf("main: waiting for all children to terminate")
	wg.Wait()
	log.Warnf("main: saving updater state")

	if err := persistence.Save(); err != nil {
		log.Fatalf("main: failed to save updater state: %v", err)
	}

	log.Warnf("main: finished")
}
