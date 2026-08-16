package verify

import (
	"context"
	"docsearch/internal/config"
	"testing"
)

func TestVerifyAnswerWithEmptyChunks(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	ctx := context.Background()

	result, err := VerifyAnswer(ctx, "вопрос", "ответ", []string{}, cfg)

	if err != nil {
		t.Errorf("VerifyAnswer() ошибка: %v", err)
	}
	if result == nil {
		t.Fatal("VerifyAnswer() вернул nil")
	}
	if result.IsAccurate {
		t.Error("IsAccurate должен быть false при пустых чанках")
	}
	if result.Reason != "Нет контекста или ответа" {
		t.Errorf("Reason = %s, ожидалось 'Нет контекста или ответа'", result.Reason)
	}
	if result.Confidence != 0 {
		t.Errorf("Confidence = %f, ожидалось 0", result.Confidence)
	}
}

func TestVerifyAnswerWithEmptyAnswer(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	ctx := context.Background()

	result, err := VerifyAnswer(ctx, "вопрос", "", []string{"контекст"}, cfg)

	if err != nil {
		t.Errorf("VerifyAnswer() ошибка: %v", err)
	}
	if result == nil {
		t.Fatal("VerifyAnswer() вернул nil")
	}
	if result.IsAccurate {
		t.Error("IsAccurate должен быть false при пустом ответе")
	}
	if result.Reason != "Нет контекста или ответа" {
		t.Errorf("Reason = %s, ожидалось 'Нет контекста или ответа'", result.Reason)
	}
}

func TestVerifyAnswerWithMockProvider(t *testing.T) {  //тест для mock
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	ctx := context.Background()

	result, err := VerifyAnswer(ctx, "Что такое RAG?", "RAG это технология", []string{"RAG это Retrieval-Augmented Generation"}, cfg)

	if err != nil {
		t.Errorf("VerifyAnswer() ошибка: %v", err)
	}
	if result == nil {
		t.Fatal("VerifyAnswer() вернул nil")
	}
	if !result.IsAccurate {
		t.Error("IsAccurate должен быть true в mock-режиме")
	}
	if result.Reason != "Mock режим" {
		t.Errorf("Reason = %s, ожидалось 'Mock режим'", result.Reason)
	}
	if result.Confidence != 0.9 {
		t.Errorf("Confidence = %f, ожидалось 0.9", result.Confidence)
	}
}

func TestVerifyAnswerWithBothEmpty(t *testing.T) { // тест для структруы verificftionresult
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	ctx := context.Background()

	result, err := VerifyAnswer(ctx, "вопрос", "", []string{}, cfg)

	if err != nil {
		t.Errorf("VerifyAnswer() ошибка: %v", err)
	}
	if result == nil {
		t.Fatal("VerifyAnswer() вернул nil")
	}
	if result.IsAccurate {
		t.Error("IsAccurate должен быть false")
	}
	if result.Confidence != 0 {
		t.Errorf("Confidence = %f, ожидалось 0", result.Confidence)
	}
}

func TestVerifyAnswerWithNilConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	ctx := context.Background()

	result, err := VerifyAnswer(ctx, "вопрос", "ответ", []string{"контекст"}, cfg)

	if err != nil {
		t.Errorf("VerifyAnswer() ошибка: %v", err)
	}
	if result == nil {
		t.Error("VerifyAnswer() вернул nil")
	}
}

func TestVerificationResultStruct(t *testing.T) {
	tests := []struct {
		name        string
		isAccurate  bool
		reason      string
		confidence  float64
		fixedAnswer string
	}{
		{
			name:        "успешная проверка",
			isAccurate:  true,
			reason:      "Ответ верный",
			confidence:  0.95,
			fixedAnswer: "",
		},
		{
			name:        "ошибка в ответе",
			isAccurate:  false,
			reason:      "Ответ не соответствует контексту",
			confidence:  0.1,
			fixedAnswer: "Исправленный ответ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerificationResult{
				IsAccurate:  tt.isAccurate,
				Reason:      tt.reason,
				Confidence:  tt.confidence,
				FixedAnswer: tt.fixedAnswer,
			}

			if result.IsAccurate != tt.isAccurate {
				t.Errorf("IsAccurate = %v, ожидалось %v", result.IsAccurate, tt.isAccurate)
			}
			if result.Reason != tt.reason {
				t.Errorf("Reason = %s, ожидалось %s", result.Reason, tt.reason)
			}
			if result.Confidence != tt.confidence {
				t.Errorf("Confidence = %f, ожидалось %f", result.Confidence, tt.confidence)
			}
			if result.FixedAnswer != tt.fixedAnswer {
				t.Errorf("FixedAnswer = %s, ожидалось %s", result.FixedAnswer, tt.fixedAnswer)
			}
		})
	}
}

func TestVerifyAnswerWithLongContext(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	ctx := context.Background()
	longChunk := ""
	for i := 0; i < 100; i++ {
		longChunk += "RAG это технология поиска и генерации. "
	}

	result, err := VerifyAnswer(ctx, "Что такое RAG?", "RAG это технология", []string{longChunk}, cfg)

	if err != nil {
		t.Errorf("VerifyAnswer() с длинным контекстом ошибка: %v", err)
	}
	if result == nil {
		t.Error("VerifyAnswer() с длинным контекстом вернул nil")
	}
	if !result.IsAccurate {
		t.Error("IsAccurate должен быть true в mock-режиме")
	}
}

func TestVerifyAnswerWithMultipleChunks(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	ctx := context.Background()

	chunks := []string{
		"Первый чанк: RAG это Retrieval-Augmented Generation",
		"Второй чанк: RAG используется для поиска",
		"Третий чанк: RAG улучшает качество ответов",
	}

	result, err := VerifyAnswer(ctx, "Что такое RAG?", "RAG это технология", chunks, cfg)

	if err != nil {
		t.Errorf("VerifyAnswer() с несколькими чанками ошибка: %v", err)
	}
	if result == nil {
		t.Error("VerifyAnswer() с несколькими чанками вернул nil")
	}
	if !result.IsAccurate {
		t.Error("IsAccurate должен быть true в mock-режиме")
	}
}

func TestVerifyAnswerWithEmptyStringConfig(t *testing.T) {  //для парсинга
	cfg := &config.Config{}
	cfg.LLM.Provider = ""
	cfg.Verification.EnableAnswerVerification = true

	ctx := context.Background()

	result, err := VerifyAnswer(ctx, "вопрос", "ответ", []string{"контекст"}, cfg)

	if err != nil {
		t.Errorf("VerifyAnswer() с пустым провайдером ошибка: %v", err)
	}
	if result == nil {
		t.Error("VerifyAnswer() с пустым провайдером вернул nil")
	}
}
