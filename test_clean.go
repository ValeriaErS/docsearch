package main
import (
    "fmt"
    "strings"
)
func cleanText(text string) string{ //убираю лишние пробелы и пустые строки
lines:=strings.Split(text,"\n")

var result []string  // сюда буду складывать чистые строки
for i:=0;i<len(lines);i++{
    line:=strings.TrimSpace(lines[i]) // убираю пробелы в начале и конце строки

    if line!=""{
        result=append(result,line)
    }
}
 return strings.Join(result, "\n")
}

func main() {
    dirty:=`
    
    Привет, Лерок!
    
    Это вторая строка.
    
    
    А это третья.
    
    `
    clean:=cleanText(dirty)
    fmt.Println("Было вот так")
    fmt.Println(dirty)
    fmt.Println("Стало вот так")
    fmt.Println(clean)
} 