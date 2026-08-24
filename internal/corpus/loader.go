package corpus

import (
	"fmt"
	"github.com/ledongthuc/pdf"
	"os"
	"path/filepath"
	"strings"
	"docsearch/internal/parser"
)

func readPDF(path string) (string, map[int]string, []int, error) {
    fmt.Printf("Пробую Docling для %s...\n", path)
    
    result, err := parser.ParsePDFDocling(path)
    if err == nil && result != nil && len(result.Text) > 100 {
        fmt.Printf("Docling прочитал %d страниц, %d символов\n", result.Pages, len(result.Text))
        
        pages := make(map[int]string) //разбиваю текст на стр
        pageOffsets := []int{}
        
        var pageTexts []string
        if strings.Contains(result.Text, "\f") {
            pageTexts = strings.Split(result.Text, "\f")
        } else {
            pageTexts = strings.Split(result.Text, "\n\n")
        }
        
        offset := 0
        for i, pageText := range pageTexts {
            pageNum := i + 1
            if pageNum > result.Pages && result.Pages > 0 {
                break
            }
            pageText = strings.TrimSpace(pageText)
            if len(pageText) > 10 {
                pages[pageNum] = pageText
                pageOffsets = append(pageOffsets, offset)
                offset += len(pageText) + 1
            }
        }
        
        if len(pages) == 0 {
            pages[1] = result.Text
            pageOffsets = []int{0}
        }
        
        return result.Text, pages, pageOffsets, nil
    }
    
    if err != nil {
        fmt.Printf("Docling не сработал: %v, пробую fallback\n", err)
    } else {
        fmt.Printf("Docling вернул пустой текст, пробую fallback\n")
    }
   
    fmt.Printf("Пробую старый парсер для %s...\n", path)
    
    file, reader, err := pdf.Open(path)
    if err != nil {
        return "", nil, nil, fmt.Errorf("не удалось открыть PDF: %w", err)
    }
    defer file.Close()

    var fullText strings.Builder
    pages := make(map[int]string)
    var pageOffsets []int
    offset := 0

    for pageNum := 1; pageNum <= reader.NumPage(); pageNum++ {
        page := reader.Page(pageNum)
        if page.V.IsNull() {
            continue
        }
        content, err := page.GetPlainText(nil)
        if err != nil {
            continue
        }
        content = strings.TrimSpace(content)
        if len(content) > 0 {
            pageOffsets = append(pageOffsets, offset)
            pages[pageNum] = content
            fullText.WriteString(content)
            fullText.WriteString("\n")
            offset += len(content) + 1
        }
    }

    finalText := fullText.String()
    if len(finalText) == 0 {
        return "", nil, nil, fmt.Errorf("PDF не содержит текста (возможно, это скан)")
    }
    
    fmt.Printf("Старый парсер прочитал %d страниц, %d символов\n", len(pages), len(finalText))
    return finalText, pages, pageOffsets, nil
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
			if ext == ".html" {
				text = RemoveHTMLTags(string(data))
				text = NormalizeNext(text)
				fmt.Printf("HTML очищен: %d символов\n", len(text))
				fmt.Printf("Текст: %s\n", text)
				pages = nil
				pageOffsets = nil
				} else {
					text = string(data)
				}
		}

		doc := Document{ // создаю документ и нормализую текст
			Name:        name,
			Text:        NormalizeNext(text),
			Pages:       pages,
			PageOffsets: pageOffsets,
		}
		docs = append(docs, doc)

	}
	return docs, nil
}
func ReadPDFFile(path string) (string, map[int]string, []int, error) {
    return readPDF(path)
}
