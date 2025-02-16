package util

type EventType int

const (
	MESSAGE = iota
	TIMEOUT
	ERROR
	TERMINATED
)

type Event struct {
	Type EventType
	Data interface{}
}
