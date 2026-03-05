package events

// EventType represents the kind of event being published
type EventType string

const (
	RepoChanged     EventType = "REPO_CHANGED"
	RepoUpdated     EventType = "REPO_UPDATED"
	CommandExecuted EventType = "COMMAND_EXECUTED"
	ErrorOccurred   EventType = "ERROR_OCCURRED"
)

// Subscriber is a function type that handles events
type Subscriber func(payload interface{})

// Bus defines the interface for an event bus system
type Bus interface {
	Publish(eventType EventType, payload interface{})
	Subscribe(eventType EventType, handler Subscriber)
}

// SimpleBus is a basic in-memory implementation of the Bus interface
type SimpleBus struct {
	subscribers map[EventType][]Subscriber
}

// NewBus creates a new simple event bus
func NewBus() Bus {
	return &SimpleBus{
		subscribers: make(map[EventType][]Subscriber),
	}
}

func (b *SimpleBus) Publish(eventType EventType, payload interface{}) {
	if handlers, found := b.subscribers[eventType]; found {
		for _, handler := range handlers {
			go handler(payload) // Handle asynchronously
		}
	}
}

func (b *SimpleBus) Subscribe(eventType EventType, handler Subscriber) {
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}
