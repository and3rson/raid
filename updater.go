package main

import (
	"context"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func Updater(ctx context.Context, wg *sync.WaitGroup, timezone *time.Location, topic *Topic, sharedStatus *Status) {
	defer wg.Done()
	wg.Add(1)

	cc := NewChannelClient("air_alert_ua")

	messages, err := cc.FetchLast(ctx, 200)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("updater: fetch %d last messages", len(messages))
	newestID := messages[len(messages)-1].ID

	ProcessMessages(messages, timezone, topic, false)

	sharedStatus.LastUpdate = time.Now()

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

		sharedStatus.LastUpdate = time.Now()

		if len(messages) > 0 {
			log.Infof("updater: fetch %d new messages", len(messages))
			newestID = messages[len(messages)-1].ID
			ProcessMessages(messages, timezone, topic, true)
		} else {
			wait = time.After(2 * time.Second)
		}
	}
}

func ProcessMessages(messages []Message, timezone *time.Location, topic *Topic, isFresh bool) {
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
			t := msg.Date.In(timezone)
			state.Changed = &t
			state.Alert = on
			log.Debugf("updater: new state: %s (id=%d) -> %v", state.Name, state.ID, on)
			if isFresh {
				topic.Broadcast(state)
			}
		}
	}
}
