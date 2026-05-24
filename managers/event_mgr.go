package managers

import (
	"sync"
	"time"
	"slices"
)

// credit to 0xataru/go_eventmanager: I read through the code before writing my own

type EventType int

const (
	KILL EventType = iota
)

type Event struct {
	Name EventType
	Timestamp time.Time
	Data any
}

type EventManager struct {
	subscribers map[EventType][]chan Event
	mtx sync.RWMutex
}

var (
	once     sync.Once
	instance *EventManager
)

func NewEventManager() *EventManager {
	once.Do(func() {
		instance = &EventManager{
			subscribers: make(map[EventType][]chan Event),
		}
	})
	return instance
}

func (event_mgr *EventManager) Subscribe(event EventType, channel chan Event) {
	event_mgr.mtx.Lock()
	event_mgr.subscribers[event] = append(event_mgr.subscribers[event], channel)
	event_mgr.mtx.Unlock()
}

func (event_mgr *EventManager) Unsubscribe(event EventType, channel chan Event) {	
	event_mgr.mtx.Lock()
	defer event_mgr.mtx.Unlock()
	subs, ok := event_mgr.subscribers[event]

	if ok {
		subs = slices.DeleteFunc(subs, func(tbd chan Event) bool {
			return tbd == channel
		})
	}

	if len(subs) == 0 {
		delete(event_mgr.subscribers, event)
		return
	}

	event_mgr.subscribers[event] = subs
}

func (event_mgr *EventManager) Send(event EventType, data any) {
	_, ok := event_mgr.subscribers[event]

	event_mgr.mtx.RLock()
	subs := append([]chan Event(nil), event_mgr.subscribers[event]...)
	event_mgr.mtx.RUnlock()

	eventStruct := Event {
		Name: event,
		Timestamp: time.Now(),
		Data: data,
	}

	if ok {
		for _, subscriber := range subs {
				select {
				case subscriber<-eventStruct:
				default:
				// Skip if the channel is busy
			}
		}
	}

}
