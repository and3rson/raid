package main

type FilterFunc[T interface{}] func(T) bool

// func FilterAll[T interface{}](*T) bool {
// 	return true
// }

type Topic[T interface{}] struct {
	channels map[chan T]FilterFunc[T]
}

func NewTopic[T interface{}]() *Topic[T] {
	return &Topic[T]{
		make(map[chan T]FilterFunc[T], 32),
	}
}

func (t *Topic[T]) Broadcast(payload T) {
	for ch, filter := range t.channels {
		if filter(payload) {
			ch <- payload
		}
	}
}

func (t *Topic[T]) Subscribe(filter func(T) bool) chan T {
	ch := make(chan T)
	t.channels[ch] = filter

	return ch
}

func (t *Topic[T]) Unsubscribe(ch chan T) {
	delete(t.channels, ch)
}
