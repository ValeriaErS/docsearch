package indexer

import (
	"fmt"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "os"
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
    err := i.VectorClient.CreateCollection("documents")   // создаю коллекцию
    if err != nil {
        return fmt.Errorf("ошибка создания коллекции: %w", err)
    }

    docs, err := corpus.LoadDocuments(i.Config.Corpus.Path)  // загружаю документы из папки
    if err != nil {
        return err
    }

    old := map[string]string{}    // читаю старые хеши из файла
    data, _ := os.ReadFile(i.IndexPath)
    json.Unmarshal(data, &old)

    for _, doc := range docs {
        hash := hashText(doc.Text)

        if old[doc.Name] != hash {    // если хеш изменился или документа не было индексирую
            fmt.Println("Индексирую:", doc.Name)
            i.deleteDoc(doc.Name)  // удаляю старые чанки
            i.saveDoc(doc)
            old[doc.Name] = hash   
        } else {
            fmt.Println("Без изменений:", doc.Name)
        }
    }

    for name := range old { // проверка не удалили ли какие то документы 
        found := false
        for _, doc := range docs {
            if doc.Name == name {
                found = true
                break
            }
        }
        if !found {
            fmt.Println("Удалён:", name)
            i.deleteDoc(name)
            delete(old, name)
        }
    }

    data, _ = json.MarshalIndent(old, "", "  ")
    os.WriteFile(i.IndexPath, data, 0644)

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