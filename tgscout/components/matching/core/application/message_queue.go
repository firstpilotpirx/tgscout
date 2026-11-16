package application

// MessageQueue is the driven port that carries collected messages to the matcher workers.
// It has no methods yet: the live-matching slice adds the ones it calls.
type MessageQueue interface{}
