package main

type Topic[T interface{}] struct {
	channels map[chan T]bool
}

func NewTopic[T interface{}]() *Topic[T] {
	return &Topic[T]{
		make(map[chan T]bool),
	}
}

func (t *Topic[T]) Broadcast(payload T) {
	for ch := range t.channels {
		ch <- payload
	}
}

func (t *Topic[T]) Subscribe() chan T {
	ch := make(chan T)
	t.channels[ch] = true

	return ch
}

func (t *Topic[T]) Unsubscribe(ch chan T) {	
	delete(t.channels, ch)
}
