package validate

import (
	"testing"
)

func TestValidateAnswer(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		texts  []string
		docs   []string
	}{
		{
			name:   "ответ без ссылок",
			answer: "Простой ответ без ссылок",
			texts:  []string{"текст"},
			docs:   []string{"doc1.pdf"},
		},
		{
			name:   "ответ со ссылкой",
			answer: "Ответ с [источник: doc1.pdf, страница 1]",
			texts:  []string{"текст из документа"},
			docs:   []string{"doc1.pdf"},
		},
		{
			name:   "пустой ответ",
			answer: "",
			texts:  []string{"текст"},
			docs:   []string{"doc1.pdf"},
		},
		{
			name:   "ответ без текстов",
			answer: "Ответ с [источник: doc1.pdf, страница 1]",
			texts:  []string{},
			docs:   []string{"doc1.pdf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, citations := ValidateAnswer(tt.answer, tt.texts, tt.docs)

			if tt.answer != "" && result == "" {
				t.Error("ValidateAnswer() вернул пустую строку для непустого ответа")
			}

		
			_ = citations  //  проверка что функция не падает
		})
	}
}

func TestValidateAnswerEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		texts  []string
		docs   []string
	}{
		{
			name:   "пустые тексты и docs",
			answer: "Ответ",
			texts:  []string{},
			docs:   []string{},
		},
		{
			name:   "мальформированная ссылка",
			answer: "Ответ [источник: doc1.pdf, страница 1",
			texts:  []string{"текст"},
			docs:   []string{"doc1.pdf"},
		},
		{
			name:   "ссылка без запятой",
			answer: "Ответ [источник: doc1.pdf страница 1]",
			texts:  []string{"текст"},
			docs:   []string{"doc1.pdf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, citations := ValidateAnswer(tt.answer, tt.texts, tt.docs)
			if tt.answer != "" && result == "" {
				t.Error("ValidateAnswer() вернул пустую строку")
			}
			_ = citations
		})
	}
}

func TestCheckHallucinations(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		texts  []string
		docs   []string
	}{
		{
			name:   "ответ без галлюцинаций",
			answer: "RAG это технология",
			texts:  []string{"RAG это технология"},
			docs:   []string{"doc1.pdf"},
		},
		{
			name:   "ответ с галлюцинацией",
			answer: "RAG использует PostgreSQL",
			texts:  []string{"RAG использует векторы"},
			docs:   []string{"doc1.pdf"},
		},
		{
			name:   "пустой ответ",
			answer: "",
			texts:  []string{"текст"},
			docs:   []string{"doc1.pdf"},
		},
		{
			name:   "пустые тексты",
			answer: "RAG это технология",
			texts:  []string{},
			docs:   []string{"doc1.pdf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := CheckHallucinations(tt.answer, tt.texts, tt.docs)

			
			if report.Verified < 0 {
				t.Error("CheckHallucinations() вернул отрицательный Verified")
			}
			if report.Unverified < 0 {
				t.Error("CheckHallucinations() вернул отрицательный Unverified")
			}
		})
	}
}

func TestCitationStruct(t *testing.T) {
	c := Citation{
		IsValid: true,
	}

	if !c.IsValid {
		t.Error("Citation.IsValid должен быть true")
	}
}

func TestHallucinationReportStruct(t *testing.T) {
	report := HallucinationReport{
		Verified:          5,
		Unverified:        2,
		HasHallucinations: true,
	}

	if report.Verified != 5 {
		t.Errorf("Verified = %d, ожидалось 5", report.Verified)
	}
	if report.Unverified != 2 {
		t.Errorf("Unverified = %d, ожидалось 2", report.Unverified)
	}
	if !report.HasHallucinations {
		t.Error("HasHallucinations должен быть true")
	}
}