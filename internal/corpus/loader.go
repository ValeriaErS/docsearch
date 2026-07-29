package corpus

import (
	"fmt"
	"github.com/ledongthuc/pdf"
	"os"
	"path/filepath"
	"strings"
)

func readPDF(path string) (string, map[int]string, []int, error) { // читаю PDF файл и достаю из него текст
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", nil, nil, err
	}
	defer file.Close()

	var fullText strings.Builder
	pages := make(map[int]string)
	var pageOffsets []int
	offset:=0

	for i := 1; i <= reader.NumPage(); i++ { // прохожу по всем страницам
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		pageOffsets=append(pageOffsets,offset)  // запоминаю позицию начала страницы в общем тексте
		pages[i] = content
		fullText.WriteString(content)
		fullText.WriteString("\n")
		offset+=len(content)+1
	}
	return fullText.String(), pages, pageOffsets, nil
}

func LoadDocuments(path string, formats []string) ([]Document, error) { //formats как параметр
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	validExts := make(map[string]bool) // map для быстрой проверки
	for _, f := range formats {
		validExts["."+f] = true
	}
	var docs []Document
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		ext := filepath.Ext(name)

		if !validExts[ext] { // расширение по списку из конфига
			continue
		}

		fullPath := filepath.Join(path, name)
		var text string
		var pages map[int]string
		var pageOffsets []int

		if ext == ".pdf" {
			 text, pages, pageOffsets, err = readPDF(fullPath)
			if err != nil {
				fmt.Printf("Ошибка чтения PDF %s: %v\n", name, err)
				continue
			}
			fmt.Printf("Документ %s: %d страниц\n", name, len(pages))
		} else {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				fmt.Printf("Ошибка чтения файла %s: %v\n", name, err)
				continue
			}

			text = string(data)
			pages = nil
			pageOffsets=nil
		}

		doc := Document{ // создаю документ и нормализую текст
			Name:  name,
			Text:  NormalizeNext(text),
			Pages: pages,
			PageOffsets: pageOffsets,
		}
		docs = append(docs, doc)

	}
	return docs, nil
}
