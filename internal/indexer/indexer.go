package indexer

import (
	"context"
	"crypto/sha256"
	"docsearch/internal/chunk"
	"docsearch/internal/config"
	"docsearch/internal/corpus"
	"docsearch/internal/embed"
	"docsearch/internal/vector"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"os"
	"path/filepath"
	"strings"
	"docsearch/internal/parser" 
)

type Indexer struct { //структура индексации
	Config       *config.Config
	VectorClient vector.VectorStore
	IndexPath    string
	UserID       string
}

func NewIndexer(cfg *config.Config, vc vector.VectorStore, userID string) *Indexer { //новый индексер
	return &Indexer{
		Config:       cfg,
		VectorClient: vc,
		IndexPath:    "./.docsearch_index_" + userID + ".json",
		UserID:       userID,
	}
}

func (i *Indexer) loadDocument(path string) (corpus.Document, error) {
	ext := strings.ToLower(filepath.Ext(path))
	var doc corpus.Document

	if ext == ".pdf" {
		parsed, err := parser.ParsePDFDocling(path)   // использую умный парсер docling
		if err != nil {
			fmt.Printf("Docling не сработал: %v, пробую fallback\n", err)
			return i.loadDocumentFallback(path)
		}

		doc = corpus.Document{
			Name:        parsed.Name,
			Text:        parsed.Text,
			Pages:       make(map[int]string),
			PageOffsets: []int{},
		}
		return doc, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return doc, err
	}

	doc = corpus.Document{
		Name:        filepath.Base(path),
		Text:        string(data),
		Pages:       make(map[int]string),
		PageOffsets: []int{},
	}
	return doc, nil
}
func (i *Indexer) loadDocumentFallback(path string) (corpus.Document, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return corpus.Document{}, fmt.Errorf("ошибка открытия PDF: %w", err)
	}
	defer file.Close()

	var fullText strings.Builder
	pages := make(map[int]string)
	var pageOffsets []int
	offset := 0

	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		pageOffsets = append(pageOffsets, offset)
		pages[i] = content
		fullText.WriteString(content)
		fullText.WriteString("\n")
		offset += len(content) + 1
	}

	doc := corpus.Document{
		Name:        filepath.Base(path),
		Text:        fullText.String(),
		Pages:       pages,
		PageOffsets: pageOffsets,
	}
	return doc, nil
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

	files, err := os.ReadDir(userDocsPath) // читаю файлы в папке пользователя
	if err != nil {
		return err
	}

	var docs []corpus.Document
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filePath := filepath.Join(userDocsPath, file.Name())

		ext := strings.ToLower(filepath.Ext(file.Name()))
		supported := false
		for _, f := range i.Config.Corpus.Formats {
			if strings.HasPrefix(ext, ".") {
				if ext == "."+f {
					supported = true
					break
				}
			} else {
				if ext == f {
					supported = true
					break
				}
			}
		}
		if !supported {
			continue
		}

		doc, err := i.loadDocument(filePath)
		if err != nil {
			fmt.Printf("Ошибка загрузки %s: %v\n", file.Name(), err)
			continue
		}
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		fmt.Printf("В папке %s нет поддерживаемых документов\n", userDocsPath)

		entries, err := os.ReadDir(userDocsPath) // проверка что в папке есть файлы
		if err != nil {
			fmt.Println("Не удалось проверить папку:", err)
		} else if len(entries) == 0 {
			fmt.Println("Папка пуста. Положите документы в:", userDocsPath)
		} else {
			fmt.Printf("В папке есть файлы, но они не подходят по формату.\n")
			fmt.Printf("Поддерживаются форматы: %v\n", i.Config.Corpus.Formats)
			fmt.Println("Проверьте расширения файлов (.md, .txt, .pdf, .html)")
		}

		i.deleteAllUserDocs(ctx)
		return nil
	}

	old := map[string]string{} // читаю старые хеши из файла
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
	err = os.WriteFile(i.IndexPath, data, 0644)
	if err != nil {
		fmt.Printf("Предупреждение: не удалось сохранить индекс: %v\n", err)
	}

	fmt.Println("Индексация завершена")
	return nil
}

func (i *Indexer) saveDoc(ctx context.Context, doc corpus.Document) error {
	if len(strings.TrimSpace(doc.Text)) == 0 {
		fmt.Printf("Документ %s пуст, пропускаем\n", doc.Name)
		return nil
	}

	chunks := chunk.SplitIntelligent(doc.Text, doc.Name, i.Config.Chunking.MaxTokens, i.Config.Chunking.OverlapTokens) // режу на чанки

	fmt.Printf("Документ: %s, страниц: %d\n", doc.Name, len(doc.Pages))

	for idx, ch := range chunks {

		page := doc.GetPageByPosition(ch.StartPos)

		fmt.Printf("Чанк %d: страница %d, позиция %d\n", idx+1, page, ch.StartPos)

		var cache *EmbeddingCache // создаю кеш один раз для всего документа
		if i.Config.Embeddings.Provider == "local" {
			cache = NewEmbeddingCache()
		}

		vec, err := func() ([]float64, error) { // внутри цикла по чанкам проверяю кеш
			if cache != nil {
				if cached, ok := cache.Get(ch.Text); ok {
					fmt.Printf("Чанк %d: эмбеддинг взят из кеша\n", idx+1)
					return cached, nil
				}
			}

			vec, err := embed.GetEmbedding(ctx, ch.Text, i.Config) // считаю эмбеддинг
			if err != nil {
				return nil, err
			}

			if cache != nil {
				cache.Save(ch.Text, vec)
			}

			return vec, nil
		}()

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
			"doc_id":      doc.Name,
			"chunk_text":  ch.Text,
			"section":     ch.Section,
			"level":       ch.Level,
			"token_count": ch.TokenCount,
			"user_id":     i.UserID,
			"page":        page,
			"chunk_id":    id,
			"text":        ch.Text,
		}

		err = i.VectorClient.Save(ctx, vector.CollectionName, id, vec32, data)
		if err != nil {
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
	data := doc.Text + //текст с настройками
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