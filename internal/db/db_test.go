package db

import (
    "os"
    "testing"
)
func TestNewDBNoURL(t *testing.T) {   // проверяет создание подключения к бд 
    original := os.Getenv("DATABASE_URL")
    defer os.Setenv("DATABASE_URL", original)

    os.Unsetenv("DATABASE_URL")

    db, err := NewDB()
    if err == nil {
        t.Error("NewDB() без DATABASE_URL должен вернуть ошибку")
    }
    if db != nil {
        t.Error("NewDB() без DATABASE_URL вернул не nil подключение")
    }
}
func TestDBStruct(t *testing.T) {  // проверяет структуру 
    db := &DB{Conn: nil}
    if db == nil {
        t.Error("DB не создался")
    }
    if db.Conn != nil {
        t.Error("DB.Conn должен быть nil")
    }
}
func TestNewMockDB(t *testing.T) {  // проверяет создание мока
    mock := NewMockDB()
    if mock == nil {
        t.Error("NewMockDB() вернул nil")
    }
    if mock.Users == nil {
        t.Error("Users не инициализирован")
    }
}
func TestMockDBAddUser(t *testing.T) {
    mock := NewMockDB()

    err := mock.AddUser("testuser", "testpass")
    if err != nil {
        t.Errorf("AddUser() ошибка: %v", err)
    }

    if len(mock.Users) != 1 {
        t.Errorf("Users = %d, ожидалось 1", len(mock.Users))
    }

    if mock.Users["testuser"] != "testpass" {
        t.Errorf("Users[testuser] = %s, ожидалось testpass", mock.Users["testuser"])
    }
}
func TestMockDBAddUserError(t *testing.T) { //проверка юзера
    mock := NewMockDB()
    mock.ShouldFail = true

    err := mock.AddUser("testuser", "testpass")
    if err == nil {
        t.Error("AddUser() с ShouldFail должен вернуть ошибку")
    }
}
func TestMockDBCheckUser(t *testing.T) {
    mock := NewMockDB()
    mock.AddUser("testuser", "testpass")

    result := mock.CheckUser("testuser", "testpass")
    if !result {
        t.Error("CheckUser() с правильным паролем вернул false")
    }

    result = mock.CheckUser("testuser", "wrongpass")
    if result {
        t.Error("CheckUser() с неправильным паролем вернул true")
    }

    result = mock.CheckUser("unknown", "testpass")
    if result {
        t.Error("CheckUser() с несуществующим пользователем вернул true")
    }
}

func TestMockDBCheckUserError(t *testing.T) {
    mock := NewMockDB()
    mock.ShouldFail = true

    result := mock.CheckUser("testuser", "testpass")
    if result {
        t.Error("CheckUser() с ShouldFail должен вернуть false")
    }
}

func TestMockDBClose(t *testing.T) {
    mock := NewMockDB()

    err := mock.Close()
    if err != nil {
        t.Errorf("Close() ошибка: %v", err)
    }
}

func TestMockDBCloseError(t *testing.T) {
    mock := NewMockDB()
    mock.ShouldFail = true

    err := mock.Close()
    if err == nil {
        t.Error("Close() с ShouldFail должен вернуть ошибку")
    }
}