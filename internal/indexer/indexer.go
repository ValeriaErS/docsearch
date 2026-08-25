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
	"os"
	"path/filepath"
	"strings"
)

type Indexer struct {
	Config       *config.Config
	VectorClient vector.VectorStore
	IndexPath    string
	UserID       string
}

func NewIndexer(cfg *config.Config, vc vector.VectorStore, userID string) *Indexer {
	return &Indexer{
		Config:       cfg,
		VectorClient: vc,
		IndexPath:    "./.docsearch_index_" + userID + ".json",
		UserID:       userID,
	}
}

func (i *Indexer) loadDocument(path string) (corpus.Document, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".pdf" {
		text, pages, pageOffsets, err := corpus.ReadPDFFile(path)
		if err != nil {
			return corpus.Document{}, fmt.Errorf("ошибка чтения PDF %s: %w", path, err)
		}
		
		return corpus.Document{
			Name:        filepath.Base(path),
			Text:        text,
			Pages:       pages,
			PageOffsets: pageOffsets,
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return corpus.Document{}, err
	}
	
	return corpus.Document{
		Name:        filepath.Base(path),
		Text:        string(data),
		Pages:       make(map[int]string),
		PageOffsets: []int{},
	}, nil
}

func (i *Indexer) Index(ctx context.Context) error {
	err := i.VectorClient.CreateCollection(ctx, vector.CollectionName)
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

	files, err := os.ReadDir(userDocsPath)
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

		entries, err := os.ReadDir(userDocsPath)
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

	old := map[string]string{}
	data, _ := os.ReadFile(i.IndexPath)
	json.Unmarshal(data, &old)

	for _, doc := range docs {
		hash := hashText(doc, i.Config)

		if old[doc.Name] != hash {
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

	// ЕСЛИ ДОКУМЕНТ ОЧЕНЬ БОЛЬШОЙ (> 500 КБ) - ИСПОЛЬЗУЕМ FALLBACK
	// Вместо постраничного разбиения (которое жрет память)
	const maxDocSize = 500000
	if len(doc.Text) > maxDocSize {
		fmt.Printf("⚠️ Документ %s слишком большой (%d байт), использую fallback (весь текст одним чанком)\n", doc.Name, len(doc.Text))
		
		// Создаем один большой чанк со всем текстом
		chunks := []chunk.IntelligentChunk{
			{
				Text:       doc.Text,
				Document:   doc.Name,
				Section:    "full",
				Level:      1,
				Index:      0,
				TokenCount: len(strings.Fields(doc.Text)),
				StartPos:   0,
			},
		}
		
		fmt.Printf("Документ: %s, страниц: %d, чанков: %d\n", doc.Name, len(doc.Pages), len(chunks))
		
		var cache *EmbeddingCache
		if i.Config.Embeddings.Provider == "local" {
			cache = NewEmbeddingCache()
			fmt.Printf("Кеш эмбеддингов создан для документа %s\n", doc.Name)
		}

		var batch []map[string]interface{}
		const batchSize = 50

		for idx, ch := range chunks {
			page := doc.GetPageByPosition(ch.StartPos)
			fmt.Printf("Чанк %d: страница %d, позиция %d\n", idx+1, page, ch.StartPos)

			var vec []float64
			var err error

			if cache != nil {
				if cached, ok := cache.Get(ch.Text); ok {
					vec = cached
					fmt.Printf("Чанк %d: эмбеддинг взят из кеша\n", idx+1)
				}
			}

			if vec == nil {
				vec, err = embed.GetEmbedding(ctx, ch.Text, i.Config)
				if err != nil {
					return fmt.Errorf("ошибка эмбеддинга для чанка %d: %w", idx+1, err)
				}

				if cache != nil {
					cache.Save(ch.Text, vec)
					fmt.Printf("Чанк %d: эмбеддинг сохранен в кеш\n", idx+1)
				}
			}

			vec32 := []float32{}
			for _, v := range vec {
				vec32 = append(vec32, float32(v))
			}

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

			batch = append(batch, map[string]interface{}{
				"id":      id,
				"vector":  vec32,
				"payload": data,
			})

			if len(batch) >= batchSize {
				fmt.Printf("Отправляю батч из %d точек в Qdrant\n", len(batch))
				if err := i.VectorClient.SaveBatch(ctx, vector.CollectionName, batch); err != nil {
					return fmt.Errorf("ошибка сохранения батча: %w", err)
				}
				batch = []map[string]interface{}{}
			}
		}

		if len(batch) > 0 {
			fmt.Printf("Отправляю остаток из %d точек в Qdrant\n", len(batch))
			if err := i.VectorClient.SaveBatch(ctx, vector.CollectionName, batch); err != nil {
				return fmt.Errorf("ошибка сохранения остатка: %w", err)
			}
		}

		fmt.Printf("Документ %s успешно сохранен (%d чанков)\n", doc.Name, len(chunks))
		return nil
	}

	// ДЛЯ МАЛЕНЬКИХ ДОКУМЕНТОВ - ИНТЕЛЛЕКТУАЛЬНОЕ РАЗБИЕНИЕ
	chunks := chunk.SplitIntelligent(doc.Text, doc.Name, i.Config.Chunking.MaxTokens, i.Config.Chunking.OverlapTokens)

	totalChunkLen := 0
	for _, ch := range chunks {
		totalChunkLen += len(ch.Text)
	}
	
	if len(chunks) == 0 || totalChunkLen < len(doc.Text)/3 || len(chunks) < 2 {
		fmt.Printf("⚠️ Интеллектуальное разбиение дало мусор (%d чанков, %d символов), использую весь текст как один чанк\n", 
			len(chunks), totalChunkLen)
		chunks = []chunk.IntelligentChunk{
			{
				Text:       doc.Text,
				Document:   doc.Name,
				Section:    "full",
				Level:      1,
				Index:      0,
				TokenCount: len(strings.Fields(doc.Text)),
				StartPos:   0,
			},
		}
	}
	
	fmt.Printf("Документ: %s, страниц: %d, чанков: %d\n", doc.Name, len(doc.Pages), len(chunks))

	var cache *EmbeddingCache
	if i.Config.Embeddings.Provider == "local" {
		cache = NewEmbeddingCache()
		fmt.Printf("Кеш эмбеддингов создан для документа %s\n", doc.Name)
	}

	var batch []map[string]interface{}
	const batchSize = 50

	for idx, ch := range chunks {
		page := doc.GetPageByPosition(ch.StartPos)
		fmt.Printf("Чанк %d: страница %d, позиция %d\n", idx+1, page, ch.StartPos)

		var vec []float64
		var err error

		if cache != nil {
			if cached, ok := cache.Get(ch.Text); ok {
				vec = cached
				fmt.Printf("Чанк %d: эмбеддинг взят из кеша\n", idx+1)
			}
		}

		if vec == nil {
			vec, err = embed.GetEmbedding(ctx, ch.Text, i.Config)
			if err != nil {
				return fmt.Errorf("ошибка эмбеддинга для чанка %d: %w", idx+1, err)
			}

			if cache != nil {
				cache.Save(ch.Text, vec)
				fmt.Printf("Чанк %d: эмбеддинг сохранен в кеш\n", idx+1)
			}
		}

		vec32 := []float32{}
		for _, v := range vec {
			vec32 = append(vec32, float32(v))
		}

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

		batch = append(batch, map[string]interface{}{
			"id":      id,
			"vector":  vec32,
			"payload": data,
		})

		if len(batch) >= batchSize {
			fmt.Printf("Отправляю батч из %d точек в Qdrant\n", len(batch))
			if err := i.VectorClient.SaveBatch(ctx, vector.CollectionName, batch); err != nil {
				return fmt.Errorf("ошибка сохранения батча: %w", err)
			}
			batch = []map[string]interface{}{}
		}
	}

	if len(batch) > 0 {
		fmt.Printf("Отправляю остаток из %d точек в Qdrant\n", len(batch))
		if err := i.VectorClient.SaveBatch(ctx, vector.CollectionName, batch); err != nil {
			return fmt.Errorf("ошибка сохранения остатка: %w", err)
		}
	}

	fmt.Printf("Документ %s успешно сохранен (%d чанков)\n", doc.Name, len(chunks))

return nil
}


/// saveDocByPages - индексирует большой документ постранично
func (i *Indexer) saveDocByPages(ctx context.Context, doc corpus.Document) error {
	fmt.Printf("Разбиваю %s на %d страниц\n", doc.Name, len(doc.Pages))
	
	var allChunks []chunk.IntelligentChunk
	pageNum := 1
	
	for pageNum <= len(doc.Pages) {
		pageText, ok := doc.Pages[pageNum]
		if !ok || len(strings.TrimSpace(pageText)) == 0 {
			pageNum++
			continue
		}
		
		// Если страница слишком большая - режем её на части принудительно
		var pageChunks []chunk.IntelligentChunk
		if len(pageText) > 50000 {
			// Режем страницу на части по 50000 символов
			pageChunks = i.splitLargePage(pageText, doc.Name, pageNum)
		} else {
			// Каждую страницу обрабатываем отдельно
			pageChunks = chunk.SplitIntelligent(pageText, doc.Name, i.Config.Chunking.MaxTokens, i.Config.Chunking.OverlapTokens)
		}
		
		// Добавляем номер страницы в каждый чанк
		for _, ch := range pageChunks {
			ch.Page = pageNum
			allChunks = append(allChunks, ch)
		}
		
		fmt.Printf("  Страница %d: %d чанков\n", pageNum, len(pageChunks))
		pageNum++
	}
	
	if len(allChunks) == 0 {
		fmt.Printf("Документ %s не содержит текста\n", doc.Name)
		return nil
	}
	
	fmt.Printf("Всего %d чанков для %s\n", len(allChunks), doc.Name)
	
	// Сохраняем все чанки батчами
	var batch []map[string]interface{}
	const batchSize = 50
	
	for idx, ch := range allChunks {
		// Получаем эмбеддинг
		vec, err := embed.GetEmbedding(ctx, ch.Text, i.Config)
		if err != nil {
			return fmt.Errorf("ошибка эмбеддинга для чанка %d: %w", idx+1, err)
		}
		
		vec32 := []float32{}
		for _, v := range vec {
			vec32 = append(vec32, float32(v))
		}
		
		id := uuid.New().String()
		data := map[string]interface{}{
			"doc_id":      doc.Name,
			"chunk_text":  ch.Text,
			"section":     ch.Section,
			"level":       ch.Level,
			"token_count": ch.TokenCount,
			"user_id":     i.UserID,
			"page":        ch.Page,
			"chunk_id":    id,
			"text":        ch.Text,
		}
		
		batch = append(batch, map[string]interface{}{
			"id":      id,
			"vector":  vec32,
			"payload": data,
		})
		
		if len(batch) >= batchSize {
			if err := i.VectorClient.SaveBatch(ctx, vector.CollectionName, batch); err != nil {
				return fmt.Errorf("ошибка сохранения батча: %w", err)
			}
			batch = []map[string]interface{}{}
		}
	}
	
	if len(batch) > 0 {
		if err := i.VectorClient.SaveBatch(ctx, vector.CollectionName, batch); err != nil {
			return fmt.Errorf("ошибка сохранения остатка: %w", err)
		}
	}
	
	return nil
}

// splitLargePage - разбивает большую страницу на части без использования splitBySize
func (i *Indexer) splitLargePage(text string, docName string, pageNum int) []chunk.IntelligentChunk {
	var result []chunk.IntelligentChunk
	chunkSize := 20000 // 20 тысяч символов на чанк для больших страниц
	overlap := 2000    // 2 тысячи символов перекрытия
	
	start := 0
	index := 0
	
	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}
		
		// Ищем конец предложения
		cutPos := end
		for i := end - 1; i > start && i > end-500; i-- {
			if text[i] == '.' || text[i] == '?' || text[i] == '!' || text[i] == '\n' {
				cutPos = i + 1
				break
			}
		}
		
		if cutPos <= start || cutPos > len(text) {
			cutPos = end
		}
		
		chunkText := text[start:cutPos]
		if len(chunkText) > 20 {
			result = append(result, chunk.IntelligentChunk{
				Text:        chunkText,
				Document:    docName,
				Section:     fmt.Sprintf("page_%d_part_%d", pageNum, index+1),
				Level:       0,
				Index:       index,
				TokenCount:  len(strings.Fields(chunkText)),
				OverlapFrom: -1,
				StartPos:    start,
				Page:        pageNum,
			})
			index++
		}
		
		start = cutPos - overlap
		if start < 0 {
			start = 0
		}
		if start >= len(text) {
			break
		}
		if start == cutPos {
			start = cutPos + 1
		}
	}
	
	return result
}
func (i *Indexer) deleteDoc(ctx context.Context, name string) {
	filter := map[string]interface{}{
		"must": []map[string]interface{}{
			{"key": "doc_id", "match": map[string]interface{}{"value": name}},
			{"key": "user_id", "match": map[string]interface{}{"value": i.UserID}},
		},
	}
	i.VectorClient.Delete(ctx, vector.CollectionName, filter)
}

func hashText(doc corpus.Document, cfg *config.Config) string {
	data := doc.Text +
		fmt.Sprintf("|%d|", cfg.Chunking.MaxTokens) +
		fmt.Sprintf("%d|", cfg.Chunking.OverlapTokens) +
		cfg.Embeddings.Model + "|" +
		fmt.Sprintf("%d|", cfg.Embeddings.VectorSize) +
		cfg.LLM.Model + "|" +
		fmt.Sprintf("%d|", cfg.Retrieval.CandidateTopK) +
		fmt.Sprintf("%d|", cfg.Retrieval.RerankTopK) +
		fmt.Sprintf("%d", cfg.Retrieval.FinalTopK)

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