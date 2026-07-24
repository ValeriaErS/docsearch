package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"docsearch/internal/chunk"
	"docsearch/internal/config"
	"docsearch/internal/corpus"
	"docsearch/internal/embed"
	"docsearch/internal/vector"
	"github.com/google/uuid"
	"context"
)

type Indexer struct { //структура индексации
	Config *config.Config
	VectorClient vector.VectorStore
	IndexPath string
	UserID string
}

func NewIndexer(cfg *config.Config, vc vector.VectorStore, userID string) *Indexer { //новый индексер
	return &Indexer{
		Config: cfg,
		VectorClient: vc,
		IndexPath: "./.docsearch_index_" + userID + ".json",
		UserID: userID,
	}
}

func (i *Indexer) Index(ctx context.Context) error {
	err := i.VectorClient.CreateCollection(ctx, vector.CollectionName) // создаю коллекцию
	if err != nil {
		return fmt.Errorf("ошибка создания коллекции: %w", err)
	}

	userDocsPath := filepath.Join(i.Config.Corpus.Path, i.UserID) // путь к папке пользователя

	if _, err := os.Stat(userDocsPath); os.IsNotExist(err) {
		os.MkdirAll(userDocsPath, 0755)
		fmt.Printf("Папка для пользователя %s создана: %s\n", i.UserID, userDocsPath)
		fmt.Println("Положите документы в папку:", userDocsPath)
		return nil
	}

	docs, err := corpus.LoadDocuments(userDocsPath, i.Config.Corpus.Formats) // загружаю документы из папки
	if err != nil {
		return err
	}

	if len(docs) == 0 {
		fmt.Printf("В папке %s нет документов\n", userDocsPath)
		i.deleteAllUserDocs(ctx)
		return nil
	}

	old := map[string]string{}           // читаю старые хеши из файла
	data, _ := os.ReadFile(i.IndexPath)  
	json.Unmarshal(data, &old)

	for _, doc := range docs {
		hash := hashText(doc, i.Config) 

		if old[doc.Name] != hash { // если хеш изменился или документа не было индексирую
			fmt.Println("Индексирую:", doc.Name)
			i.deleteDoc(ctx, doc.Name) 

			err := i.saveDoc(ctx, doc)
			if err != nil {
				fmt.Println("Ошибка сохранения:", err)
				continue
			}

			old[doc.Name] = hash
		} else {
			fmt.Println("Без изменений:", doc.Name)
		}
	}

	for name := range old { // проверка не удалила ли какие-то документы
		found := false
		for _, doc := range docs {
			if doc.Name == name {
				found = true
				break
			}
		}
		if !found {
			fmt.Println("Удалён из Qdrant:", name)
			i.deleteDoc(ctx, name)
			delete(old, name)
		}
	}

	data, _ = json.MarshalIndent(old, "", "  ")
	os.WriteFile(i.IndexPath, data, 0644)

	fmt.Println("Индексация завершена")
	return nil
}

func (i *Indexer) saveDoc(ctx context.Context, doc corpus.Document) error {
	chunks := chunk.SplitIntelligent(doc.Text, doc.Name, i.Config.Chunking.MaxTokens, i.Config.Chunking.OverlapTokens) // режу на чанки

	fmt.Printf("Документ: %s, страниц: %d\n", doc.Name, len(doc.Pages))

	for idx, ch := range chunks {
		
		page := 1
		if doc.Pages != nil && len(doc.Pages) > 0 {

			page = 1 + (idx * len(doc.Pages) / len(chunks))
			if page > len(doc.Pages) {
				page = len(doc.Pages)
			}
		}
		fmt.Printf("Чанк %d: страница %d\n", idx+1, page)

		vec, err := embed.GetEmbedding(ctx, ch.Text, i.Config) // получаю эмбеддинг
		if err != nil {
			fmt.Println("Ошибка эмбеддинга:", err)
			return err
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
			"page": page,
		}

		err = i.VectorClient.Save(ctx, vector.CollectionName, id, vec32, data)
		if err!= nil {
			fmt.Println("Ошибка сохранения:", err)
			return err
		}
	}
	return nil
}

func (i *Indexer) deleteDoc(ctx context.Context, name string) { // удаляю все чанки документа из бд
	filter := map[string]interface{}{
        "must": []map[string]interface{}{
            {"key": "doc_id", "match": map[string]interface{}{"value": name}},
            {"key": "user_id", "match": map[string]interface{}{"value": i.UserID}},
        },
    }
    i.VectorClient.Delete(ctx, vector.CollectionName, filter)
}

func hashText(doc corpus.Document, cfg *config.Config) string { // считаю хеш текста
	data:=doc.Text + //текст с настройками 
	    fmt.Sprintf("|%d|", cfg.Chunking.MaxTokens) +
        fmt.Sprintf("%d|", cfg.Chunking.OverlapTokens) +
        cfg.Embeddings.Model + "|" +
        fmt.Sprintf("%d|", cfg.Embeddings.VectorSize) +
        cfg.LLM.Model + "|" +
        fmt.Sprintf("%d", cfg.Retrieval.TopK)

    h := sha256.Sum256([]byte(data))
    return hex.EncodeToString(h[:])
}

func (i *Indexer) deleteAllUserDocs(ctx context.Context) {
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
	i.VectorClient.Delete(ctx, vector.CollectionName, filter)
	
	fmt.Printf("Все документы пользователя %s удалены\n", i.UserID)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}