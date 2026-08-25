package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	CollectionName = "documents"
)

type QdrantClient struct {
	Host       string
	Port       int
	VectorSize int
	httpClient *http.Client
}

func NewQdrantClient() (*QdrantClient, error) { // создаю нового клиента
	host := os.Getenv("QDRANT_HOST")
	if host == "" {
		return nil, fmt.Errorf("QDRANT_HOST не задан в .env")
	}

	portStr := os.Getenv("QDRANT_PORT") //  порт из .env
	if portStr == "" {
		return nil, fmt.Errorf("QDRANT_PORT не задан в .env")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("QDRANT_PORT должен быть положительным числом")
	}

	return &QdrantClient{
		Host: host,
		Port: port,
		httpClient: &http.Client{  
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
    }, nil
}

func (q *QdrantClient) url(path string) string { //адрес
	scheme := "http"
	return fmt.Sprintf("%s://%s:%d%s", scheme, q.Host, q.Port, path)
}

func (q *QdrantClient) Ping(ctx context.Context) error {
	cl := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", q.url("/collections"), nil)
	if err != nil {
		return err
	}

	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("ошибка %d", resp.StatusCode)
	}
	return nil
}

func (q *QdrantClient) CreateCollection(ctx context.Context, name string) error { // создаю коллекцию
	if q.VectorSize <= 0 { //проверка размера вектора
		return fmt.Errorf("некорректный размер вектора:%d", q.VectorSize)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", q.url("/collections/"+name), nil)
	if err != nil {
		return err
	}

	resp, err := retryRequest(req, 2)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			return nil // коллекция уже существует
		}
	} else {
		fmt.Printf("Коллекция не найдена, создаем новую: %v\n", err)
	}

	body := []byte(`{"vectors":{"size":` + fmt.Sprint(q.VectorSize) + `,"distance":"Cosine"}}`) // коллекция с retry
	req, err = http.NewRequestWithContext(ctx, "PUT", q.url("/collections/"+name), bytes.NewBuffer(body))

	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = retryRequest(req, 3)
	if err != nil {
		return fmt.Errorf("ошибка создания коллекции: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("ошибка создания коллекции: статус %d", resp.StatusCode)
	}
	return nil
}

func (q *QdrantClient) Save(ctx context.Context, name string, id string, vec []float32, data map[string]interface{}) error { // сохраняю один чанк в бд
	fmt.Printf("Размер вектора: %d, ожидается: %d\n", len(vec), q.VectorSize)
	if len(vec) != q.VectorSize {
		return fmt.Errorf("Размер вектора %d, ожидается %d", len(vec), q.VectorSize)
	}

	d := map[string]interface{}{
		"points": []map[string]interface{}{
			{"id": id, "vector": vec, "payload": data},
		},
	}
	j, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", q.url("/collections/"+name+"/points"), bytes.NewBuffer(j))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	r, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка сохранения: %w", err)
	}
	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)
	if r.StatusCode != 200 {
		return fmt.Errorf("ошибка %d: %s", r.StatusCode, string(body))
	}
	return nil
}

func (q *QdrantClient) Search(ctx context.Context, name string, vec []float32, limit int, userID string) ([]map[string]interface{}, error) { // ищу похожие чанки
	d := map[string]interface{}{
		"vector":       vec,
		"limit":        limit,
		"with_payload": true,
	}

	filterUserID := userID
	if filterUserID == "" {
		filterUserID = "default"
	}
	d["filter"] = map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "user_id",
				"match": map[string]interface{}{
					"value": filterUserID,
				},
			},
		},
	}
	j, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", q.url("/collections/"+name+"/points/search"), bytes.NewBuffer(j))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	r, err := retryRequest(req, 3)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к Qdrant: %w", err)
	}
	defer r.Body.Close()

	var res struct {
		Result []struct {
			Id      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа Qdrant: %w", err)
	}

	out := []map[string]interface{}{}
	for _, item := range res.Result {

		payloadUserID, ok := item.Payload["user_id"].(string) //точно ли док нашего пользователя
		if !ok || payloadUserID != userID {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":      item.Id,
			"score":   item.Score,
			"payload": item.Payload,
		})
	}
	return out, nil
}

func (q *QdrantClient) Delete(ctx context.Context, name string, filter map[string]interface{}) error {
	data := map[string]interface{}{
		"filter": filter,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", q.url("/collections/"+name+"/points/delete"), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	r, err := retryRequest(req, 3)
	if err != nil {
		return fmt.Errorf("ошибка удаления: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return fmt.Errorf("ошибка удаления: статус %d", r.StatusCode)
	}
	return nil
}

func retryRequest(req *http.Request, maxRetries int) (*http.Response, error) { //повторные попытки
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения тела запроса: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Повторная попытка %d из %d\n", attempt+1, maxRetries)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		if len(bodyBytes) > 0 {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 408 || resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("статус %d: %s", resp.StatusCode, string(body))
			continue
		}
		return nil, fmt.Errorf("статус %d: %s", resp.StatusCode, string(body))

	}
	return nil, fmt.Errorf("не удалось выполнить запрос после %d попыток: %w", maxRetries, lastErr)
}

func (q *QdrantClient) GetAllVectors(ctx context.Context, name string, userID string) ([]map[string]interface{}, error) { //все векторы из бд
	var allPoints []map[string]interface{}
	var offset interface{}
	limit := 100

	for {
		d := map[string]interface{}{
			"limit":        limit,
			"with_vector":  true,
			"with_payload": true,
		}

		if userID != "" {
			d["filter"] = map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"key": "user_id",
						"match": map[string]interface{}{
							"value": userID,
						},
					},
				},
			}
		}
		if offset != nil {
			d["offset"] = offset
		}

		j, err := json.Marshal(d)
		if err != nil {
			return nil, fmt.Errorf("ошибка маршалинга: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", q.url("/collections/"+name+"/points/scroll"), bytes.NewBuffer(j))
		if err != nil {
			return nil, fmt.Errorf("ошибка создания запроса: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		r, err := retryRequest(req, 3)
		if err != nil {
			return nil, fmt.Errorf("ошибка запроса к Qdrant: %w", err)
		}

		var res struct {
			Result struct {
				Points []struct {
					Id      string                 `json:"id"`
					Vector  []float32              `json:"vector"`
					Payload map[string]interface{} `json:"payload"`
				} `json:"points"`
				NextPageOffset interface{} `json:"next_page_offset"`
			} `json:"result"`
		}

		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			r.Body.Close()
			return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
		}
		r.Body.Close()

		for _, point := range res.Result.Points {
			allPoints = append(allPoints, map[string]interface{}{
				"id":      point.Id,
				"vector":  point.Vector,
				"payload": point.Payload,
			})
		}
		if res.Result.NextPageOffset == nil { // если следующей страницы нет  выхожу
			break
		}
		offset = res.Result.NextPageOffset
	}
	return allPoints, nil
}

func (q *QdrantClient) SearchText(ctx context.Context, name string, query string, limit int, userID string) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}
	_ = q.ensureTextIndex(ctx, name)

	if userID == "" {
		userID = "default"
	}

	body := map[string]interface{}{
		"limit":        limit,
		"with_payload": true,
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key":   "user_id",
					"match": map[string]interface{}{"value": userID},
				},
				{
					"key":   "text",
					"match": map[string]interface{}{"text": query},
				},
			},
		},
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		q.url("/collections/"+name+"/points/scroll"), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := retryRequest(req, 3)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SearchText status %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Result struct {
			Points []struct {
				Id      string                 `json:"id"`
				Payload map[string]interface{} `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	out := make([]map[string]interface{}, 0, len(result.Result.Points))
	for i, p := range result.Result.Points {
		out = append(out, map[string]interface{}{
			"id":      p.Id,
			"score":   1.0 / float64(i+1), // для RRF важен порядок
			"payload": p.Payload,
		})
	}
	return out, nil
}

func (q *QdrantClient) ensureTextIndex(ctx context.Context, collectionName string) error {
    // Проверяем существование индекса
    req, err := http.NewRequestWithContext(ctx, "GET", q.url("/collections/"+collectionName+"/index"), nil)
    if err != nil {
        return err
    }

    resp, err := q.httpClient.Do(req)
    if err == nil && resp.StatusCode == 200 {
        resp.Body.Close()
        return nil
    }
    if resp != nil {
        resp.Body.Close()
    }

    
    indexConfig := map[string]interface{}{
	"field_name": "text",
	"field_schema": map[string]interface{}{
		"type":          "text",
		"tokenizer":     "word",
		"min_token_len": 2,
		"max_token_len": 40,
		"lowercase":     true,
	},
}

    jsonData, err := json.Marshal(indexConfig)
    if err != nil {
        return fmt.Errorf("ошибка маршалинга индекса: %w", err)
    }

    req, err = http.NewRequestWithContext(ctx, "PUT", q.url("/collections/"+collectionName+"/index"), bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err = q.httpClient.Do(req)
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

func (q *QdrantClient) SaveBatch(ctx context.Context, name string, points []map[string]interface{}) error {  // сохраняет несколько векторов за один HTTP запрос
    if len(points) == 0 {
        return nil
    }

    if len(points) <= 5 {
        for _, p := range points {
            id, _ := p["id"].(string)
            vec, _ := p["vector"].([]float32)
            payload, _ := p["payload"].(map[string]interface{})
            if err := q.Save(ctx, name, id, vec, payload); err != nil {
                return err
            }
        }
        return nil
    }

    fmt.Printf("Отправляю батч из %d точек в Qdrant\n", len(points))

    data := map[string]interface{}{
        "points": points,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("ошибка маршалинга батча: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "PUT", q.url("/collections/"+name+"/points"), bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("ошибка создания запроса: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := q.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("ошибка отправки батча: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("ошибка батча: статус %d, тело: %s", resp.StatusCode, string(body))
    }

    fmt.Printf("Батч из %d точек успешно отправлен\n", len(points))
    return nil
}