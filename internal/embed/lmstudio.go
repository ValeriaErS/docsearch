package embed

import (
	"bytes"
	"context"
	"docsearch/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

func GetEmbedding(ctx context.Context, text string, cfg *config.Config) ([]float64, error) { //отправка текста в LM с возвратом эмбеддинга
	if cfg.Embeddings.Provider == "mock" { //возврат фиктивного вектора при моке
		vectorSize := cfg.Embeddings.VectorSize
		if vectorSize <= 0 {
			vectorSize = 768
		}

		wordPositions := map[string]int{ // словарик слово позиция в векторе
			"rag": 0, "retrieval": 0, "augmented": 0, "generation": 0,
			"эмбеддинг": 1, "эмбеддинги": 1, "embedding": 1, "vector": 1, "вектор": 1,
			"поиск": 2, "search": 2,
			"qdrant": 4, "коллекция": 4,
			"fileauditor": 5, "auditor": 5, "аудит": 5,
			"установка": 6, "установить": 6, "install": 6, "docsearch": 6,
			"индексация": 7, "index": 7, "чанк": 8, "chunk": 8,
			"llm": 9, "модель": 16,
			"документ": 10, "документация": 12,
			"файл": 21, "доступ": 22, "журнал": 23,
			"сканирование": 28, "контроль": 25,
		}

		embedding := make([]float64, vectorSize)
		textLower := strings.ToLower(text)

		for word, pos := range wordPositions { // ключевые слова высокий вес
			if pos < vectorSize && strings.Contains(textLower, word) {
				embedding[pos] += 5.0
			}
		}

		hash := 0
		for _, ch := range textLower {
			hash = hash*31 + int(ch)
		}
		for i := 0; i < vectorSize; i++ {
			embedding[i] += float64((hash+i*7)%100) / 5000.0 // слабый вклад
		}

		var norm float64 // нормализация
		for _, v := range embedding {
			norm += v * v
		}
		norm = math.Sqrt(norm)
		if norm > 0 {
			for i := range embedding {
				embedding[i] /= norm
			}
		}

		return embedding, nil
	}

	url := cfg.Embeddings.BaseURL + "/v1/embeddings" // реальный запрос к ллм
	model := cfg.Embeddings.Model

	data := map[string]interface{}{ //запрос
		"input": []string{text},
		"model": model,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			fmt.Printf("Повторная попытка %d для LM Studio\n", attempt+1)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		client := &http.Client{Timeout:120*time.Second} //таймаут
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData)) // отправка post
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode!=200{ //проверка статуса
			body,_:=io.ReadAll(resp.Body)
			resp.Body.Close()
		
		if resp.StatusCode == 408 || resp.StatusCode == 429 || resp.StatusCode >= 500{    // ретраю 408, 429 и 5xx
			lastErr=fmt.Errorf("LM Studio ошибка %d: %s", resp.StatusCode, string(body))
			continue
		}
		return nil, fmt.Errorf("LM Studio ошибка %d: %s", resp.StatusCode, string(body)) //а другие нет
	}

		var result struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			continue
		}

		if len(result.Data) == 0 {
			lastErr = fmt.Errorf("LM Studio вернул пустой ответ")
			continue
		}

		return result.Data[0].Embedding, nil
	}
	return nil, fmt.Errorf("не получилось получить эмбеддинг после 3 попыток: %w", lastErr)
}

