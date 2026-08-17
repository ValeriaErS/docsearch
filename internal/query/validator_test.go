package query

import (
	"testing"
)

func TestQueryValidatorValidate(t *testing.T) {  // проверяет валидатор на разных запросах
	validator := NewQueryValidator()

	tests := []struct {
		name     string
		query    string
		expected ValidationStatus
	}{
		{
			name:     "валидный запрос",
			query:    "Как работает RAG?",
			expected: StatusUncertain,
		},
		{
			name:     "пустой запрос",
			query:    "",
			expected: StatusInvalid,
		},
		{
			name:     "очень короткий запрос",
			query:    "Hi",
			expected: StatusInvalid,
		},
		{
			name:     "запрос с пробелами",
			query:    "   ",
			expected: StatusInvalid,
		},
		{
			name:     "только специальные символы",
			query:    "!!!???",
			expected: StatusInvalid,
		},
		{
			name:     "повторяющиеся символы",
			query:    "ааааааааааааааа",
			expected: StatusInvalid,
		},
		{
			name:     "случайный набор клавиш qwerty",
			query:    "qwerty",
			expected: StatusInvalid,
		},
		{
			name:     "случайный набор клавиш йцукен",
			query:    "йцукен",
			expected: StatusInvalid,
		},
		{
			name:     "нормальный запрос с вопросом",
			query:    "Что такое эмбеддинг?",
			expected: StatusUncertain,
		},
		{
			name:     "длинный запрос",
			query:    "Как настроить подключение к базе данных в DocSearch?",
			expected: StatusUncertain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.query)
			if result.Status != tt.expected {
				t.Errorf("Validate(%q) = %v, ожидалось %v", tt.query, result.Status, tt.expected)
			}
		})
	}
}

func TestHasTooManyRepeats(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"ааааа", true},
		{"аааааааааа", true},
		{"aaa", false},
		{"abcde", false},
		{"привееееееет", true},
		{"hello", false},
		{"", false},
		{"аа", false},
		{"ааааааааааааааа", true},
	}

	for _, tt := range tests {
		result := hasTooManyRepeats(tt.query)
		if result != tt.expected {
			t.Errorf("hasTooManyRepeats(%q) = %v, ожидалось %v", tt.query, result, tt.expected)
		}
	}
}

func TestIsKeyboardMash(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"qwerty", true},
		{"йцукен", true},
		{"фывапр", true},
		{"asdfgh", true},
		{"zxcvbn", true},
		{"hello", false},
		{"world", false},
		{"привет", false},
		{"qwertyuiop", true},
		{"фывапрол", true},
		{"", false},
		{"test", false},
	}

	for _, tt := range tests {
		result := isKeyboardMash(tt.query)
		if result != tt.expected {
			t.Errorf("isKeyboardMash(%q) = %v, ожидалось %v", tt.query, result, tt.expected)
		}
	}
}

func TestValidationResultStruct(t *testing.T) {  // проверяет структуру ValidationResult
	result := &ValidationResult{
		Status: StatusValid,
		Reason: "Запрос прошёл проверку",
	}

	if result.Status != StatusValid {
		t.Errorf("Status = %v, ожидалось %v", result.Status, StatusValid)
	}
	if result.Reason != "Запрос прошёл проверку" {
		t.Errorf("Reason = %s, ожидалось 'Запрос прошёл проверку'", result.Reason)
	}
}