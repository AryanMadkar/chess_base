package services

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

type SocketManager struct {
	rooms map[string]map[*websocket.Conn]struct{}
	mu    sync.Mutex
}

var Manager = SocketManager{
	rooms: make(map[string]map[*websocket.Conn]struct{}),
}

func (m *SocketManager) Join(room string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rooms[room]; !ok {
		m.rooms[room] = make(map[*websocket.Conn]struct{})
	}
	m.rooms[room][conn] = struct{}{}
}

func (m *SocketManager) LeaveAll(conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for room, clients := range m.rooms {
		if _, ok := clients[conn]; ok {
			delete(clients, conn)
		}
		if len(clients) == 0 {
			delete(m.rooms, room)
		}
	}
}

func (m *SocketManager) Broadcast(room string, msg []byte) {
	m.mu.Lock()
	clientsMap, ok := m.rooms[room]
	if !ok {
		m.mu.Unlock()
		return
	}
	clients := make([]*websocket.Conn, 0, len(clientsMap))
	for conn := range clientsMap {
		clients = append(clients, conn)
	}
	m.mu.Unlock()

	for _, conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			m.mu.Lock()
			if roomClients, exists := m.rooms[room]; exists {
				delete(roomClients, conn)
				if len(roomClients) == 0 {
					delete(m.rooms, room)
				}
			}
			m.mu.Unlock()
		}
	}
}
