package main
import(
	"fmt"
    "strings"
    "github.com/pkoukk/tiktoken-go"
)
func main(){
	enc,_:=tiktoken.GetEncoding("cl100k_base")
	text:="Люблю кататься на машине. Бензина нет. Беда. Где брать его, подскажите мне, ибо ну надо?. Где-то. Дайте немного."

	parts:=strings.Split(text,". ") // режу текст по точкам с пробелом
	
	var chunks []string
	
	var current string // текущий чанк который собираю
	currentTokens:=0

	for i:=0;i<len(parts);i++{
		s:=parts[i]
		if i<len(parts)-1{
			s=s+"."
		}
		tokens:=enc.Encode(s,nil,nil)  // считаю токены в предложении
		tokenCount:=len(tokens)

		if currentTokens+tokenCount <= 10{
			if current!=""{
				current=current+" "+s
			}else{
				current=s
			}
			currentTokens=currentTokens+tokenCount
		} else{
			if current!=""{
			chunks=append(chunks,current)
			}
			current=s
			currentTokens=tokenCount
		}
	}
	if current!=""{
		chunks=append(chunks,current)
	}
	fmt.Println("Чанков:", len(chunks))
    for i := 0; i < len(chunks); i++ {
        fmt.Printf("%d. %s\n", i+1, chunks[i])
    }
}