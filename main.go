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

	updater := NewUpdater(settings.Timezone, settings.BacklogSize)
	apiServer := NewAPIServer(10101, settings.APIKeys, updater.Polls, updater.Updates)
	tcpServer := NewTCPServer(1024, settings.APIKeys, updater.Polls, updater.Updates)

	go updater.Run(ctx, wg, errch)
	go apiServer.Run(ctx, wg, errch)
	go tcpServer.Run(ctx, wg, errch)

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
	log.Warnf("main: finished")
}
