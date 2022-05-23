package raid

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Updater struct {
	telegramChannel string
	timezone        *time.Location
	backlogSize     int
	updaterState    *UpdaterState
	Updates         *Topic[Update]
}

type UpdaterState struct {
	States        []State   `json:"states"`
	LastUpdate    time.Time `json:"last_update"`
	LastMessageID int64     `json:"last_message_id"`
}

type State struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	NameEn  string     `json:"name_en"`
	Alert   bool       `json:"alert"`
	Changed *time.Time `json:"changed"`
}

type Update struct {
	IsFresh bool
	State   State
}

func NewUpdater(telegramChannel string, timezone *time.Location, backlogSize int, updaterState *UpdaterState) *Updater {
	if len(updaterState.States) == 0 {
		updaterState.States = []State{
			{1, "Вінницька область", "Vinnytsia oblast", false, nil},
			{2, "Волинська область", "Volyn oblast", false, nil},
			{3, "Дніпропетровська область", "Dnipropetrovsk oblast", false, nil},
			{4, "Донецька область", "Donetsk oblast", false, nil},
			{5, "Житомирська область", "Zhytomyr oblast", false, nil},
			{6, "Закарпатська область", "Zakarpattia oblast", false, nil},
			{7, "Запорізька область", "Zaporizhzhia oblast", false, nil},
			{8, "Івано-Франківська область", "Ivano-Frankivsk oblast", false, nil},
			{9, "Київська область", "Kyiv oblast", false, nil},
			{10, "Кіровоградська область", "Kirovohrad oblast", false, nil},
			{11, "Луганська область", "Luhansk oblast", false, nil},
			{12, "Львівська область", "Lviv oblast", false, nil},
			{13, "Миколаївська область", "Mykolaiv oblast", false, nil},
			{14, "Одеська область", "Odesa oblast", false, nil},
			{15, "Полтавська область", "Poltava oblast", false, nil},
			{16, "Рівненська область", "Rivne oblast", false, nil},
			{17, "Сумська область", "Sumy oblast", false, nil},
			{18, "Тернопільська область", "Ternopil oblast", false, nil},
			{19, "Харківська область", "Kharkiv oblast", false, nil},
			{20, "Херсонська область", "Kherson oblast", false, nil},
			{21, "Хмельницька область", "Khmelnytskyi oblast", false, nil},
			{22, "Черкаська область", "Cherkasy oblast", false, nil},
			{23, "Чернівецька область", "Chernivtsi oblast", false, nil},
			{24, "Чернігівська область", "Chernihiv oblast", false, nil},
			{25, "м. Київ", "Kyiv", false, nil},
		}
	}

	return &Updater{
		telegramChannel,
		timezone,
		backlogSize,
		updaterState,
		NewTopic[Update](),
	}
}

func (u *Updater) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer log.Debug("updater: exit")

	defer wg.Done()
	wg.Add(1)

	cc := NewChannelClient(u.telegramChannel)

	var wait <-chan time.Time

	if u.updaterState.LastMessageID == 0 {
		log.Infof("updater: no previous ID, will fetch backlog")

		messages, err := cc.FetchLast(ctx, u.backlogSize)
		if err != nil {
			errch <- fmt.Errorf("updater: fetch initial batch: %w", err)

			return
		}

		log.Infof("updater: fetch %d last messages", len(messages))
		u.updaterState.LastMessageID = messages[len(messages)-1].ID

		u.ProcessMessages(ctx, messages, false)

		u.updaterState.LastUpdate = time.Now()

		wait = time.After(2 * time.Second)
	} else {
		log.Infof("updater: continue from ID %d", u.updaterState.LastMessageID)

		wait = time.After(0)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-wait:
		}

		messages, err := cc.FetchNewer(ctx, u.updaterState.LastMessageID)
		if err != nil {
			log.Error(err)

			wait = time.After(10 * time.Second)

			continue
		}

		u.updaterState.LastUpdate = time.Now()

		if len(messages) > 0 {
			log.Infof("updater: fetch %d new messages", len(messages))
			u.updaterState.LastMessageID = messages[len(messages)-1].ID
			u.ProcessMessages(ctx, messages, true)

			wait = time.After(0)
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

		if len(msg.Text) < 2 {
			log.Debugf("updater: not enough text in message: %v", msg.Text)

			continue
		}

		sentence := msg.Text[1]

		switch {
		case strings.Contains(sentence, "Повітряна тривога"):
			on = true
		case strings.Contains(sentence, "Відбій"):
			on = false
		default:
			log.Errorf("updater: don't know how to parse \"%s\"", sentence)
		}

		for index, other := range u.updaterState.States {
			if strings.Contains(sentence, other.Name) {
				state = &u.updaterState.States[index]
			}
		}

		if state == nil {
			log.Debugf("updater: no known states found in \"%s\"", sentence)
		} else {
			t := msg.Date.In(u.timezone)
			state.Changed = &t
			state.Alert = on
			log.Debugf("updater: new state: %s (id=%d) -> %v", state.Name, state.ID, on)
			u.Updates.Broadcast(Update{
				IsFresh: isFresh,
				State:   *state,
			})
		}
	}
}

func (s *UpdaterState) FindState(id int) *State {
	for i, state := range s.States {
		if state.ID == id {
			return &s.States[i]
		}
	}

	return nil
}
