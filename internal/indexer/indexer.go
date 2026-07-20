package indexer

import (
	"fmt"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "os"
    "path/filepath"
    "github.com/google/uuid"
    "docsearch/internal/chunk"
    "docsearch/internal/config"
    "docsearch/internal/corpus"
    "docsearch/internal/embed"
    "docsearch/internal/vector"
)

type Indexer struct {  //структура индексации
    Config *config.Config
    VectorClient *vector.QdrantClient
    IndexPath string
    UserID string
}

func NewIndexer(cfg *config.Config, vc *vector.QdrantClient, userID string) *Indexer {  //новый индексер
    return &Indexer{
        Config:cfg,
        VectorClient:vc,
        IndexPath:"./.docsearch_index.json",
        UserID:userID,
    }
}

func (i *Indexer) Index() error {
    
    err := i.VectorClient.CreateCollection("documents")
    if err != nil {
        return fmt.Errorf("ошибка создания коллекции: %w", err)
    }

    
    userDocsPath := filepath.Join(i.Config.Corpus.Path, i.UserID)

   
    if _, err := os.Stat(userDocsPath); os.IsNotExist(err) {
        os.MkdirAll(userDocsPath, 0755)
        fmt.Printf("Папка для пользователя %s создана: %s\n", i.UserID, userDocsPath)
        fmt.Println("Положите документы в папку:", userDocsPath)
        return nil
    }

   
    docs, err := corpus.LoadDocuments(userDocsPath)
    if err != nil {
        return err
    }

    if len(docs) == 0 {
        fmt.Printf("В папке %s нет документов\n", userDocsPath)
        
        i.deleteAllUserDocs()
        return nil
    }

    
    old := map[string]string{}
    data, _ := os.ReadFile(i.IndexPath)
    json.Unmarshal(data, &old)

   
    for _, doc := range docs {
        hash := hashText(doc.Text)

        if old[doc.Name] != hash {
            fmt.Println("Индексирую:", doc.Name)
            i.deleteDoc(doc.Name)
            i.saveDoc(doc)
            old[doc.Name] = hash
        } else {
            fmt.Println("Без изменений:", doc.Name)
        }
    }

  
    for name := range old {
        found := false
        for _, doc := range docs {
            if doc.Name == name {
                found = true
                break
            }
        }
        if !found {
            fmt.Println("Удалён из Qdrant:", name)
            i.deleteDoc(name)
            delete(old, name)
        }
    }

  
    data, _ = json.MarshalIndent(old, "", "  ")
    os.WriteFile(i.IndexPath, data, 0644)

    fmt.Println("Индексация завершена")
    return nil
}

func (i *Indexer) saveDoc(doc corpus.Document) {
    chunks := chunk.SplitIntelligent(doc.Text, doc.Name, i.Config.Chunking.MaxTokens)

    for _, ch := range chunks {
        vec, err := embed.GetEmbedding(ch.Text)  // получаю эмбеддинг
        if err != nil {
            fmt.Println("Ошибка эмбеддинга:", err)
            continue
        }

        vec32 := []float32{}
        for _, v := range vec {
            vec32 = append(vec32, float32(v))
        }
        fmt.Println("Сохраняю чанк, размер вектора:", len(vec32))

        id := uuid.New().String()
        data := map[string]interface{}{
            "doc_id": doc.Name,
            "chunk_text": ch.Text,
            "section": ch.Section,
            "level": ch.Level,
            "token_count": ch.TokenCount,
            "user_id": i.UserID, 
        }

        err = i.VectorClient.Save("documents", id, vec32, data)
        if err != nil {
            fmt.Println("Ошибка сохранения:", err)
        }
    }
}

func (i *Indexer) deleteDoc(name string) {  //удаляю все чанки дока из бд
    filter := map[string]interface{}{"doc_id": name}
    i.VectorClient.Delete("documents", filter)
}

func hashText(text string) string {   //считаю хеш текста
    h:= sha256.Sum256([]byte(text))
    return hex.EncodeToString(h[:])
}
func (i *Indexer) deleteAllUserDocs() {
    filter := map[string]interface{}{
        "must": []map[string]interface{}{
            {
                "key": "user_id",
                "match": map[string]interface{}{
                    "value": i.UserID,
                },
            },
        },
    }
    i.VectorClient.Delete("documents", filter)
    fmt.Printf("Все документы пользователя %s удалены\n", i.UserID)
}