package vector

import (
    "bytes"
    "fmt"
    "net/http"
    "encoding/json" 
)

type QdrantClient struct { //хранит настройки подключения к базе
    Host string
    Port int
}

func NewQdrantClient() *QdrantClient {   //создает новый клиент с настройками по умолчанию
    return &QdrantClient{
        Host: "localhost",
        Port: 6333,
    }
}

func (q *QdrantClient) url(path string) string {   // собирает полный адрес для запроса
    return fmt.Sprintf("http://%s:%d%s", q.Host, q.Port, path)
}

func (q *QdrantClient) Ping() error {    // проверяет что запущен и отвечает на запросы
    resp, err := http.Get(q.url("/collections"))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("ошибка %d", resp.StatusCode)
    }
    return nil
}

func (q *QdrantClient) CreateCollection(name string, size int) error {
    resp, err := http.Get(q.url("/collections/" + name)) // проверяю, существует ли коллекция
    if err != nil {
        return err
    }
    resp.Body.Close()

    if resp.StatusCode == 200 {
        return nil
    }

    jsonData := []byte(`{"vectors":{"size":` + fmt.Sprint(size) + `,"distance":"Cosine"}}`)

    req, err := http.NewRequest("PUT", q.url("/collections/"+name), bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err = client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("ошибка создания коллекции: %d", resp.StatusCode)
    }
    return nil
}

func (q *QdrantClient) SavePoint(collectionName string, id string, vector []float32, payload map[string]interface{}) error {  // сохраняет один чанк в бд
    data := map[string]interface{}{
        "points": []map[string]interface{}{
            {
                "id":      id,
                "vector":  vector,
                "payload": payload,
            },
        },
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("PUT", q.url("/collections/"+collectionName+"/points"), bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("ошибка сохранения: %d", resp.StatusCode)
    }
    return nil
}