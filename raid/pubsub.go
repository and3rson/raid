package raid

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

type FilterFunc[T interface{}] func(T) bool

func FilterAll[T interface{}](T) bool {
	return true
}

type Topic[T interface{}] struct {
	channels        map[chan T]FilterFunc[T]
	subscriberNames map[chan T]string
	mutex           sync.Mutex
}

func NewTopic[T interface{}]() *Topic[T] {
	return &Topic[T]{
		channels:        make(map[chan T]FilterFunc[T]),
		subscriberNames: make(map[chan T]string),
	}
}

func (t *Topic[T]) sendSafe(ch chan T, payload T) (ok bool) {
	// https://groups.google.com/g/golang-nuts/c/6bL3lXoC4Ek
	// TODO: When channel write is blocked, it's better to make Broadcast unlock mutex
	// to give subscribers time to unsubscribe.
	defer func() { recover() }()
	ch <- payload

	return true
}

func (t *Topic[T]) Broadcast(payload T) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for ch, filter := range t.channels {
		if filter(payload) {
			if len(ch) == cap(ch) {
				log.Warnf("pubsub: broadcast: channel %s is full, will block", t.subscriberNames[ch])
			}
			t.sendSafe(ch, payload)
		}
	}
}

func (t *Topic[T]) Subscribe(name string, filter func(T) bool) chan T {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	ch := make(chan T, 32)
	t.channels[ch] = filter
	t.subscriberNames[ch] = name

	return ch
}

func (t *Topic[T]) Unsubscribe(ch chan T) {
	// https://groups.google.com/g/golang-nuts/c/6bL3lXoC4Ek
	close(ch)

	t.mutex.Lock()
	defer t.mutex.Unlock()

	delete(t.channels, ch)
	delete(t.subscriberNames, ch)
}
