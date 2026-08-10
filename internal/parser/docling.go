package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type DoclingResult struct {
	Name     string              `json:"name"`
	Text     string              `json:"text"`
	Tables   []DoclingTable      `json:"tables"`
	Sections []DoclingSection    `json:"sections"`
	Pages    int                 `json:"pages"`
	Metadata map[string]string   `json:"metadata"`
}

type DoclingTable struct {
	Markdown string `json:"markdown"`
	Page     int    `json:"page"`
	Caption  string `json:"caption"`
}

type DoclingSection struct {
	Title string `json:"title"`
	Level int    `json:"level"`
	Page  int    `json:"page"`
}

func ParsePDFDocling(path string) (*DoclingResult, error) { //  парсит PDF с помощью Docling (Python)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("файл не найден: %w", err)
	}

	scriptPath := "scripts/parse_pdf_docling.py"
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("скрипт не найден: %s", scriptPath)
	}

	cmd := exec.Command("python", scriptPath, path)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ошибка парсинга: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("ошибка запуска: %w", err)
	}

	var result DoclingResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if result.Text == "" && len(result.Tables) == 0 {
		return nil, fmt.Errorf("парсер вернул пустой результат")
	}

	if len(result.Tables) > 0 {  // добавляю таблицы в текст
		var tableText string
		tableText += "\n\n--- TABLES ---\n"
		for i, table := range result.Tables {
			tableText += fmt.Sprintf("\n[Table %d, page %d]\n", i+1, table.Page)
			if table.Caption != "" {
				tableText += fmt.Sprintf("Caption: %s\n", table.Caption)
			}
			tableText += table.Markdown + "\n"
		}
		result.Text = result.Text + tableText
		fmt.Printf("Найдено %d таблиц в %s\n", len(result.Tables), filepath.Base(path))
	}

	if len(result.Sections) > 0 {
		fmt.Printf("Найдено %d разделов в %s\n", len(result.Sections), filepath.Base(path))
	}

	fmt.Printf("Страниц: %d\n", result.Pages)
	return &result, nil
}