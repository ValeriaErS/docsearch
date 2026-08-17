package alert

import (
	"testing"
)

func TestNewTelegramBot(t *testing.T) { //создан ли бот
	bot := NewTelegramBot("test_token", "test_chat_id")

	if bot == nil {
		t.Error("NewTelegramBot() вернул nil")
	}
	if bot.Token != "test_token" {
		t.Errorf("Token = %s, ожидалось test_token", bot.Token)
	}
	if bot.ChatID != "test_chat_id" {
		t.Errorf("ChatID = %s, ожидалось test_chat_id", bot.ChatID)
	}
}

func TestNewTelegramBotEmpty(t *testing.T) {  // проверяет создание бота с пустыми значениями
	bot := NewTelegramBot("", "")

	if bot == nil {
		t.Error("NewTelegramBot() с пустыми значениями вернул nil")
	}
	if bot.Token != "" {
		t.Errorf("Token = %s, ожидалась пустая строка", bot.Token)
	}
	if bot.ChatID != "" {
		t.Errorf("ChatID = %s, ожидалась пустая строка", bot.ChatID)
	}
}

func TestTelegramBotSendEmpty(t *testing.T) {
	bot := NewTelegramBot("token", "chat_id")

	err := bot.Send("")
	if err == nil {
		t.Log("Send() с пустым текстом вернул nil (зависит от реализации)")
	}
}

func TestTelegramBotSendNoPanic(t *testing.T) {
	bot := NewTelegramBot("test_token", "test_chat_id")

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Send() вызвал панику: %v", r)
		}
	}()

	bot.Send("test message")
}