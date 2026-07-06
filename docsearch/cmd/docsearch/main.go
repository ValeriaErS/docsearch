package main
import (
    "encoding/json"
    "fmt"
    "os"
    "docsearch/internal/config"
    "docsearch/internal/rag"
    "time"
)
func main() {
    args:= os.Args[1:]                                          // читаю кроме 1 слова команды
    configFile:= "configs/config.yml"
    needIndex:= false                                       // флаг для индексации, вопрос,сохранение
    question:= ""
    outFile:= ""
    serveMode:= false
    port:= ":8080"

    for i:= 0; i < len(args); i++ {                            //смторю что ввел пользователь
        if args[i] == "--config" && i+1 < len(args) {
            configFile = args[i+1]
            i = i + 1
        } else if args[i] == "index" {
            needIndex = true
        } else if args[i] == "ask" && i+1 < len(args) {                      // поиск
            question = args[i+1]
            i = i + 1
        } else if args[i] == "--out" && i+1 < len(args) {                //сохранение в файл
            outFile = args[i+1]
            i = i + 1
        } else if args[i] == "--serve" {                                  // запускаю сервак
          serveMode = true
    } else if args[i] == "--port" && i+1 < len(args) {
      port = args[i+1]
      i = i + 1
    }
        
    }

    cfg, err:= config.LoadConfig(configFile)             //загрузка настроек
    if err!= nil {
    fmt.Println("Ошибка загрузки конфига:", err)
    return
    }
    if serveMode {                                            // режим сервера есть
    runServe(configFile, port)
    return
}
    if needIndex {                                           //есть индексация
    rag.Index(*cfg)
    return
    }
    if question != "" {
    startTime:= time.Now()                                     //время и ищу ответ
    results, docs, scores, answer:= rag.Ask(*cfg, question)

    found:= false
    for i:= 0; i < len(scores); i++ {                   //проверка порога
            if scores[i] >= cfg.Retrieval.MinScore {
                found = true
                break
            }
        }

        if !found {
            fmt.Println("Ответа в документации нет.")
            return
        }

        type Source struct {
            DocID string `json:"doc_id"`
            Score float64 `json:"score"`
            Snippet string `json:"snippet"`
        }

        type Response struct {
            Query string `json:"query"`
            Answer string `json:"answer"`
            Sources []Source `json:"sources"`
            Model string `json:"model"`
            TokensUsed int `json:"tokens_used"`
            DurationMs int64 `json:"duration_ms"`
        }

        var sources []Source                       //сборка источников
        for i:= 0; i < len(results); i++ {
            snippet:= results[i]
            if len(snippet) > 100 {
                snippet = snippet[:100] + "..."
            }
            sources = append(sources, Source{
                DocID:docs[i],
                Score:scores[i],
                Snippet:snippet,
            })
        }
        duration:= time.Since(startTime).Milliseconds()                //время выполнения
        resp:= Response{
            Query:question,
            Answer:answer,
            Sources:sources,
            Model:cfg.LLM.Model,       
            TokensUsed:512,                  
            DurationMs:duration,
        }

        jsonData, err:= json.MarshalIndent(resp, "", "  ")      //в json
        if err!= nil {
        fmt.Println("Ошибка формирования JSON:", err)
        return
        }

        if outFile!= "" {
            err:= os.WriteFile(outFile, jsonData, 0644)
            if err!= nil {
                fmt.Println("Ошибка сохранения в файл:", err)
            } else {
                fmt.Println("Результат сохранён в", outFile)
            }
        } else {
            fmt.Println(string(jsonData))
        }
        return
    }

    fmt.Println("Команды:")
    fmt.Println("index - индексация документов")                    // подсказка
    fmt.Println("ask 'текст' - поиск по документации")
    fmt.Println("ask 'текст' --out file.json - сохранить результат в JSON-файл")
    fmt.Println("--serve - запустить HTTP сервер")
    fmt.Println("--port :8080 - порт для сервера")
}

   
/*
package main

import "fmt"

func main() {
	fmt.Println("Запуск программы")
}
*/

/*
package main

import (
	"fmt"
	"docsearch/internal/config"
	"docsearch/internal/corpus"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yml")
	if err != nil {
		fmt.Println("Ошибка", err)
		return
	}

	docs, err := corpus.LoadDocuments(cfg.Corpus.Path)
	if err != nil {
		fmt.Println("Ошибка", err)
		return
	}

	fmt.Println("Документов:", len(docs))
}
*/

/*
package main

import (
	"fmt"
	"docsearch/internal/config"
	"docsearch/internal/corpus"
	"docsearch/internal/chunk"
)

func main() {
	cfg, _ := config.LoadConfig("configs/config.yml")

	docs, _ := corpus.LoadDocuments(cfg.Corpus.Path)

	for i := 0; i < len(docs); i++ {
		doc := docs[i]
		fmt.Println("Файл:", doc.Name)

		chunks := chunk.SplitText(doc.Text, 500, 50)
		fmt.Println("Чанков:", len(chunks))

		for j := 0; j < len(chunks); j++ {
			fmt.Println("Чанк", j+1, chunks[j].Text[:50])
		}
	}
}
*/

/*
package main

import (
	"fmt"
	"docsearch/internal/config"
	"docsearch/internal/corpus"
	"docsearch/internal/chunk"
	"docsearch/internal/embed"
)

func main() {
	cfg, _ := config.LoadConfig("configs/config.yml")

	docs, _ := corpus.LoadDocuments(cfg.Corpus.Path)

	slovar := []string{"embedding", "вектор", "поиск", "документ", "текст"}

	for i := 0; i < len(docs); i++ {
		doc := docs[i]
		chunks := chunk.SplitText(doc.Text, 500, 50)

		for j := 0; j < len(chunks); j++ {
			ch := chunks[j]
			v := embed.GetVector(ch.Text, slovar)
			fmt.Println("Вектор:", v[:5])
		}
	}
}
*/
