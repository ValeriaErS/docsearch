package vector

import (
    "fmt"
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "time"
    "strconv"
    "context" 
)
const (
    CollectionName = "documents"
)

type QdrantClient struct {
    Host string
    Port int
    VectorSize int
}

func NewQdrantClient() (*QdrantClient, error) {   // создаю нового клиента
   /* return &QdrantClient{Host: "localhost", Port: 6333}*/
host := os.Getenv("QDRANT_HOST")
    if host == "" {
        return nil,fmt.Errorf ("QDRANT_HOST не задан в .env")
    }
    
    portStr := os.Getenv("QDRANT_PORT")  //  порт из .env
    if portStr == "" {
        return nil,fmt.Errorf ("QDRANT_PORT не задан в .env")
    }

    port, err := strconv.Atoi(portStr)
    if err != nil || port <= 0 {
        return nil,fmt.Errorf ("QDRANT_PORT должен быть положительным числом")
    }
    
    return &QdrantClient{
        Host: host,
        Port: port,
    }, nil
}

func (q *QdrantClient) url(path string) string { //адрес
    scheme := "http"
    if q.Host != "localhost" {
        scheme = "https"
    }
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

func (q *QdrantClient) CreateCollection(ctx context.Context, name string) error {  // создаю коллекцию
    
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

    body := []byte(`{"vectors":{"size":` + fmt.Sprint(q.VectorSize) + `,"distance":"Cosine"}}`)  // коллекция с retry
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

func (q *QdrantClient) Save(ctx context.Context, name string, id string, vec []float32, data map[string]interface{}) error {  // сохраняю один чанк в бд
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

    r, err := retryRequest(req, 3)
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

func (q *QdrantClient) Search(ctx context.Context, name string, vec []float32, limit int, userID string) ([]map[string]interface{}, error) {   // ищу похожие чанки
    d := map[string]interface{}{
        "vector": vec,
        "limit": limit,
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
        Id string `json:"id"`
        Score float64 `json:"score"`
        Payload map[string]interface{} `json:"payload"`
        } `json:"result"`
    }
    if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
        return nil, fmt.Errorf("ошибка парсинга ответа Qdrant: %w", err)
    }

    out := []map[string]interface{}{}
    for _, item := range res.Result {
        out = append(out, map[string]interface{}{
            "id": item.Id, 
            "score": item.Score, 
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
    
    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        if attempt > 0 {
            time.Sleep(time.Duration(attempt) * time.Second) 
        }
        
        resp, err := client.Do(req)
        if err != nil { 
            lastErr = err
            continue
        }

        if resp.StatusCode == 200 {
            return resp, nil
        }
        
        body, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        lastErr = fmt.Errorf("статус %d: %s", resp.StatusCode, string(body))
    }
    return nil, fmt.Errorf("не удалось выполнить запрос после %d попыток: %w", maxRetries, lastErr)
}