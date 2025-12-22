package events

import "time"

// EventType describes the kind of event emitted by the game.
type EventType string

const EventTypeScrapSettled EventType = "ScrapSettled"

// ScrapSettledData is the payload for a scrap settlement event.
type ScrapSettledData struct {
	Minted uint64
	From   time.Time
	To     time.Time
}

// Event represents a game event produced by command execution.
type Event struct {
	ID        uint64
	At        time.Time
	CommandID string
	Type      EventType
	Data      any
}

// New constructs a new Event with the provided fields.
func New(id uint64, at time.Time, commandID string, eventType EventType, data any) Event {
	return Event{
		ID:        id,
		At:        at,
		CommandID: commandID,
		Type:      eventType,
		Data:      data,
	}
}
