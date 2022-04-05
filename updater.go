package main

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var LastUpdate time.Time

func Updater() {
	cc := NewChannelClient("air_alert_ua")
	messages, err := cc.FetchLast(200)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("updater: fetched %d last messages", len(messages))
	newestID := messages[len(messages)-1].ID
	ProcessMessages(messages)
	LastUpdate = time.Now()
	for {
		messages, err = cc.FetchNewer(newestID)
		if err != nil {
			log.Error(err)
			continue
		}
		LastUpdate = time.Now()
		if len(messages) > 0 {
			log.Infof("updater: fetched %d new messages", len(messages))
			newestID = messages[len(messages)-1].ID
			ProcessMessages(messages)
		} else {
			<-time.After(5 * time.Second)
		}
	}
}

func ProcessMessages(messages []Message) {
	for _, msg := range messages {
		sentence := msg.Text[1]
		var on bool
		if strings.Contains(sentence, "Повітряна тривога") {
			on = true
		} else if strings.Contains(sentence, "Відбій") {
			on = false
		} else {
			log.Errorf("updater: don't know how to parse \"%s\"", sentence)
		}
		var state *State
		for index, other := range States {
			if strings.Contains(sentence, other.Name) {
				state = &States[index]
			}
		}
		if state == nil {
			log.Warnf("updater: no known states found in \"%s\"", sentence)
		} else {
			log.Infof("%s (%d) -> %v", state.Name, state.ID, on)
			t := msg.Date
			state.Changed = &t
			state.Alert = on
			DefaultTopic.Broadcast(state)
		}
	}
}
