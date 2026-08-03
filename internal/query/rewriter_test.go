package query

import (
	"context"
	"docsearch/internal/config"
	"testing"
)

func TestQueryRewriterInMockMode(t *testing.T) { // проверяет работу в тестовом режиме
	// Создаем конфиг с mock-режимом
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	rewriter := NewQueryRewriter(cfg) // создаю рерайтер

	query := "Как это работает?"
	result, err := rewriter.Rewrite(context.Background(), query, []map[string]string{})  //должен возвращать оригинал
	if err != nil {
		t.Errorf("Ошибка: %v", err)
	}
	if result != query {
		t.Errorf("В mock-режиме ожидался '%s', получено '%s'", query, result)
	}

	hyde, err := rewriter.GenerateHyDE(context.Background(), query) //  HyDE должен возвращать оригинал
	if err != nil {
		t.Errorf("Ошибка HyDE: %v", err)
	}
	if hyde != query {
		t.Errorf("В mock-режиме HyDE ожидался '%s', получено '%s'", query, hyde)
	}

	t.Log("Mock-режим работает правильно")
}

func TestRewriteWithHistory(t *testing.T) {  // проверяет что история не ломает работу
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	rewriter := NewQueryRewriter(cfg)

	history := []map[string]string{
		{"role": "user", "content": "Что такое FileAuditor?"},
		{"role": "assistant", "content": "FileAuditor - это система для аудита файлов"},
	}

	query := "А как его настроить?"
	result, err := rewriter.Rewrite(context.Background(), query, history)
	if err != nil {
		t.Errorf("Ошибка с историей: %v", err)
	}
	if result != query {
		t.Errorf("В mock-режиме с историей ожидался '%s', получено '%s'", query, result)
	}

	t.Log("История диалога обрабатывается корректно")
}