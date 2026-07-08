package main
import (
	"fmt"
    "github.com/pkoukk/tiktoken-go"
)
func countTokens(text string) int {  //количество токенов
	enc,err:=tiktoken.GetEncoding("cl100k_base") //токенизатор openai
	if err!=nil{
		return 0
	}
	tokens:=enc.Encode(text,nil,nil) /текст в список токенов
	return len(tokens)
}
func main(){
	text := "Привет, как дела, как работа, бензин появился? Это тестовый текст для подсчёта токенов."
    count := countTokens(text)
    fmt.Println("Текст:", text)
    fmt.Println("Количество токенов:", count)
}