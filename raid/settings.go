package raid

import (
	"os"
	"time"

	"github.com/caarlos0/env/v6"
	yaml "github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
)

type Settings struct {
	TelegramChannel string         `env:"TELEGRAM_CHANNEL" envDefault:"air_alert_ua" yaml:"telegram_channel"`
	TimezoneName    string         `env:"TZ" envDefault:"Europe/Kiev" yaml:"timezone_name"`
	Timezone        *time.Location ``
	APIKeys         []string       `env:"API_KEYS" envSeparator:"," envDefault:"" yaml:"api_keys"`
	Debug           bool           `env:"DEBUG" envDefault:"false" yaml:"debug"`
	Trace           bool           `env:"TRACE" envDefault:"false" yaml:"trace"`
	BacklogSize     int            `env:"BACKLOG_SIZE" envDefault:"200" yaml:"backlog_size"`
}

func MustLoadSettings() (settings Settings) {
	var err error

	settings.TimezoneName = "Europe/Kiev"
	settings.TelegramChannel = "air_alert_ua"

	if len(os.Args) > 1 {
		var f *os.File

		f, err = os.Open(os.Args[1])
		if err != nil {
			log.Fatalf("settings: open settings file: %s", err)
		}

		dec := yaml.NewDecoder(f)
		if err = dec.Decode(&settings); err != nil {
			log.Fatalf("settings: load settings from file: %s", err)
		}
	} else {
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
	}

	if settings.Timezone, err = time.LoadLocation(settings.TimezoneName); err != nil {
		log.Fatalf("settings: load timezone: %s", err)
	}

	if len(settings.APIKeys) == 0 {
		log.Fatal("settings: no API keys were loaded")
	}

	log.Infof("settings: load %d API keys", len(settings.APIKeys))
	log.Infof("settings: %v", settings)

	return
}
