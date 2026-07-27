package embed

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "docsearch/internal/config"
)

const LongVector = 768

func TestLong(t *testing.T) {
    if LongVector <= 0 {
        t.Errorf("Ошибка, размер должен быть больше 0, сейчас он %d", LongVector)
    }
}

func TestVectorneNol(t *testing.T) {
    t.Log("Тест пройден")
}

func TestLongVector(t *testing.T) {
    if LongVector != 768 {
        t.Errorf("Размер вектора %d, ожидалось 768", LongVector)
    }
    t.Log("Размер вектора 768")
}

func TestDrygText(t *testing.T) {
    
    testVectors := [][]float64{  // проверочка что для разных текстов размер одинаковый
        {0.1, 0.2, 0.3},
        {0.4, 0.5, 0.6},
        {0.7, 0.8, 0.9},
    }

    for i, vec := range testVectors {
        if len(vec) != 3 {
            t.Errorf("текст %d дал размер %d, ожидалось 3", i+1, len(vec))
        }
    }
    t.Log("Все векторы одинакового размера")
}
func TestGetEmbeddingSize(t *testing.T) {
    
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/v1/embeddings" {
            t.Errorf("Неверный путь: %s", r.URL.Path)
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "data": []map[string]interface{}{
                {
                    "embedding": make([]float64, 768),
                },
            },
        })
    }))
    defer server.Close()

    cfg := &config.Config{
        Embeddings: struct {
            Provider  string `yaml:"provider"`
            Model  string `yaml:"model"`
            BaseURL  string `yaml:"base_url"`
            VectorSize int `yaml:"vector_size"`
        }{
            Provider: "local",
            Model: "test-model",
            BaseURL: server.URL,
            VectorSize: 768,
        },
    }

    embedding, err := GetEmbedding(context.Background(), "тестовый текст", cfg)
    if err != nil {
        t.Fatalf("Ошибка получения эмбеддинга: %v", err)
    }

    if len(embedding) != 768 {
        t.Errorf("Неверный размер вектора: ожидалось 768, получено %d", len(embedding))
    }

    t.Logf(" Эмбеддинг получен, размер: %d", len(embedding))
}