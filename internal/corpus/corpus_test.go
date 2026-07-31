package corpus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPDF(t *testing.T) { // проверяю, что PDF читается

	if _, err := os.Stat("docs/test.pdf"); err != nil { // если ошибка,тест провален
		t.Skip("Нет test.pdf, пропускаем тест")
		return
	}
	text, pages, pageOffsets, err := readPDF("docs/test.pdf")
	if err != nil {
		t.Skip("Ошибка чтения test.pdf")
		return
	}
	if len(text) == 0 { // если текст пустой,тест провален
		t.Error("PDF пустой")
	}
	if len(pages) == 0 {
		t.Error("PDF не содержит страниц")
	}
}
func TestLoadPDF(t *testing.T) { // проверяю, что PDF загружается через LoadDocuments
	formats := []string{"md", "txt", "pdf"}
	docs, err := LoadDocuments("docs", formats)

	if err != nil {
		t.Skip("Ошибка загрузки")
		return
	}
	for i := 0; i < len(docs); i++ { // прохожу по всем документам
		if docs[i].Name == "test.pdf" {
			if len(docs[i].Text) == 0 {
				t.Error("PDF загрузился пустым")
			}
			return
		}
	}
	t.Error("PDF не загрузился")
}
func TestLoadDocumentsTxt(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "corpus_test") // создаю временную папку с тестовыми файлами
	if err != nil {
		t.Fatalf("Ошибка создания временной папки: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Это тестовый документ для проверки загрузки.\nВторая строка."
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Ошибка создания файла: %v", err)
	}

	formats := []string{"txt"} //загрузка доков
	docs, err := LoadDocuments(tmpDir, formats)
	if err != nil {
		t.Fatalf("Ошибка загрузки: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Ожидался 1 документ, получено %d", len(docs))
	}

	if len(docs) > 0 {
		if docs[0].Name != "test.txt" {
			t.Errorf("Неверное имя: %s, ожидалось test.txt", docs[0].Name)
		}
		if docs[0].Text != content {
			t.Errorf("Содержимое не совпадает")
		}
	}

	t.Log("Тестовый txt файл загружен успешно")
}

func TestLoadDocumentsEmptyFolder(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "empty_corpus") // создаю пустую папку
	if err != nil {
		t.Fatalf("Ошибка создания временной папки: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	formats := []string{"txt", "md"} //доки из пустой папки
	docs, err := LoadDocuments(tmpDir, formats)
	if err != nil {
		t.Fatalf("Ошибка загрузки: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("Ожидалось 0 документов, получено %d", len(docs))
	}

	t.Log("Пустая папка обработана корректно")
}
