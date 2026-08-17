package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type TelegramBot struct {
	Token  string
	ChatID string
}

func NewTelegramBot(token, chatID string) *TelegramBot {
	return &TelegramBot{
		Token:  token,
		ChatID: chatID,
	}
}

func (t *TelegramBot) Send(text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token)

	data := map[string]string{
		"chat_id": t.ChatID,
		"text":    text,
	}

	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram error: %d", resp.StatusCode)
	}

	return nil
}