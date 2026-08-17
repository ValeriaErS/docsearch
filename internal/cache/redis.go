package cache

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "time"

    "github.com/go-redis/redis/v8"
)

type RedisCache struct {
    client *redis.Client
    ttl    time.Duration
}

type CachedAnswer struct {
    Answer    string    `json:"answer"`
    Sources   []string  `json:"sources"`
    Tokens    int       `json:"tokens"`
    Timestamp time.Time `json:"timestamp"`
}

func NewRedisCache(addr, password string, db int, ttl time.Duration) *RedisCache {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
    })

    return &RedisCache{
        client: client,
        ttl:    ttl,
    }
}

func (r *RedisCache) Ping(ctx context.Context) error {
    return r.client.Ping(ctx).Err()
}

func (r *RedisCache) key(parts ...string) string {
    result := "docsearch"
    for _, p := range parts {
        result += ":" + p
    }
    return result
}

func (r *RedisCache) hash(s string) string {
    h := sha256.Sum256([]byte(s))
    return hex.EncodeToString(h[:16])
}

func (r *RedisCache) GetSearch(ctx context.Context, query, userID string) ([]map[string]interface{}, bool) {
    key := r.key("search", userID, r.hash(query))
    data, err := r.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, false
    }
    if err != nil {
        return nil, false
    }

    var results []map[string]interface{}
    if err := json.Unmarshal([]byte(data), &results); err != nil {
        return nil, false
    }

    return results, true
}

func (r *RedisCache) SaveSearch(ctx context.Context, query, userID string, results []map[string]interface{}) error {
    key := r.key("search", userID, r.hash(query))
    data, err := json.Marshal(results)
    if err != nil {
        return err
    }

    return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *RedisCache) GetAnswer(ctx context.Context, question, userID string) (*CachedAnswer, bool) {
    key := r.key("answer", userID, r.hash(question))
    data, err := r.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, false
    }
    if err != nil {
        return nil, false
    }

    var answer CachedAnswer
    if err := json.Unmarshal([]byte(data), &answer); err != nil {
        return nil, false
    }

    return &answer, true
}

func (r *RedisCache) SaveAnswer(ctx context.Context, question, userID string, answer *CachedAnswer) error {
    key := r.key("answer", userID, r.hash(question))
    data, err := json.Marshal(answer)
    if err != nil {
        return err
    }

    return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *RedisCache) DeleteUserCache(ctx context.Context, userID string) error {
    pattern := r.key("*", userID, "*")
    iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()

    for iter.Next(ctx) {
        if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
            return err
        }
    }

    return iter.Err()
}

func (r *RedisCache) Close() error {
    return r.client.Close()
}