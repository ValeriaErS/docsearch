package main

import (
	"fmt"
    "encoding/json"
    "net/http"
    "docsearch/internal/embed"
    "docsearch/internal/llm"
    "docsearch/internal/vector"
    "docsearch/internal/config"
    "net"
)
var chatHistory=make(map[string][]map[string]string)

func runWeb(cfg *config.Config, port string, userID string) {     //запуск сервера
    if userID==""{
        userID="default"
    }
    fmt.Println("Пользователь:",userID)

if chatHistory[userID]==nil{
    chatHistory[userID]=[]map[string]string{}
}

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "web/index.html")
    })

    http.HandleFunc("/ask", func(w http.ResponseWriter, r *http.Request) {  // обрабатываю вопросы которые приходят из чата
        if r.Method != "POST" {
            http.Error(w, "Нужен POST", http.StatusMethodNotAllowed)
            return
        }

        var req struct {               // читаю вопрос из тела
            Query string `json:"query"`
        }
        json.NewDecoder(r.Body).Decode(&req)

        if req.Query == "" {
            http.Error(w, "Пустой вопрос", http.StatusBadRequest)
            return
        }
        chatHistory[userID]=append(chatHistory[userID],map [string]string{
            "role": "user",
            "content": req.Query,
        })

       answer, sources := findAnswer(req.Query, userID)

       chatHistory[userID]=append(chatHistory[userID],map[string]string{
        "role": "assistant",
        "content": answer,
       })

        w.Header().Set("Content-Type", "application/json")     // отправляю ответ
        json.NewEncoder(w).Encode(map[string]interface{}{
            "answer":  answer,
            "sources": sources,
        })
    })
    fullAddress:="0.0.0.0" + port

    fmt.Println("Сайт запущен: http://localhost" + port)
     fmt.Println("В сети: http://" + getLocalIP() + port)
    http.ListenAndServe(fullAddress, nil)
}
func getLocalIP() string{
    addrs,err:=net.InterfaceAddrs()
    if err!=nil{
        return "localhost"
    }
    for _, addr:=range addrs{
        if ipnet,ok:=addr.(*net.IPNet);ok && !ipnet.IP.IsLoopback(){
           if ipnet.IP.To4() != nil {
                return ipnet.IP.String() 
           }
        }
    }
    return "localhost"
}



func findAnswer(question string, userID string) (string, []map[string]interface{}) {   //ищет ответ на вопрос в документации
    fmt.Println("Поиск для пользователя",userID)
    client := vector.NewQdrantClient()
    client.VectorSize = 999
   

    vec, err := embed.GetEmbedding(question)
    if err != nil {
        return "Ошибка: не могу понять вопрос", nil
    }

    vec32 := []float32{}
    for _, v := range vec {
        vec32 = append(vec32, float32(v))
    }

    results, err := client.Search("documents", vec32, 10, userID)   // ищу похожие чанки
    if err != nil || len(results) == 0 {
        return "Ничего не нашла", nil
    }

    context := []string{}
    sources := []map[string]interface{}{}

    for _, r := range results {
        payload := r["payload"].(map[string]interface{})
        text := payload["chunk_text"].(string)
        context = append(context, text)
        sources = append(sources, map[string]interface{}{
            "doc_id": payload["doc_id"],
            "score":  r["score"],
        })
    }

seen := map[string]bool{}
uniqueSources := []map[string]interface{}{}
for _, s := range sources {
    docID := s["doc_id"].(string)
    if !seen[docID] {
        seen[docID] = true
        uniqueSources = append(uniqueSources, s)
    }
}
sources = uniqueSources


    answer, err := llm.GetAnswerWithHistory(question, context, chatHistory[userID])    // отправляю в llm
    if err != nil {
         fmt.Println("Ошибка LLM:", err) 
        return "Ошибка: нейросеть не отвечает", sources
    }

    return answer, sources
}