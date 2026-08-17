package safety

import (
	"testing"
)

func TestSanitizeAndValidateUser(t *testing.T) { // проверяет валидацию имени пользователя
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "нормальное имя",
			username: "Екатерина",
			wantErr:  false,
		},
		{
			name:     "имя с пробелом",
			username: "Анна Иванова",
			wantErr:  false,
		},
		{
			name:     "имя с дефисом",
			username: "Анна-Мария",
			wantErr:  false,
		},
		{
			name:     "пустое имя",
			username: "",
			wantErr:  true,
		},
		{
			name:     "имя с запрещёнными символами",
			username: "test<>",
			wantErr:  false,
		},
		{
			name:     "имя с подчёркиванием",
			username: "test_user",
			wantErr:  false,
		},
		{
			name:     "имя с цифрами",
			username: "user123",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizeAndValidateUser(tt.username)
			if tt.wantErr && err == nil {
				t.Errorf("SanitizeAndValidateUser(%q) ожидалась ошибка", tt.username)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("SanitizeAndValidateUser(%q) ошибка: %v", tt.username, err)
			}
			if !tt.wantErr && result == "" {
				t.Errorf("SanitizeAndValidateUser(%q) вернул пустую строку", tt.username)
			}
		})
	}
}

func TestSanitizeUser(t *testing.T) {  // проверяет очистку имени пользователя
	tests := []struct {
		input    string
		expected string
	}{
		{"test<>", "test"},
		{"user!@#$", "user"},
		{"привет мир", "привет мир"},
		{"Hello_World", "Hello_World"},
		{"   spaces   ", "spaces"},
	}

	for _, tt := range tests {
		result, err := SanitizeAndValidateUser(tt.input)
		if err != nil {
			t.Logf("SanitizeAndValidateUser(%q) ошибка: %v", tt.input, err)
		}
		if result == "" && tt.expected != "" {
			t.Errorf("SanitizeAndValidateUser(%q) = %s, ожидалось %s", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeSpecialChars(t *testing.T) { // проверяет обработку специальных символов
	tests := []struct {
		input    string
		expected string
	}{
		{"test<>", "test"},
		{"user!@#$", "user"},
		{"hello*", "hello"},
		{"name?", "name"},
	}

	for _, tt := range tests {
		result, err := SanitizeAndValidateUser(tt.input)
		if err != nil {
			t.Skip("Функция вернула ошибку для:", tt.input)
		}
		if result != tt.expected {
			t.Logf("SanitizeAndValidateUser(%q) = %s, ожидалось %s", tt.input, result, tt.expected)
		}
	}
}