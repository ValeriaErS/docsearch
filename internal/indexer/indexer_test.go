package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"docsearch/internal/config"
	"docsearch/internal/corpus"
	"docsearch/internal/vector"
)

func TestNewIndexer(t *testing.T) {  // тест создания индексатора
	cfg := &config.Config{}
	cfg.Corpus.Path = "./docs"
	cfg.Chunking.MaxTokens = 512
	cfg.Embeddings.VectorSize = 768

	fakeClient := vector.NewFakeVectorStore()
	idx := NewIndexer(cfg, fakeClient, "test")

	if idx == nil {
		t.Error("Индексатор не создался")
	}
	if idx.UserID != "test" {
		t.Errorf("UserID не совпадает: %s, ожидалось: test", idx.UserID)
	}
	t.Log("Индексатор создан")
}

func TestHashText(t *testing.T) {  // тест хеширования текста с разными параметрами
	doc := corpus.Document{
		Name: "test.md",
		Text: "Тестовый текст",
		Pages: nil,
	}

	cfg := &config.Config{}
	cfg.Chunking.MaxTokens = 512
	cfg.Chunking.OverlapTokens = 64
	cfg.Embeddings.Model = "test-model"
	cfg.Embeddings.VectorSize = 768
	cfg.LLM.Model = "test-llm"
	cfg.Retrieval.TopK = 5

	hash1 := hashText(doc, cfg)
	if hash1 == "" {
		t.Error("Хеш пустой")
	}
	if len(hash1) != 64 {
		t.Errorf("Длина хеша: %d, ожидалось 64", len(hash1))
	}

	cfg.Chunking.MaxTokens = 1024
	hash2 := hashText(doc, cfg)
	if hash1 == hash2 {
		t.Error("Хеш не изменился при изменении параметров")
	}
	t.Log("Хеширование работает")
}

func TestSaveDoc(t *testing.T) {   // тест сохр доков
	fakeClient := vector.NewFakeVectorStore()

	cfg := &config.Config{}
	cfg.Corpus.Path = "./docs"
	cfg.Chunking.MaxTokens = 512
	cfg.Chunking.OverlapTokens = 64
	cfg.Embeddings.VectorSize = 768
	cfg.Embeddings.Provider = "mock"
	cfg.LLM.Provider = "mock"

	idx := NewIndexer(cfg, fakeClient, "testuser")

	doc := corpus.Document{
		Name: "test.md",
		Text: "Тестовый документ для индексации.\nВторая строка.\nТретья строка.",
		Pages: nil,
	}

	err := idx.saveDoc(context.Background(), doc)
	if err != nil {
		t.Fatalf("Ошибка сохранения: %v", err)
	}

	if len(fakeClient.Points) == 0 {
		t.Error("Документ не сохранён")
	}
	t.Logf("Сохранено %d чанков", len(fakeClient.Points))
}


func TestDeleteDoc(t *testing.T) {  //тест удаления дока
	fakeClient := vector.NewFakeVectorStore()

	cfg := &config.Config{}
	cfg.Corpus.Path = "./docs"
	cfg.Chunking.MaxTokens = 512
	cfg.Embeddings.VectorSize = 768
	cfg.Embeddings.Provider = "mock"
	cfg.LLM.Provider = "mock"

	idx := NewIndexer(cfg, fakeClient, "testuser")

	doc := corpus.Document{
		Name: "test.md",
		Text: "Тестовый документ",
		Pages: nil,
	}

	err := idx.saveDoc(context.Background(), doc)
	if err != nil {
		t.Fatalf("Ошибка сохранения: %v", err)
	}

	if len(fakeClient.Points) == 0 {
		t.Fatal("Документ не сохранился")
	}

	idx.deleteDoc(context.Background(), "test.md")

	if len(fakeClient.Points) != 0 {
		t.Errorf("После удаления точек: %d, ожидалось 0", len(fakeClient.Points))
	}
	t.Log("Документ удалён")
}

func TestDeleteAllUserDocs(t *testing.T) {   //удаление всех доков пользователя
	fakeClient := vector.NewFakeVectorStore()

	cfg := &config.Config{}
	cfg.Corpus.Path = "./docs"
	cfg.Chunking.MaxTokens = 512
	cfg.Embeddings.VectorSize = 768
	cfg.Embeddings.Provider = "mock"
	cfg.LLM.Provider = "mock"

	idx := NewIndexer(cfg, fakeClient, "testuser")

	doc1 := corpus.Document{
		Name: "doc1.md",
		Text: "Документ 1",
		Pages: nil,
	}
	doc2 := corpus.Document{
		Name: "doc2.md",
		Text: "Документ 2",
		Pages: nil,
	}

	idx.saveDoc(context.Background(), doc1)
	idx.saveDoc(context.Background(), doc2)

	if len(fakeClient.Points) == 0 {
		t.Fatal("Документы не сохранились")
	}

	idx.deleteAllUserDocs(context.Background())

	if len(fakeClient.Points) != 0 {
		t.Errorf("После удаления всех документов точек: %d, ожидалось 0", len(fakeClient.Points))
	}
	t.Log("Все документы удалены")
}

func TestIndexWithDocuments(t *testing.T) {   // тест индексации документов из папки пользователя
	fakeClient := vector.NewFakeVectorStore()

	cfg := &config.Config{}
	cfg.Corpus.Path = t.TempDir()
	cfg.Corpus.Formats = []string{"md"}
	cfg.Chunking.MaxTokens = 512
	cfg.Embeddings.VectorSize = 768
	cfg.Embeddings.Provider = "mock"
	cfg.LLM.Provider = "mock"

	userID := "testuser"
	userDocsPath := filepath.Join(cfg.Corpus.Path, userID)
	os.MkdirAll(userDocsPath, 0755)

	docPath := filepath.Join(userDocsPath, "test.md")
	os.WriteFile(docPath, []byte("# Тест\n\nСодержимое."), 0644)

	idx := NewIndexer(cfg, fakeClient, userID)
	idx.Index(context.Background())

	if len(fakeClient.Points) == 0 {
		t.Error("Документ не проиндексирован")
	} else {
		t.Logf("Проиндексировано %d чанков", len(fakeClient.Points))
	}
}

func TestIndexIncremental(t *testing.T) {  //обновление дока
	fakeClient := vector.NewFakeVectorStore()

	cfg := &config.Config{}
	cfg.Corpus.Path = t.TempDir()
	cfg.Corpus.Formats = []string{"md"}
	cfg.Chunking.MaxTokens = 512
	cfg.Embeddings.VectorSize = 768
	cfg.Embeddings.Provider = "mock"
	cfg.LLM.Provider = "mock"

	userID := "testuser"
	userDocsPath := filepath.Join(cfg.Corpus.Path, userID)
	os.MkdirAll(userDocsPath, 0755)

	docPath := filepath.Join(userDocsPath, "test.md")
	os.WriteFile(docPath, []byte("# Тест\n\nВерсия 1"), 0644)

	idx := NewIndexer(cfg, fakeClient, userID)
	idx.Index(context.Background())
	firstCount := len(fakeClient.Points)

	os.WriteFile(docPath, []byte("# Тест\n\nВерсия 2 - изменено"), 0644)
	idx.Index(context.Background())
	secondCount := len(fakeClient.Points)

	t.Logf("Первая: %d, вторая: %d", firstCount, secondCount)
}

func TestDeleteDocOtherUser(t *testing.T) {  //другой не может удалить док
	fakeClient := vector.NewFakeVectorStore()

	cfg := &config.Config{}
	cfg.Corpus.Path = "./docs"
	cfg.Chunking.MaxTokens = 512
	cfg.Embeddings.VectorSize = 768
	cfg.Embeddings.Provider = "mock"
	cfg.LLM.Provider = "mock"

	idx1 := NewIndexer(cfg, fakeClient, "user1")
	doc := corpus.Document{
		Name: "test.md",
		Text: "Документ пользователя 1",
		Pages: nil,
	}
	idx1.saveDoc(context.Background(), doc)

	idx2 := NewIndexer(cfg, fakeClient, "user2")
	idx2.deleteDoc(context.Background(), "test.md")

	if len(fakeClient.Points) == 0 {
		t.Error("Документ удалён другим пользователем")
	} else {
		t.Log("Tenant изоляция работает")
	}
}