package services

import (
	"chess/models"
	"sync"
)

type Matchmaking struct {
	queue   []string
	inQueue map[string]bool
	mu      sync.Mutex
}

var Matchmaker = Matchmaking{
	queue:   []string{},
	inQueue: map[string]bool{},
}

func (m *Matchmaking) Join(playerId string) (string, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.inQueue[playerId] {
		return "", "", false
	}

	for len(m.queue) > 0 {
		opponent := m.queue[0]
		m.queue = m.queue[1:]
		delete(m.inQueue, opponent)
		if opponent == "" || opponent == playerId {
			continue
		}
		return opponent, playerId, true
	}

	m.queue = append(m.queue, playerId)
	m.inQueue[playerId] = true
	return "", "", false
}

func (m *Matchmaking) QueueLength() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.queue)
}

func StartMatch(p1, p2 string) (*models.Game, error) {
	game, err := CreateGame()
	if err != nil {
		return nil, err
	}
	_, _, err = JoinGame(game.GameID, p1)
	if err != nil {
		return nil, err
	}
	finalGame, _, err := JoinGame(game.GameID, p2)
	if err != nil {
		return nil, err
	}
	return finalGame, nil
}
