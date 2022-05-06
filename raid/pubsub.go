package raid

import (
	"sync"
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

func (t *Topic[T]) Broadcast(payload T) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for ch, filter := range t.channels {
		if filter(payload) {
			ch <- payload
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
	t.mutex.Lock()
	defer t.mutex.Unlock()

	delete(t.channels, ch)
	delete(t.subscriberNames, ch)
	close(ch)
}
