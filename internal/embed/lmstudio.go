package embed

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func GetEmbedding(text string) ([]float64, error) { //отправка текста в LM с возвратом эмбеддинга
    url := "http://127.0.0.1:1234/v1/embeddings"

    data := map[string]interface{}{  //запрос
        "input": []string{text},
        "model": "text-embedding-nomic-embed-text-v1.5",
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return nil, err
    }

    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData)) // отправка post
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Data []struct {
            Embedding []float64 `json:"embedding"` //ответ
        } `json:"data"`
    }

    err = json.NewDecoder(resp.Body).Decode(&result)
    if err != nil {
        return nil, err
    }

    if len(result.Data) == 0 {
        return nil, nil
    }

    return result.Data[0].Embedding, nil
}

