package server

import (
	"context"
	"docsearch/internal/auth"
	"docsearch/internal/config"
	"docsearch/internal/db"
	"docsearch/internal/rag"
	"docsearch/internal/safety"
	"docsearch/internal/vector"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"net"
)
func getClientIP(r *http.Request) string{
	host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return host
}

const maxHistorySize = 50

var loginAttempts = make(map[string]int)
var loginBlocked = make(map[string]time.Time)
var loginMutex sync.Mutex

var chatHistory = make(map[string][]map[string]string)
var chatMutex sync.RWMutex
var database *db.DB
var globalCfg *config.Config
var vectorClientGlobal vector.VectorStore

func RunWeb(cfg *config.Config, port string, vectorClient vector.VectorStore) {
	globalCfg = cfg
	vectorClientGlobal = vectorClient

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
	http.HandleFunc("/health", handleHealth)

	srv := &http.Server{
		Addr:         "0.0.0.0" + port,
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		fmt.Println("Сайт запущен: http://localhost" + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("Ошибка сервера:", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	fmt.Println("Завершаем сервер...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Println("Ошибка завершения:", err)
	}
	if err := database.Close(); err != nil {
		fmt.Println("Ошибка закрытия БД:", err)
	}

	fmt.Println("Сервер остановлен")
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
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // ограничиваюразмер тела запроса (1 MB)
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Ошибка чтения запроса",
		})
		return
	}
	safeUsername, err := safety.SanitizeAndValidateUser(req.Username)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Некорректное имя пользователя",
		})
		return
	}
	clientIP := getClientIP(r)
	key := clientIP + "_" + safeUsername

	loginMutex.Lock()
	if blockTime, exists := loginBlocked[key]; exists {
		if time.Now().Before(blockTime) {
			loginMutex.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error": "Слишком много попыток. Попробуйте через 5 минут",
			})
			return
		}
		delete(loginBlocked, key)
		delete(loginAttempts, key)
	}
	loginMutex.Unlock()

	ok := database.CheckUser(safeUsername, req.Password)

	if ok {
		loginMutex.Lock() //успешный-сброс
		delete(loginAttempts, key)
		delete(loginBlocked, key)
		loginMutex.Unlock()

		token, err := auth.MakeToken(safeUsername)
		if err != nil {
			http.Error(w, "Ошибка создания токена", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"user":    req.Username,
			"token":   token,
		})
	} else {
		loginMutex.Lock() //неудачный вход
		loginAttempts[key]++
		if loginAttempts[key] >= 5 {
			loginBlocked[key] = time.Now().Add(5 * time.Minute)
		}
		loginMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Неверный логин или пароль",
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
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Ошибка чтения запроса",
		})
		return
	}

	safeUsername, err := safety.SanitizeAndValidateUser(req.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	userDir := filepath.Join("docs", safeUsername)
	os.MkdirAll(userDir, 0755)
	fmt.Println("Папка создана:", userDir)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    safeUsername,
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
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Ошибка чтения запроса",
		})
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
		"role":    "user",
		"content": req.Query,
	})
	chatMutex.Unlock()

	chatMutex.Lock()
	if len(chatHistory[userID]) > maxHistorySize {
		chatHistory[userID] = chatHistory[userID][len(chatHistory[userID])-maxHistorySize:]

	}
	chatMutex.Unlock()

	chatMutex.RLock()
	history := chatHistory[userID]
	chatMutex.RUnlock()

	texts, docs, scores, answer, pages, chunkIDs, _, timings := rag.Ask(r.Context(), *globalCfg, req.Query, userID, history, vectorClientGlobal)
	sources := []map[string]interface{}{}
	for i := 0; i < len(texts); i++ {
		sources = append(sources, map[string]interface{}{
			"doc_id":   docs[i],
			"score":    scores[i],
			"page":     pages[i],
			"chunk_id": chunkIDs[i],
		})
	}

	chatMutex.Lock()
	chatHistory[userID] = append(chatHistory[userID], map[string]string{
		"role":    "assistant",
		"content": answer,
	})
	chatMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"answer":  answer,
		"sources": sources,
		"timings": timings,
	})
}
func handleHealth(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization") //проверка токена
	if authHeader == "" {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	_, err := auth.CheckToken(tokenString)
	if err != nil {
		http.Error(w, "Неверный токен", http.StatusUnauthorized)
		return
	}

	client, err := vector.NewQdrantClient() //проверка бд
	if err != nil {
		http.Error(w, "Qdrant недоступен", http.StatusServiceUnavailable)
		return
	}
	if err := client.Ping(context.Background()); err != nil {
		http.Error(w, "Qdrant не отвечает", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"qdrant": "connected",
	})
}
