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
)

type QdrantClient struct {
    Host string
    Port int
    VectorSize int
}

func NewQdrantClient() *QdrantClient {   // создаю нового клиента
   /* return &QdrantClient{Host: "localhost", Port: 6333}*/
host := os.Getenv("QDRANT_HOST")
    if host == "" {
        panic("QDRANT_HOST не задан в .env")
    }
    
    portStr := os.Getenv("QDRANT_PORT")  //  порт из .env
    if portStr != "" {
        panic("QDRANT_PORT не задан в .env")
    }

    port, err := strconv.Atoi(portStr)
    if err != nil || port <= 0 {
        panic("QDRANT_PORT должен быть положительным числом")
    }
    
    return &QdrantClient{
        Host: host,
        Port: port,
    }
}

func (q *QdrantClient) url(path string) string { //адрес
    scheme := "http"
    if q.Host != "localhost" {
        scheme = "https"
    }
    return fmt.Sprintf("%s://%s:%d%s", scheme, q.Host, q.Port, path)
}

func (q *QdrantClient) Ping() error {
    cl := &http.Client{
        Timeout: 10 * time.Second,
    }
    req, err := http.NewRequest("GET", q.url("/collections"), nil)
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

func (q *QdrantClient) CreateCollection(name string) error {  // создаю коллекцию
    
    cl := &http.Client{
        Timeout: 10 * time.Second,
    }
    req, err := http.NewRequest("GET", q.url("/collections/" + name), nil)
    if err != nil {
        return err
    }

    resp, err := cl.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode == 200 {
        return nil
    }

    body := []byte(`{"vectors":{"size":` + fmt.Sprint(q.VectorSize) + `,"distance":"Cosine"}}`)
    req, err = http.NewRequest("PUT", q.url("/collections/"+name), bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    cl = &http.Client{
        Timeout: 30 * time.Second,
    }
    resp, err = cl.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("ошибка %d", resp.StatusCode)
    }
    return nil
}

func (q *QdrantClient) Save(name string, id string, vec []float32, data map[string]interface{}) error {  // сохраняю один чанк в бд
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
        return err
    }

    req, err := http.NewRequest("PUT", q.url("/collections/"+name+"/points"), bytes.NewBuffer(j))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    cl := &http.Client{
        Timeout: 30 * time.Second,
    }
    r, err := cl.Do(req)
    if err != nil {
        return err
    }
    defer r.Body.Close()

    body, _ := io.ReadAll(r.Body)    // читаем тело ответа из-за ошибок
    if r.StatusCode != 200 {
        return fmt.Errorf("ошибка %d: %s", r.StatusCode, string(body))
    }
    return nil
}

func (q *QdrantClient) Search(name string, vec []float32, limit int, userID string) ([]map[string]interface{}, error) {   // ищу похожие чанки
    d := map[string]interface{}{
        "vector": vec,
        "limit": limit,
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
    j, _ := json.Marshal(d)

    req, _ := http.NewRequest("POST", q.url("/collections/"+name+"/points/search"), bytes.NewBuffer(j))
    req.Header.Set("Content-Type", "application/json")

    cl := &http.Client{
        Timeout: 60 * time.Second,
    }
    r, _ := cl.Do(req)
    defer r.Body.Close()

    var res struct {    // читаю ответ
        Result []struct {
            Id string `json:"id"`
            Score float64 `json:"score"`
            Payload map[string]interface{} `json:"payload"`
        } `json:"result"`
    }
    json.NewDecoder(r.Body).Decode(&res)

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

func (q *QdrantClient) Delete(name string, filter map[string]interface{}) error {
    data := map[string]interface{}{
        "filter": filter,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", q.url("/collections/"+name+"/points/delete"), bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    cl := &http.Client{
        Timeout: 30 * time.Second,
    }
    r, err := cl.Do(req)
    if err != nil {
        return err
    }
    defer r.Body.Close()

    if r.StatusCode != 200 {
        return fmt.Errorf("ошибка удаления: %d", r.StatusCode)
    }
    return nil
}