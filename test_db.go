package main

import (
    "fmt"
    "docsearch/internal/db"
)

func main() {
    fmt.Println("Проверяю подключение к Supabase")

    // Подключаемся
    database, err := db.NewDB()
    if err != nil {
        fmt.Println("Ошибка:", err)
        return
    }
    defer database.Close()

    fmt.Println("Всё работает.База данных подключена.")

	ok:=database.CheckUser("Валерия","123")
	if ok {
        fmt.Println("Вход выполнен")
    } else {
        fmt.Println("Неверный логин или пароль")
    }
}