package events

// Event represents a game event produced by command execution.
type Event struct {
	Name string
}

// New creates a new Event with the provided name.
func New(name string) Event {
	return Event{Name: name}
}
