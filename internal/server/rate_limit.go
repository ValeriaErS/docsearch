package server

import (
    "sync"
    "time"
)

type RateLimiter struct { //  ограничивает количество запросов от одного пользователя
    mu       sync.Mutex
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *RateLimiter) Allow(key string) bool {  //  проверяет можно ли выполнить запрос
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()

    if requests, ok := rl.requests[key]; ok {   // очищаю старые запросы
        var valid []time.Time
        for _, t := range requests {
            if now.Sub(t) < rl.window {
                valid = append(valid, t)
            }
        }
        rl.requests[key] = valid
    }

    if len(rl.requests[key]) >= rl.limit {
        return false
    }

    rl.requests[key] = append(rl.requests[key], now)
    return true
}