package db

import "errors"

type MockDB struct {
    Users map[string]string
    ShouldFail bool
}

func NewMockDB() *MockDB {
    return &MockDB{
        Users: make(map[string]string),
    }
}

func (m *MockDB) Close() error {
    if m.ShouldFail {
        return errors.New("close error")
    }
    return nil
}

func (m *MockDB) CheckUser(username, password string) bool {
    if m.ShouldFail {
        return false
    }
    hashed, exists := m.Users[username]
    if !exists {
        return false
    }
    return hashed == password
}

func (m *MockDB) AddUser(username, password string) error {
    if m.ShouldFail {
        return errors.New("add user error")
    }
    m.Users[username] = password
    return nil
}