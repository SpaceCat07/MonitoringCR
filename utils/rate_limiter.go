package utils

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ClientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clients = make(map[string]*ClientLimiter)
	mu      sync.Mutex
)

func GetLimiter(userID string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	if c, ok := clients[userID]; ok {
		c.lastSeen = time.Now()
		return c.limiter
	}

	// Allow higher bursts for UI activity without triggering 429.
	limiter := rate.NewLimiter(50, 200)
	clients[userID] = &ClientLimiter{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}

func CleanupClients() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		mu.Lock()
		for userID, c := range clients {
			if time.Since(c.lastSeen) > 5*time.Minute {
				delete(clients, userID)
			}
		}
		mu.Unlock()
	}
}
