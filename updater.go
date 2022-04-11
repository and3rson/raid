package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Updater struct {
	timezone    *time.Location
	backlogSize int
	Polls       chan time.Time
	Updates     chan *State
}

func NewUpdater(timezone *time.Location, backlogSize int) *Updater {
	return &Updater{
		timezone,
		backlogSize,
		make(chan time.Time),
		make(chan *State),
	}
}

func (u *Updater) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer wg.Done()
	wg.Add(1)

	cc := NewChannelClient("air_alert_ua")

	messages, err := cc.FetchLast(ctx, u.backlogSize)
	if err != nil {
		errch <- fmt.Errorf("updater: fetch initial batch: %w", err)

		return
	}

	log.Infof("updater: fetch %d last messages", len(messages))
	newestID := messages[len(messages)-1].ID

	u.ProcessMessages(ctx, messages, false)

	select {
	case u.Polls <- time.Now():
	case <-ctx.Done():
		return
	}

	wait := time.After(2 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-wait:
		}

		messages, err = cc.FetchNewer(ctx, newestID)
		if err != nil {
			log.Error(err)

			wait = time.After(10 * time.Second)

			continue
		}

		select {
		case u.Polls <- time.Now():
		case <-ctx.Done():
			return
		}

		if len(messages) > 0 {
			log.Infof("updater: fetch %d new messages", len(messages))
			newestID = messages[len(messages)-1].ID
			u.ProcessMessages(ctx, messages, true)
		} else {
			wait = time.After(2 * time.Second)
		}
	}
}

func (u *Updater) ProcessMessages(ctx context.Context, messages []Message, isFresh bool) {
	for _, msg := range messages {
		var (
			on    bool
			state *State
		)

		sentence := msg.Text[1]

		switch {
		case strings.Contains(sentence, "Повітряна тривога"):
			on = true
		case strings.Contains(sentence, "Відбій"):
			on = false
		default:
			log.Errorf("updater: don't know how to parse \"%s\"", sentence)
		}

		for index, other := range States {
			if strings.Contains(sentence, other.Name) {
				state = &States[index]
			}
		}

		if state == nil {
			log.Debugf("updater: no known states found in \"%s\"", sentence)
		} else {
			t := msg.Date.In(u.timezone)
			state.Changed = &t
			state.Alert = on
			log.Debugf("updater: new state: %s (id=%d) -> %v", state.Name, state.ID, on)
			if isFresh {
				select {
				case u.Updates <- state:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
