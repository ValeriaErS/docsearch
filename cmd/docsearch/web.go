package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"docsearch/internal/auth"
	"docsearch/internal/config"
	"docsearch/internal/db"
	"docsearch/internal/embed"
	"docsearch/internal/llm"
	"docsearch/internal/vector"
)

var chatHistory = make(map[string][]map[string]string)
var database *db.DB

func runWeb(cfg *config.Config, port string) {
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

	err = database.AddUser(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Пользователь уже существует", http.StatusConflict)
		return
	}

	userDir := "docs/" + req.Username // создаю папку
	os.MkdirAll(userDir, 0755)
	fmt.Println("Папка создана:", userDir)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    req.Username,
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
		User  string `json:"user"`
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

	userID := req.User
	if userID == "" {
		userID = "default"
	}

	if chatHistory[userID] == nil {
		chatHistory[userID] = []map[string]string{}
	}

	chatHistory[userID] = append(chatHistory[userID], map[string]string{
		"role":    "user",
		"content": req.Query,
	})

	answer, sources, timings := getAnswer(req.Query, userID)

	chatHistory[userID] = append(chatHistory[userID], map[string]string{
		"role":    "assistant",
		"content": answer,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"answer":  answer,
		"sources": sources,
		"timings": timings,
	})
}

func getAnswer(question string, userID string) (string, []map[string]interface{}, map[string]float64) {
	startTotal := time.Now()

	startEmbed := time.Now()
	client := vector.NewQdrantClient()
	client.VectorSize = 768

	vec, err := embed.GetEmbedding(question)
	if err != nil {
		return "Ошибка: не могу понять вопрос", nil, map[string]float64{}
	}
	embedDuration := time.Since(startEmbed).Seconds()

	vec32 := []float32{} // подготовка вектора
	for _, v := range vec {
		vec32 = append(vec32, float32(v))
	}

	startSearch := time.Now() // поиск в Qdrant
	results, err := client.Search("documents", vec32, 10, userID)
	if err != nil || len(results) == 0 {
		return "Ничего не нашла", nil, map[string]float64{}
	}
	searchDuration := time.Since(startSearch).Seconds()

	context := []string{}
	sources := []map[string]interface{}{}
	docNames := []string{}
	pages := []int{} 

	for _, r := range results {
		payload := r["payload"].(map[string]interface{})
		text := payload["chunk_text"].(string)
		docName := payload["doc_id"].(string)

		
		page := 1  // достаю страницу из qdrant
		if p, ok := payload["page"].(float64); ok && int(p) > 0 {
			page = int(p)
		}

		context = append(context, text)
		sources = append(sources, map[string]interface{}{
			"doc_id": payload["doc_id"],
			"score": r["score"],
			"page": page,
		})
		docNames = append(docNames, docName)
		pages = append(pages, page)
	}

	seen := map[string]bool{} // убираю дубликаты
	unique := []map[string]interface{}{}
	for _, s := range sources {
		name := s["doc_id"].(string)
		if !seen[name] {
			seen[name] = true
			unique = append(unique, s)
		}
	}

	startLLM := time.Now()
	answer, err := llm.GetAnswerWithHistory(question, context, docNames, pages, chatHistory[userID]) // передаю страницы
	if err != nil {
		fmt.Println("Ошибка LLM:", err)
		return "Ошибка: нейросеть не отвечает", unique, map[string]float64{}
	}
	llmDuration := time.Since(startLLM).Seconds()

	totalDuration := time.Since(startTotal).Seconds()

	timings := map[string]float64{
		"total": totalDuration,
		"embed": embedDuration,
		"search": searchDuration,
		"llm": llmDuration,
	}

	return answer, unique, timings
}