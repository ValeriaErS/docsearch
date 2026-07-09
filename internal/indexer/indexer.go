// indexer.go — сохраняю документы в Qdrant
package indexer

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"

    "docsearch/internal/chunk"
    "docsearch/internal/config"
    "docsearch/internal/corpus"
    "docsearch/internal/embed"
    "docsearch/internal/vector"
)

type Indexer struct {
    Config       *config.Config
    VectorClient *vector.QdrantClient
    IndexPath    string
}

func NewIndexer(cfg *config.Config, vc *vector.QdrantClient) *Indexer {
    return &Indexer{
        Config:       cfg,
        VectorClient: vc,
        IndexPath:    "./.docsearch_index.json",
    }
}

// главная функция
func (i *Indexer) Index() error {
    // читаю все документы
    docs, err := corpus.LoadDocuments(i.Config.Corpus.Path)
    if err != nil {
        return err
    }

    // читаю старые хеши
    old := map[string]string{}
    data, _ := os.ReadFile(i.IndexPath)
    json.Unmarshal(data, &old)

    // для каждого документа смотрю, изменился ли он
    for _, doc := range docs {
        h := hashText(doc.Text)

        if old[doc.Name] != h {
            fmt.Println("Индексирую:", doc.Name)
            deleteDoc(i.VectorClient, doc.Name)
            saveDoc(i.VectorClient, doc, i.Config.Chunking.MaxTokens)
            old[doc.Name] = h
        } else {
            fmt.Println("Пропускаю:", doc.Name)
        }
    }

    // удаляю документы, которых нет в папке
    for name := range old {
        found := false
        for _, doc := range docs {
            if doc.Name == name {
                found = true
                break
            }
        }
        if !found {
            fmt.Println("Удаляю:", name)
            deleteDoc(i.VectorClient, name)
            delete(old, name)
        }
    }

    // сохраняю новые хеши
    data, _ = json.MarshalIndent(old, "", "  ")
    os.WriteFile(i.IndexPath, data, 0644)

    return nil
}

// сохраняю один документ
func saveDoc(vc *vector.QdrantClient, doc corpus.Document, maxTokens int) {
    chunks := chunk.SplitIntelligent(doc.Text, doc.Name, maxTokens)

    for i := 0; i < len(chunks); i++ {
        ch := chunks[i]
        vec, _ := embed.GetEmbedding(ch.Text)

        id := doc.Name + "_" + string(rune(i))
        data := map[string]interface{}{
            "doc_id":     doc.Name,
            "chunk_text": ch.Text,
            "section":    ch.Section,
            "level":      ch.Level,
        }
        vc.UpsertPoint("documents", id, vec, data)
    }
}

// удаляю документ
func deleteDoc(vc *vector.QdrantClient, name string) {
    vc.DeletePoints("documents", map[string]interface{}{"doc_id": name})
}

// считаю хеш
func hashText(text string) string {
    h := sha256.Sum256([]byte(text))
    return hex.EncodeToString(h[:])
}