package main

// type Topic struct {
// 	channels map[chan Payload]bool
// }
// type Payload = interface{}

// func NewTopic() *Topic {
// 	return &Topic{
// 		make(map[chan Payload]bool),
// 	}
// }

// func (t *Topic) Broadcast(payload Payload) {
// 	for ch := range t.channels {
// 		ch <- payload
// 	}
// }

// func (t *Topic) Subscribe() chan Payload {
// 	ch := make(chan Payload)
// 	t.channels[ch] = true

// 	return ch
// }

// func (t *Topic) Unsubscribe(ch chan Payload) {	
// 	delete(t.channels, ch)
// }
