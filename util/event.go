package util

type EventType int

const (
	MESS = iota
	TIMEOUT
	ERROR
)

type Event struct {
	Type EventType
	Data interface{}
}
