package main

import (
	"os"
	"time"

	"github.com/caarlos0/env/v6"
	yaml "github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
)

type Settings struct {
	TimezoneName string `env:"TZ" envDefault:"Europe/Kiev"`
	Timezone     *time.Location
	APIKeys      []string `env:"API_KEYS" envSeparator:"," envDefault:""`
	APIKeysFile  string   `env:"API_KEYS_FILE" envDefault:""`
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

	if len(settings.APIKeys) > 0 && settings.APIKeysFile != "" {
		log.Fatalf("settings: cannot provide both API_KEYS and API_KEYS_FILE, choose one")
	}

	var keysData struct {
		APIKeys []string `yaml:"keys"`
	}

	if settings.APIKeysFile != "" {
		var f *os.File

		f, err = os.Open(settings.APIKeysFile)
		if err != nil {
			log.Fatalf("settings: open API keys file: %s", err)
		}

		dec := yaml.NewDecoder(f)
		if err = dec.Decode(&keysData); err != nil {
			log.Fatalf("settings: load API keys from file: %s", err)
		}

		settings.APIKeys = keysData.APIKeys
	}

	if len(settings.APIKeys) == 0 {
		log.Fatal("settings: no API keys were loaded")
	}

	log.Infof("settings: load %d API keys", len(settings.APIKeys))

	return
}
