package main
import(
	"fmt"
    "docsearch/internal/chunk"
    "docsearch/internal/corpus"
)
func main(){
	docs,err:=corpus.LoadDocuments("./docs")
	if err!=nil{
		fmt.Println("Ошибка загрузки:", err)
        return 
	}
	fmt.Println("Найдено документов:",len(docs))
	for _, doc:=range docs{
		fmt.Println("\n---", doc.Name, "---")
		fmt.Println("Размер текста:", len(doc.Text), "символов")

		chunks:=chunk.SplitIntelligent(doc.Text,doc.Name,512)
			fmt.Println("Чанков:",len(chunks))

			for i:=0;i<3 && i<len(chunks);i++{
				ch:=chunks[i]
				preview:=ch.Text
				if len(preview)>100{
					preview=preview[:100]+"..."
				}
				fmt.Printf("Чанк %d (уровень %d, %d токенов): %s\n", i+1, ch.Level, ch.TokenCount, preview)
			}
	}
	}
