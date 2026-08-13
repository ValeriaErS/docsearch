package agent

import (
    "sync"
    "time"
)

type MemoryItem struct {   // один элемент памяти
    Query     string    `json:"query"`
    Answer    string    `json:"answer"`
    Sources   []string  `json:"sources"`
    Timestamp time.Time `json:"timestamp"`
    Step      int       `json:"step"`
}

type Memory struct {   // хранит историю выполнения агента
    mu       sync.RWMutex
    items    []MemoryItem
    maxSize  int
}

func NewMemory(maxSize int) *Memory {
    if maxSize <= 0 {
        maxSize = 10
    }
    return &Memory{
        items:   []MemoryItem{},
        maxSize: maxSize,
    }
}

func (m *Memory) Add(query, answer string, sources []string, step int) {  //  добавляет новый элемент в память
    m.mu.Lock()
    defer m.mu.Unlock()

    item := MemoryItem{
        Query:     query,
        Answer:    answer,
        Sources:   sources,
        Timestamp: time.Now(),
        Step:      step,
    }

    m.items = append(m.items, item)

    if len(m.items) > m.maxSize {
        m.items = m.items[len(m.items)-m.maxSize:]
    }
}

func (m *Memory) GetHistory() []MemoryItem {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.items
}

func (m *Memory) GetLast() *MemoryItem {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if len(m.items) == 0 {
        return nil
    }
    return &m.items[len(m.items)-1]
}

func (m *Memory) Clear() {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.items = []MemoryItem{}
}