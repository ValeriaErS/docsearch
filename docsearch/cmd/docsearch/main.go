package main

import (
    "os"
    "fmt"
    "docsearch/internal/config"
    "docsearch/internal/rag"
)

func main() {
 args:=os.Args[1:] //читаю из терминала все кроме первого слова
 configFile:="configs/config.yml"
 needIndex:=false
 question:=""

 for i:=0;i<len(args);i++{
	if args[i]=="--config" && i+1<len(args){ //что ввел чел
		configFile=args[i+1]
		i=i+1
	} else if args[i]=="--index"{
		needIndex=true
	} else if args[i]=="--query" && i+1<len(args){
		question=args[i+1]
		i=i+1
	}
}
	cfg,err:=config.LoadConfig(configFile) //загрузка настроек
	if err!=nil{
		fmt.Println("Ошибка",err)
		return
	}
	if needIndex{
		rag.Index(*cfg)
		return
	}
	if question!="" {                     //вопрос-ответ
		fmt.Println("Вопрос:",question)
		results,scores:=rag.Ask(*cfg,question)
	
	found:=false
	for i:=0;i<len(scores);i++{
		if scores[i]>=cfg.Retrieval.MinScore{  //проверка результата через порог
			found=true
			break
		}
	}
	if !found{
		fmt.Println("В документации нет ответа")
		return
 }
 fmt.Println("Найдено:", len(results))  //вывод чанков
        for i := 0; i < len(results); i++ {
            fmt.Printf("  %d. %s (оценка: %.2f)\n", i+1, results[i], scores[i])
        }
       
    }

    fmt.Println("Что делать:")
    fmt.Println("  --index - индексация")
    fmt.Println("  --query 'текст'- поиск")
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