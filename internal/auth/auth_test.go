package auth

import (
	"os"
	"testing"
)

func TestMakeToken(t *testing.T) {  // проверяет создание токена
	os.Setenv("JWT_SECRET", "test_secret_123")
	defer os.Unsetenv("JWT_SECRET")

	token, err := MakeToken("testuser")
	if err != nil {
		t.Errorf("MakeToken() ошибка: %v", err)
	}
	if token == "" {
		t.Error("MakeToken() вернул пустой токен")
	}
}

func TestMakeTokenNoSecret(t *testing.T) {  // проверяет создание токена без секрета
	os.Unsetenv("JWT_SECRET")

	token, err := MakeToken("testuser")
	if err == nil {
		t.Error("MakeToken() без секрета должен вернуть ошибку")
	}
	if token != "" {
		t.Error("MakeToken() без секрета вернул токен")
	}
}

func TestCheckToken(t *testing.T) {  // проверяет проверку токена
	os.Setenv("JWT_SECRET", "test_secret_123")
	defer os.Unsetenv("JWT_SECRET")

	token, err := MakeToken("testuser")
	if err != nil {
		t.Fatalf("MakeToken() ошибка: %v", err)
	}

	username, err := CheckToken(token)
	if err != nil {
		t.Errorf("CheckToken() ошибка: %v", err)
	}
	if username != "testuser" {
		t.Errorf("CheckToken() username = %s, ожидалось testuser", username)
	}
}

func TestCheckTokenInvalid(t *testing.T) {   // проверяет проверку неверного токена
	os.Setenv("JWT_SECRET", "test_secret_123")
	defer os.Unsetenv("JWT_SECRET")

	username, err := CheckToken("invalid.token.here")
	if err == nil {
		t.Error("CheckToken() с неверным токеном должен вернуть ошибку")
	}
	if username != "" {
		t.Error("CheckToken() с неверным токеном вернул username")
	}
}

func TestCheckTokenNoSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")

	username, err := CheckToken("some.token.here")
	if err == nil {
		t.Error("CheckToken() без секрета должен вернуть ошибку")
	}
	if username != "" {
		t.Error("CheckToken() без секрета вернул username")
	}
}

func TestCheckTokenWrongMethod(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret_123")
	defer os.Unsetenv("JWT_SECRET")

	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjoidGVzdCJ9.signature"

	username, err := CheckToken(token)
	if err == nil {
		t.Error("CheckToken() с неправильным методом подписи должен вернуть ошибку")
	}
	if username != "" {
		t.Error("CheckToken() с неправильным методом подписи вернул username")
	}
}