package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type SearchResultCache struct {  //  кэширует результаты поиска
	mu       sync.RWMutex
	cache    map[string]CachedResult
	filePath string
	ttl      time.Duration
}

type CachedResult struct {  //  хранит результат поиска
	Results   []map[string]interface{} `json:"results"`
	Timestamp time.Time                `json:"timestamp"`
	Hits      int                      `json:"hits"`
}

var (
	searchCache *SearchResultCache
	searchOnce  sync.Once
)

func GetSearchCache() *SearchResultCache {  //  возвращает синглтон кэша поиска
	searchOnce.Do(func() {
		searchCache = &SearchResultCache{
			cache:    make(map[string]CachedResult),
			filePath: "./.cache/search_results.json",
			ttl:      24 * time.Hour, 
		}
		searchCache.load()
	})
	return searchCache
}

func (c *SearchResultCache) Get(query string, userID string) ([]map[string]interface{}, bool) {  //  возвращает кэшированный результат поиска
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.hashKey(query + "|" + userID)
	val, ok := c.cache[key]
	if !ok {
		return nil, false
	}

	if time.Since(val.Timestamp) > c.ttl {
		delete(c.cache, key)
		return nil, false
	}

	fmt.Printf("Результат поиска найден в кэше (хитов: %d)\n", val.Hits)
	return val.Results, true
}

func (c *SearchResultCache) Set(query string, userID string, results []map[string]interface{}) {  //  сохраняет результат поиска в кэш
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.hashKey(query + "|" + userID)
	c.cache[key] = CachedResult{
		Results:   results,
		Timestamp: time.Now(),
		Hits:      1,
	}
	c.save()
}

func (c *SearchResultCache) IncrementHit(query string, userID string) {  //  увеличивает счетчик обращений
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.hashKey(query + "|" + userID)
	if val, ok := c.cache[key]; ok {
		val.Hits++
		c.cache[key] = val
	}
}

func (c *SearchResultCache) hashKey(text string) string {  //  создает хеш для запроса
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func (c *SearchResultCache) save() {
	data, err := json.Marshal(c.cache)
	if err != nil {
		return
	}
	os.WriteFile(c.filePath, data, 0644)
}

func (c *SearchResultCache) load() {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return
	}
	var cache map[string]CachedResult
	if err := json.Unmarshal(data, &cache); err != nil {
		return
	}
	c.cache = cache
	fmt.Printf("Загружено %d результатов поиска из кэша\n", len(c.cache))
}

func (c *SearchResultCache) Stats() int {  //  возвращает статистику кэша
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}