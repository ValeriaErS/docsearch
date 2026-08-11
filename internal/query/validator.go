package query

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxQueryLength = 2000 //ограничение запроса

type ValidationStatus string

const (
	StatusValid     ValidationStatus = "valid"
	StatusInvalid   ValidationStatus = "invalid"
	StatusUncertain ValidationStatus = "uncertain"
)

type ValidationResult struct {
	Status ValidationStatus `json:"status"`
	Reason string           `json:"reason,omitempty"`
}

type QueryValidator struct{}

func NewQueryValidator() *QueryValidator {
	return &QueryValidator{}
}

func (v *QueryValidator) Validate(query string) *ValidationResult {
	query = strings.TrimSpace(query)

	if utf8.RuneCountInString(query) > MaxQueryLength {
		return &ValidationResult{
			Status: StatusInvalid,
			Reason: "Запрос слишком длинный. Сформулируйте вопрос короче (максимум 2000 символов).",
		}
	}

	if utf8.RuneCountInString(query) < 3 {
		return &ValidationResult{
			Status: StatusInvalid,
			Reason: "Запрос слишком короткий. Пожалуйста, задайте более конкретный вопрос.",
		}
	}

	if isOnlySpecialChars(query) { 
		return &ValidationResult{
			Status: StatusInvalid,
			Reason: "Запрос содержит только специальные символы. Пожалуйста, задайте вопрос.",
		}
	}

	if hasTooManyRepeats(query) {
		return &ValidationResult{
			Status: StatusInvalid,
			Reason: "Запрос содержит слишком много повторяющихся символов. Попробуйте сформулировать вопрос понятнее.",
		}
	}

	if isKeyboardMash(query) {
		return &ValidationResult{
			Status: StatusInvalid,
			Reason: "Запрос выглядит как случайный набор символов. Пожалуйста, задайте осмысленный вопрос.",
		}
	}

	return &ValidationResult{
		Status: StatusUncertain,
		Reason: "Запрос требует дополнительной проверки.",
	}
}

func isOnlySpecialChars(query string) bool {
	for _, ch := range query {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

func hasTooManyRepeats(query string) bool {
	runes := []rune(strings.ToLower(query))

	if len(runes) < 4 {
		return false
	}

	repeatCount := 1

	for i := 1; i < len(runes); i++ {
		if !unicode.IsLetter(runes[i]) || !unicode.IsLetter(runes[i-1]) {
			repeatCount = 1
			continue
		}

		if runes[i] == runes[i-1] {
			repeatCount++
			if repeatCount >= 5 {
				return true
			}
		} else {
			repeatCount = 1
		}
	}

	return false
}

func isKeyboardMash(query string) bool {
	query = strings.ToLower(query)

	keyboardSequences := []string{
		"qwerty", "qwertyui", "asdfgh", "asdfghj", "zxcvbn",
		"йцукен", "йцукенг", "фывапр", "фывапрол", "ячсмить",
		"фывфыв", "дждждж", "рпавыпрва", "ываываыва",
	}

	for _, sequence := range keyboardSequences {
		if strings.Contains(query, sequence) {
			return true
		}
	}

	return false
}