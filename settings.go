package main

import (
	"time"

	"github.com/caarlos0/env/v6"
	log "github.com/sirupsen/logrus"
)

type Settings struct {
	TimezoneName string `env:"TZ" envDefault:"Europe/Kiev"`
	Timezone     *time.Location
	APIKeys      []string `env:"API_KEYS" envSeparator:","`
	Debug        bool     `env:"DEBUG" envDefault:"false"`
	BacklogSize  int      `env:"BACKLOG_SIZE" envDefault:"200"`
}

func MustLoadSettings() (settings Settings) {
	var err error

	opts := env.Options{
		RequiredIfNoDef: true,
		OnSet: func(tag string, value interface{}, isDefault bool) {
			if isDefault {
				log.Warnf("settings: using default value for env var %s: %s", tag, value)
			}
		},
	}

	if err = env.Parse(&settings, opts); err != nil {
		log.Fatalf("settings: load: %s", err)
	}

	if settings.Timezone, err = time.LoadLocation(settings.TimezoneName); err != nil {
		log.Fatalf("settings: load timezone: %s", err)
	}

	if len(settings.APIKeys) == 0 {
		log.Fatal("settings: no API_KEYS defined in environment")
	}

	log.Infof("settings: load %d keys from API_KEYS env var", len(settings.APIKeys))

	return
}
