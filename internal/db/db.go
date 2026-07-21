package db

import (
    "database/sql"
    "fmt"
    "os"
    _ "github.com/lib/pq"
    "github.com/joho/godotenv"
    "golang.org/x/crypto/bcrypt"
)

type DB struct {
    Conn *sql.DB   // подключение к бд
}

func NewDB() (*DB, error) {
    godotenv.Load()

    connStr := os.Getenv("DATABASE_URL")
    if connStr == "" {
        return nil, fmt.Errorf("нет ссылки на базу")
    }

    conn, err := sql.Open("postgres", connStr)   // открываю соединение
    if err != nil {
        return nil, err
    }

    err = conn.Ping()   // проверяю, что база отвечает
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
    var hashedPassword string   // сюда хеш из базы

    err:=d.Conn.QueryRow("SELECT password FROM users WHERE username = $1", username).Scan(&hashedPassword)
    if err!=nil{
        fmt.Println("Пользователь не найден",err)
        return false
    }
    err=bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
    if err!=nil{
        fmt.Println("Пароли не совпадают",err)
        return false
    }
    fmt.Println("Пароли совпадают!")
    return true
}


func (d *DB) AddUser(username, password string) error {
    hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)  // Хеширую пароль
    if err!=nil{
        return err
    }

    _, err = d.Conn.Exec(
        "INSERT INTO users (username, password) VALUES ($1, $2)",
        username, string(hashed),
    )
    return err
}