package main

import (
	"sync"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"docsearch/internal/auth"
	"docsearch/internal/config"
	"docsearch/internal/db"
	"docsearch/internal/rag"
	"path/filepath"
)
func sanitizeUsername(username string) string{
	re:=regexp.MustCompile(`[^a-zA-Zа-яА-Я0-9_ ]`)
	return re.ReplaceAllString(username,"")
}
func makeSafeUserDir(username string) (string, error) {  //безопасен ли путь
    safeName := sanitizeUsername(username)
    if safeName == "" {
        return "", fmt.Errorf("пустое имя")
    }

    fullPath := filepath.Join("docs", safeName)

    cleanPath := filepath.Clean(fullPath)
    if !strings.HasPrefix(cleanPath, "docs/") {
        return "", fmt.Errorf("небезопасное имя пользователя")
    }
    
    return cleanPath, nil
}


var chatHistory = make(map[string][]map[string]string)
var chatMutex sync.RWMutex
var database *db.DB
var globalCfg *config.Config

func runWeb(cfg *config.Config, port string) {
	globalCfg = cfg
	
	var err error
	database, err = db.NewDB()
	if err != nil {
		fmt.Println("Ошибка базы:", err)
		return
	}
	defer database.Close()

	http.HandleFunc("/", showIndex) // страницы
	http.HandleFunc("/chat.html", showChat)
	http.HandleFunc("/test.html", showTest)
	http.HandleFunc("/login.html", showLogin)
	http.HandleFunc("/register.html", showRegister)

	http.HandleFunc("/login", handleLogin) // обработчики
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/ask", handleAsk)
	

	fmt.Println("Сайт запущен: http://localhost" + port)
	http.ListenAndServe("0.0.0.0"+port, nil)
}

func showIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func showChat(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/chat.html")
}

func showTest(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/test.html")
}

func showLogin(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/login.html")
}

func showRegister(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/register.html")
}

func handleLogin(w http.ResponseWriter, r *http.Request) { //обработчик вход
	if r.Method != "POST" {
		http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Ошибка чтения", http.StatusBadRequest)
		return
	}

	ok := database.CheckUser(req.Username, req.Password)

	if ok {
		token, err := auth.MakeToken(req.Username)
		if err != nil {
			http.Error(w, "Ошибка создания токена", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"user": req.Username,
			"token":token,
		})
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error": "Неверный логин или пароль",
		})
	}
}

func handleRegister(w http.ResponseWriter, r *http.Request) { // обработчик регистрации
	if r.Method != "POST" {
		http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Ошибка чтения", http.StatusBadRequest)
		return
	}
	safeUsername := sanitizeUsername(req.Username)
    if safeUsername == "" {
        http.Error(w, "Некорректное имя пользователя", http.StatusBadRequest)
        return
    }

	if len(req.Password) < 6 {
		http.Error(w, "Пароль должен быть не менее 6 символов", http.StatusBadRequest)
		return
	}

	weakPasswords := []string{"123456", "password", "qwerty", "111111", "123123", "admin", "letmein", "555555", "000000", "12345"}
	for _, wp := range weakPasswords {
		if req.Password == wp {
			http.Error(w, "Слишком простой пароль", http.StatusBadRequest)
			return
		}
	}

    err = database.AddUser(safeUsername, req.Password)
	if err != nil {
		http.Error(w, "Пользователь уже существует", http.StatusConflict)
		return
	}

    userDir, err := makeSafeUserDir(req.Username)
    if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
    os.MkdirAll(userDir, 0755)
    fmt.Println("Папка создана:", userDir)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user": safeUsername,
	})
}

func handleAsk(w http.ResponseWriter, r *http.Request) { //обработчик вопрос
	authHeader := r.Header.Get("Authorization") // проверяю токен
	if authHeader == "" {
		http.Error(w, "Нет токена", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	username, err := auth.CheckToken(tokenString)
	if err != nil {
		http.Error(w, "Неверный токен", http.StatusUnauthorized)
		return
	}
	fmt.Println("Пользователь из токена:", username)

	if r.Method != "POST" {
		http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query string `json:"query"`
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Ошибка чтения", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Пустой вопрос", http.StatusBadRequest)
		return
	}

	userID := username
    
	chatMutex.Lock()
	if chatHistory[userID] == nil {
		chatHistory[userID] = []map[string]string{}
	}
    chatHistory[userID] = append(chatHistory[userID], map[string]string{
    "role":"user",
    "content": req.Query,
})
chatMutex.Unlock()

	texts, docs, scores, answer, pages, timings := rag.Ask(*globalCfg, req.Query, userID)

	sources := []map[string]interface{}{}
	for i := 0; i < len(texts); i++ {
		sources = append(sources, map[string]interface{}{
			"doc_id": docs[i],
			"score": scores[i],
			"page": pages[i],
		})
	}

	chatMutex.Lock()
    chatHistory[userID] = append(chatHistory[userID], map[string]string{
    "role": "assistant",
    "content": answer,
})
chatMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"answer": answer,
		"sources": sources,
		"timings": timings,
	})
}