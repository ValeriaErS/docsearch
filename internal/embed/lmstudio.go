package embed

import (
    "bytes"
    "encoding/json"
    "net/http"
    "docsearch/internal/config"
)

func GetEmbedding(text string, cfg *config.Config) ([]float64, error) { //отправка текста в LM с возвратом эмбеддинга
    url := cfg.Embeddings.BaseURL + "/v1/embeddings"
    model := cfg.Embeddings.Model

    data := map[string]interface{}{  //запрос
        "input": []string{text},
        "model": model,
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

