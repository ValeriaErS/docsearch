package llm

import (
    "testing"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "docsearch/internal/config"
)
func TestMockAnswer(t *testing.T) {
    mockAnswer := "Это тестовый ответ"
    if mockAnswer == "" {
        t.Error("Ответ пустой")
    }
    t.Log("Тестовый ответ:", mockAnswer)
}
func TestGetAnswerWithHistoryMock(t *testing.T){ //тест без реального API
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        if r.URL.Path!="/chat/completions" {
        t.Errorf("Неверный путь: %s", r.URL.Path)
        }
        if r.Method!= "POST" {
        
        t.Errorf("Неверный метод: %s", r.Method)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "choices": []map[string]interface{}{
        {
            "message": map[string]string{
            "content": "Это тестовый ответ от LLM",
        },
    },
},
"usage":map[string]int{
"total_tokens":42,
        },
    })
}))
    defer server.Close()

    cfg:=&config.Config{    //создаю конфиг
    LLM: struct{
    Provider string `yaml:"provider"`
    Model string `yaml:"model"`
    BaseURL string `yaml:"base_url"`
    Temperature float64 `yaml:"temperature"`
    MaxTokens int `yaml:"max_tokens"`
    }{
    Provider:"test",
    Model:"test-model",
    BaseURL:server.URL,
    Temperature: 0.7,
    MaxTokens: 100,
    },
}
   os.Setenv("LLM_API_KEY", "test-key")
   defer os.Unsetenv("LLM_API_KEY")  

   answer,tokens,err:=GetAnswerWithHistory(  //вызываю функцию
   context.Background(),
   "Что такое RAG?",
   []string{"Контекст: RAG - это подход..."},
   []string{"doc1.md"},
   []int{1},
   []map[string]string{},
   cfg,
   )

   if err!=nil{
   t.Fatalf("Ошибка получения ответа: %v", err)
   }
   if answer==""{
   t.Error("Ответ пустой")
   }
   if tokens==0{
   t.Error("Токены не вернулись")
   }

   t.Logf("Ответ получен: %s", answer)
   t.Logf("Токены: %d", tokens)
   }

   func TestGetAnswerWithHistoryNoKey(t *testing.T){  // проверяет то функция возвращает ошибку без API ключа
   os.Unsetenv ("LLM_API_KEY")
   
   cfg:=&config.Config{
   LLM: struct {
            Provider string `yaml:"provider"`
            Model string `yaml:"model"`
            BaseURL string `yaml:"base_url"`
            Temperature float64 `yaml:"temperature"`
            MaxTokens int `yaml:"max_tokens"`
        }{
          Provider: "test",
            Model: "test-model",
            BaseURL: "http://localhost:8080",
        },
    }
_, _, err:=GetAnswerWithHistory(
    context.Background(),
    "test",
    []string{},
    []string{},
    []int{},
    []map[string]string{},
    cfg,
)
if err==nil{
    t.Error("Ожидалась ошибка при отсутствии ключа, но её нет")
} else{
    t.Log("Ошибка получена:", err)
}
}
