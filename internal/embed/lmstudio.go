package embed

import (
    "bytes"
    "encoding/json"
    "net/http"
    "docsearch/internal/config"
    "time"
    "fmt"
    "context"
    "io"
)

func GetEmbedding(ctx context.Context, text string, cfg *config.Config) ([]float64, error) { //отправка текста в LM с возвратом эмбеддинга
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
    var lastErr error
    for attempt:=0;attempt<3;attempt++{
        if attempt>0{
            time.Sleep(time.Duration(attempt)*time.Second)
        }

    client:=&http.Client{ //таймаут
        Timeout:120 * time.Second,
    }

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
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
            body, _ := io.ReadAll(resp.Body)
            lastErr = fmt.Errorf("LM Studio ошибка %d: %s", resp.StatusCode, string(body))
            continue
        }
    
        var result struct {
            Data []struct {
                Embedding []float64 `json:"embedding"`
            } `json:"data"`
        }

        err = json.NewDecoder(resp.Body).Decode(&result)
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
