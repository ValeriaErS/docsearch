package db

import (
    "database/sql"
    "fmt"
    "os"
    _ "github.com/lib/pq"
    "github.com/joho/godotenv"
)

type DB struct {
    Conn *sql.DB
}

func NewDB() (*DB, error) {
    godotenv.Load()

    connStr := os.Getenv("DATABASE_URL")
    if connStr == "" {
        return nil, fmt.Errorf("нет ссылки на базу")
    }

    conn, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }

    err = conn.Ping()
    if err != nil {
        return nil, err
    }

    fmt.Println("База работает")
    return &DB{Conn: conn}, nil
}

func (d *DB) Close() {
    d.Conn.Close()
}

func (d *DB) CheckUser(username, password string) bool {
    var dbPassword string

    fmt.Println("Ищу пользователя:", username)

    
    err := d.Conn.QueryRow("SELECT password FROM users WHERE username = $1", username).Scan(&dbPassword)  // Ищем пароль в базе

    if err != nil {
        fmt.Println("Ошибка запроса:", err)
        return false
    }

    fmt.Println("Пароль в базе:", dbPassword)
    fmt.Println("Пароль введённый:", password)

    if dbPassword != password {
        fmt.Println("Пароли НЕ совпадают")
        return false
    }

    fmt.Println("Пароли совпадают")
    return true
}

func (d *DB) AddUser(username, password string) error {
    _, err := d.Conn.Exec(
        "INSERT INTO users (username, password) VALUES ($1, $2)",
        username, password,
    )
    return err
}