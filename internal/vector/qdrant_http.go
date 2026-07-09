package vector

import (
    "fmt"
    "bytes"
    "encoding/json"
    "net/http"
)

type QdrantClient struct {
    Host string
    Port int
}

func NewQdrantClient() *QdrantClient {
    return &QdrantClient{Host: "localhost", Port: 6333}
}

func (q *QdrantClient) url(path string) string {
    return fmt.Sprintf("http://%s:%d%s", q.Host, q.Port, path)
}
func (q *QdrantClient) Ping() error {    // проверяю работает ли бд
    r, _:= http.Get(q.url("/collections"))
    defer r.Body.Close()
    if r.StatusCode != 200 {
        return fmt.Errorf("ошибка %d", r.StatusCode)
    }
    return nil
}
func (q *QdrantClient) CreateCollection(name string) error {  // создаю коллекцию
    r, _:= http.Get(q.url("/collections/" + name))
    defer r.Body.Close()
    if r.StatusCode == 200 {
        return nil
    }

    body:= []byte(`{"vectors":{"size":768,"distance":"Cosine"}}`)   // создаю
    req, _:= http.NewRequest("PUT", q.url("/collections/"+name), bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    cl := &http.Client{}
    r, _ = cl.Do(req)
    defer r.Body.Close()

    if r.StatusCode != 200 {
        return fmt.Errorf("ошибка %d", r.StatusCode)
    }
    return nil
}

func (q *QdrantClient) Save(name string, id string, vec []float32, data map[string]interface{}) error { // сохраняю чанк
    d:= map[string]interface{}{
        "points": []map[string]interface{}{
            {"id": id, "vector": vec, "payload": data},
        },
    }
    j, _:= json.Marshal(d)

    req, _:= http.NewRequest("PUT", q.url("/collections/"+name+"/points"), bytes.NewBuffer(j))
    req.Header.Set("Content-Type", "application/json")

    cl:= &http.Client{}
    r, _:= cl.Do(req)
    defer r.Body.Close()

    if r.StatusCode!= 200 {
        return fmt.Errorf("ошибка %d", r.StatusCode)
    }
    return nil
}
func (q *QdrantClient) Search(name string, vec []float32, limit int) ([]map[string]interface{}, error) {  // ищу похожие
    d:= map[string]interface{}{
        "vector": vec,
        "limit": limit,
        "with_payload": true,
    }
    j, _:= json.Marshal(d)

    req, _:= http.NewRequest("POST", q.url("/collections/"+name+"/points/search"), bytes.NewBuffer(j))
    req.Header.Set("Content-Type", "application/json")

    cl:= &http.Client{}
    r, _:= cl.Do(req)
    defer r.Body.Close()

    var res struct {
        Result []struct {
            Id string `json:"id"`
            Score float64 `json:"score"`
            Payload map[string]interface{} `json:"payload"`
        } `json:"result"`
    }
    json.NewDecoder(r.Body).Decode(&res)

    out:= []map[string]interface{}{}
    for _, item:= range res.Result {
        out = append(out, map[string]interface{}{
            "id": item.Id, "score": item.Score, "payload": item.Payload,
        })
    }
    return out, nil
}