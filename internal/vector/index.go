package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"bytes"
	"io"
)

func (q *QdrantClient) EnsureTextIndex(ctx context.Context, collectionName string) error {  //создает текстовый индекс для поля text
	exists, err := q.textIndexExists(ctx, collectionName)
	if err != nil {
		fmt.Printf("Проверка индекса: %v, создаю...\n", err)
		return q.createTextIndex(ctx, collectionName)
	}

	if exists {
		return nil
	}

	return q.createTextIndex(ctx, collectionName)
}

func (q *QdrantClient) textIndexExists(ctx context.Context, collectionName string) (bool, error) {  // проверяет наличие текстового индекса
	url := q.url("/collections/" + collectionName + "/index/text")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := retryRequest(req, 2)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, nil
	}
	if resp.StatusCode == 404 {
		return false, nil
	}

	return false, fmt.Errorf("неожиданный статус: %d", resp.StatusCode)
}

func (q *QdrantClient) createTextIndex(ctx context.Context, collectionName string) error {
    fmt.Printf("Создаю текстовый индекс для поля 'text' в коллекции %s...\n", collectionName)

	indexConfig := map[string]interface{}{
        "field_name":  "text",
        "field_type":  "text",
        "tokenizer":   "word",     
        "min_token_len": 2,
        "max_token_len": 20,
    }

	jsonData, err := json.Marshal(indexConfig)
    if err != nil {
        return fmt.Errorf("ошибка маршалинга: %w", err)
    }

    url := q.url("/collections/" + collectionName + "/index")
    req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := retryRequest(req, 3)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
        fmt.Printf("Текстовый индекс создан\n")
        return nil
    }

    if resp.StatusCode == 409 {
        fmt.Printf("Индекс уже существует\n")
        return nil
    }

    body, _ := io.ReadAll(resp.Body)
    return fmt.Errorf("ошибка создания индекса: статус %d, тело: %s", resp.StatusCode, string(body))
}