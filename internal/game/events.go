package game

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// broadcast sends an event to all connected clients
func (e *Engine) broadcast(event GameEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	ctx := context.Background()
	e.rdb.Publish(ctx, "game:events", data)

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, client := range e.clients {
		select {
		case client.Send <- data:
		default:
			// Client's send buffer is full, skip
		}
	}
}

// SubscribeEvents returns a pubsub subscription for game events
func (e *Engine) SubscribeEvents() *redis.PubSub {
	ctx := context.Background()
	return e.rdb.Subscribe(ctx, "game:events")
}