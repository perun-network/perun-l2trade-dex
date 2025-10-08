package message

import "sync"

type messageHandlerMap struct {
	m map[uint64]chan Message
	sync.RWMutex
}

func newMessageHandlerMap() *messageHandlerMap {
	return &messageHandlerMap{
		m:       make(map[uint64]chan Message),
		RWMutex: sync.RWMutex{},
	}
}

func (m *messageHandlerMap) set(id uint64, h chan Message) {
	m.Lock()
	defer m.Unlock()
	m.m[id] = h
}

func (m *messageHandlerMap) get(id uint64) (h chan Message, ok bool) {
	m.RLock()
	defer m.RUnlock()
	h, ok = m.m[id]
	return
}

func (m *messageHandlerMap) delete(id uint64) {
	m.Lock()
	defer m.Unlock()
	delete(m.m, id)
}
