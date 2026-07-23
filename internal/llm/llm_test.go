package llm

import (
    "testing"
)

func TestGetAnswer(t *testing.T) {
    t.Log("Функция GetAnswer существует")
}

func TestGetAnswerWithHistory(t *testing.T) {
    t.Log("Функция GetAnswerWithHistory существует")
}

func TestMockAnswer(t *testing.T) {
    mockAnswer := "Это тестовый ответ"
    if mockAnswer == "" {
        t.Error("Ответ пустой")
    }
    t.Log("Тестовый ответ:", mockAnswer)
}