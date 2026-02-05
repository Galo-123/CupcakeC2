package hub

import (
	"sync"
)

type WsPacket struct {
	MsgType string `json:"type"`    // "TERM" | "JSON_DATA"
	Content string `json:"content"` // The actual payload string
	TaskID  string `json:"task_id,omitempty"`
}

type TaskHub struct {
	subscribers map[string][]chan WsPacket
	mu          sync.Mutex
}

func NewTaskHub() *TaskHub {
	return &TaskHub{
		subscribers: make(map[string][]chan WsPacket),
	}
}

func (h *TaskHub) Broadcast(taskID string, packet WsPacket) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.subscribers[taskID]; ok {
		for _, ch := range subs {
			select {
			case ch <- packet:
			default:
			}
		}
	}
}

func (h *TaskHub) Subscribe(taskID string) chan WsPacket {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan WsPacket, 100)
	h.subscribers[taskID] = append(h.subscribers[taskID], ch)
	return ch
}

func (h *TaskHub) Unsubscribe(taskID string, ch chan WsPacket) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.subscribers[taskID]; ok {
		for i, s := range subs {
			if s == ch {
				h.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}
}

var BuildHub = NewTaskHub()
