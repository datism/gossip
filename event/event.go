package event

type EventType int

const (
	RECV = iota
	SEND
	TIMEOUT
	ERROR
)

type Event struct {
	Type EventType
	Data interface{}
}
