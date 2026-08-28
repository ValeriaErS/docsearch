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
	"docsearch/internal/agent"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"docsearch/internal/alert"
	"docsearch/internal/llm"
	 "github.com/swaggo/http-swagger"
    _ "docsearch/docs" 
)
var telegramBot *alert.TelegramBot

var startTime = time.Now()
var rateLimiter = NewRateLimiter(30, 1*time.Minute)

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

	 token := os.Getenv("TELEGRAM_BOT_TOKEN")
    chatID := os.Getenv("TELEGRAM_CHAT_ID")
    if token != "" && chatID != "" {
        telegramBot = alert.NewTelegramBot(token, chatID)
        telegramBot.Send("DocSearch запущен")
        fmt.Println("Telegram бот инициализирован")
    } else {
        fmt.Println("Telegram бот не настроен (нет токена или chat_id)")
    }

    rateLimitMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            key := getClientIP(r)
            if user := r.Header.Get("Authorization"); user != "" {
                key += ":" + user
            }

            if !rateLimiter.Allow(key) {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusTooManyRequests)
                json.NewEncoder(w).Encode(map[string]interface{}{
                    "error": "Слишком много запросов. Попробуйте через минуту.",
                })
                return
            }

            next(w, r)
        }
    }
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)
	http.HandleFunc("/ask/stream", rateLimitMiddleware(RequestIDMiddleware(handleAskStream)))

	http.HandleFunc("/ask", rateLimitMiddleware(RequestIDMiddleware(handleAsk)))
    http.HandleFunc("/agent/ask", rateLimitMiddleware(RequestIDMiddleware(handleAgentAsk)))
    http.HandleFunc("/login", rateLimitMiddleware(RequestIDMiddleware(handleLogin)))

	http.HandleFunc("/", showIndex) // страницы
	http.HandleFunc("/chat.html", showChat)
	http.HandleFunc("/test.html", showTest)
	http.HandleFunc("/login.html", showLogin)
	http.HandleFunc("/register.html", showRegister)                                                
	http.HandleFunc("/register", handleRegister)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/live", handleLiveness) 
	http.HandleFunc("/ready", handleReadiness)
	http.Handle("/metrics", promhttp.Handler()) 

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
// handleLogin — вход пользователя
// @Summary Вход в систему
// @Description Авторизация пользователя, возвращает JWT токен
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body object true "Логин и пароль" example({"username":"test","password":"123456"})
// @Success 200 {object} map[string]interface{} "Токен и имя пользователя"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /login [post]
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
// handleRegister — регистрация пользователя
// @Summary Регистрация
// @Description Создание нового пользователя
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body object true "Логин и пароль" example({"username":"test","password":"123456"})
// @Success 200 {object} map[string]interface{} "Успешная регистрация"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 409 {object} map[string]string "Conflict (пользователь уже существует)"
// @Router /register [post]
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
// handleAsk — обработчик вопросов
// @Summary Задать вопрос
// @Description Отправляет вопрос в RAG систему и возвращает ответ с источниками
// @Tags Ask
// @Accept json
// @Produce json
// @Param request body object true "Запрос с вопросом" example({"query":"Что такое RAG?"})
// @Success 200 {object} map[string]interface{} "Ответ с answer, sources, timings"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 429 {object} map[string]string "Too Many Requests"
// @Security BearerAuth
// @Router /ask [post]
func handleAsk(w http.ResponseWriter, r *http.Request) { //обработчик вопрос
	fmt.Println("[handleAsk] Начало обработки запроса")

	authHeader := r.Header.Get("Authorization") // проверяю токен
	if authHeader == "" {
		fmt.Println("[handleAsk] Нет токена")
		http.Error(w, "Нет токена", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	username, err := auth.CheckToken(tokenString)
	if err != nil {
		 fmt.Println("[handleAsk] Неверный токен:", err)
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
	
	fmt.Println("[handleAsk] Отправляю ответ")
	fmt.Printf("Длина ответа: %d символов\n", len(answer))
	
	sources := []map[string]interface{}{}
	for i := 0; i < len(texts); i++ {
   
    if i >= len(docs) || i >= len(scores) || i >= len(pages) || i >= len(chunkIDs) {
        break
    }
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
	fmt.Println("[handleAsk] Ответ отправлен успешно")
}
// @Summary Задать вопрос агенту
// @Description Отправляет вопрос AI-агенту, который использует RAG
// @Tags Agent
// @Accept json
// @Produce json
// @Param request body object true "Запрос с вопросом" example({"query":"Что такое RAG?"})
// @Success 200 {object} map[string]interface{} "Ответ агента"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Security BearerAuth
// @Router /agent/ask [post]
func handleAgentAsk(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
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

	if r.Method != "POST" {
		http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка запроса", http.StatusBadRequest)
		return
	}

	ag := agent.NewAgent(globalCfg, vectorClientGlobal)
	answer, _, _, err := ag.Ask(r.Context(), req.Query, username, []map[string]string{})
	if err != nil {
		http.Error(w, "Ошибка агента: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"answer": answer,
	})
}
// handleHealth — проверка состояния
// @Summary Проверка здоровья системы
// @Description Проверяет доступность Qdrant и других компонентов
// @Tags Health
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Все компоненты работают"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 503 {object} map[string]string "Service Unavailable"
// @Router /health [get]
func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[handleAsk] Начало обработки запроса")
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		fmt.Println("[handleAsk] Нет токена")
		http.Error(w, "Нет токена", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	username, err := auth.CheckToken(tokenString)
	if err != nil {
		fmt.Println("[handleAsk] Неверный токен:", err)
		http.Error(w, "Неверный токен", http.StatusUnauthorized)
		return
	}
	fmt.Println("[handleAsk] Пользователь:", username)

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
		if i >= len(docs) || i >= len(scores) || i >= len(pages) || i >= len(chunkIDs) {
			break
		}
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

	fmt.Println("[handleAsk] Отправляю ответ")
	fmt.Printf("Длина ответа: %d символов\n", len(answer))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"answer":  answer,
		"sources": sources,
		"timings": timings,
	}); err != nil {
		fmt.Printf("Ошибка отправки ответа: %v\n", err)
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	fmt.Println("[handleAsk] Ответ отправлен успешно")
}
// handleLiveness — проверка жизни процесса
// @Summary Liveness probe
// @Description Проверяет, жив ли процесс (для Kubernetes)
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{} "Process is alive"
// @Router /live [get]
func handleLiveness(w http.ResponseWriter, r *http.Request) {  //проверка жив ли сервер
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":        "alive",
        "uptime_seconds": int64(time.Since(startTime).Seconds()),
    })
}
// handleReadiness — проверка готовности
// @Summary Readiness probe
// @Description Проверяет, готов ли сервер принимать запросы (для Kubernetes)
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string "Ready to serve"
// @Failure 503 {object} map[string]string "Not ready"
// @Router /ready [get]
func handleReadiness(w http.ResponseWriter, r *http.Request) {  //проверка готов ли сервер принимать запросы
    w.Header().Set("Content-Type", "application/json")
    
    if vectorClientGlobal == nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "not_ready",
            "reason": "vector client not initialized",
        })
        return
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    vec := make([]float32, 768)    // делаю легкий поиск с limit 1 чтобы проверить соединение
    _, err := vectorClientGlobal.Search(ctx, vector.CollectionName, vec, 1, "health_check")
    
    if err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "not_ready",
            "reason": "qdrant not responding: " + err.Error(),
        })
        return
    }
    
    json.NewEncoder(w).Encode(map[string]string{
        "status": "ready",
    })
}
// @Summary Потоковый ответ на вопрос
// @Description Отправляет вопрос в RAG и возвращает ответ через Server-Sent Events
// @Tags Ask
// @Accept json
// @Produce text/event-stream
// @Param request body object true "Запрос с вопросом" example({"query":"Что такое RAG?"})
// @Success 200 {string} string "Потоковый ответ"
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Security BearerAuth
// @Router /ask/stream [post]
func handleAskStream(w http.ResponseWriter, r *http.Request) {  // обработчик streaming
    authHeader := r.Header.Get("Authorization")
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

    if r.Method != "POST" {
        http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        Query string `json:"query"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Ошибка запроса", http.StatusBadRequest)
        return
    }

    if req.Query == "" {
        http.Error(w, "Пустой вопрос", http.StatusBadRequest)
        return
    }

    userID := username

    texts, docs, _, _, pages, _, _, _ := rag.Ask(r.Context(), *globalCfg, req.Query, userID, []map[string]string{}, vectorClientGlobal)

    if len(texts) == 0 {
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        fmt.Fprintf(w, "data: В документации нет информации по этому вопросу.\n\n")
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming не поддерживается", http.StatusInternalServerError)
        return
    }

    sourcesData, _ := json.Marshal(docs)
    fmt.Fprintf(w, "data: {\"sources\": %s}\n\n", sourcesData)
    flusher.Flush()

    stream, err := llm.GetAnswerStream(r.Context(), req.Query, texts, docs, pages, globalCfg)
    if err != nil {
        fmt.Fprintf(w, "data: Ошибка получения ответа: %s\n\n", err.Error())
        flusher.Flush()
        return
    }

    for chunk := range stream {
        if chunk.Error != nil {
            fmt.Fprintf(w, "data: Ошибка: %s\n\n", chunk.Error.Error())
            flusher.Flush()
            continue
        }
        if chunk.Done {
            fmt.Fprintf(w, "data: [DONE]\n\n")
            flusher.Flush()
            break
        }
        fmt.Fprintf(w, "data: %s\n\n", chunk.Content)
        flusher.Flush()
    }
}