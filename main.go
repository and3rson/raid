package main

import (
	_ "time/tzdata"

	log "github.com/sirupsen/logrus"
)

// var Timezone *time.Location

func main() {
	settings := MustLoadSettings()
	// if err = os.Setenv("TZ", TimezoneName); err != nil {
	// 	log.Fatalf("main: set TZ: %s", err)
	// }
	// if Timezone, err = time.LoadLocation(TimezoneName); err != nil {
	// 	log.Fatalf("main: load TZ: %s", err)
	// }

	go Updater(settings.Timezone)

	httpServer := CreateHTTPServer(settings.APIKeys)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("main: %s", err)
	}
}
