package vector

import (
    "fmt"
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "os"
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
        host = "localhost"
    }
    
    port := 6333
    if host != "localhost" {
        port = 443 
    }
    
    return &QdrantClient{
        Host: host,
        Port: port,
    }
}

func (q *QdrantClient) url(path string) string { //адрес
    return fmt.Sprintf("http://%s:%d%s", q.Host, q.Port, path)
}

func (q *QdrantClient) Ping() error {
    r, _ := http.Get(q.url("/collections"))
    defer r.Body.Close()
    if r.StatusCode != 200 {
        return fmt.Errorf("ошибка %d", r.StatusCode)
    }
    return nil
}

func (q *QdrantClient) CreateCollection(name string) error {  // создаю коллекцию
    r, _ := http.Get(q.url("/collections/" + name))
    defer r.Body.Close()
    if r.StatusCode == 200 {
        return nil
    }

    body:= []byte(`{"vectors":{"size":` + fmt.Sprint(q.VectorSize) + `,"distance":"Cosine"}}`)
    req, _ := http.NewRequest("PUT", q.url("/collections/"+name), bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    cl:= &http.Client{}
    r, _ = cl.Do(req)
    defer r.Body.Close()

    if r.StatusCode != 200 {
        return fmt.Errorf("ошибка %d", r.StatusCode)
    }
    return nil
}

func (q *QdrantClient) Save(name string, id string, vec []float32, data map[string]interface{}) error {  // сохраняю один чанк в бд
     fmt.Printf("Размер вектора: %d, ожидается: %d\n", len(vec), q.VectorSize)
    if len(vec)!=q.VectorSize{
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

    cl := &http.Client{}
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

    if userID != "" && userID != "admin" {
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

    cl := &http.Client{}
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

    cl := &http.Client{}
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