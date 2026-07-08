package chunk

import (
    "regexp"
    "strings"
    "fmt"  
)
func isHeading(line string)bool{   //проверяет начинается ли строка с # ## или ###
    if string.HasPrefix(line,"# "){
        return true
    }
    if string.HasPrefix(line,"## "){
        return true
    }
    if string.HasPrefix(line,"### "){
        return true
    }
    return false
}






type IntelligentChunk struct {  // я храню один кусок текста
    Text string
    Document string
    Section string
    Level int
    Index int
    TokenCount int
}
type Section struct{  //одн раздел
    Level int
    Title string
    Content string
}

func SplitIntelligent(text string, docName string, maxTokens int, overlapTokens int) []IntelligentChunk { // эта функция режет текст на куски
    var chunks []IntelligentChunk
    return chunks
}

func parseSections(text string) []Section {  // ищу заголовки,собираю текст
    var sections []Section
    lines := strings.Split(text, "\n")
    
    var current Section
    current.Level = 1
    current.Title = "root"

    for i:=0;i<len(lines);i++{
        line:=strings.TrimSpace(lines[i])
        if strings.HasPrefix(line,"# "){
            if current.Content!=""{
                sections=append(sections,current)
            }
            current.Level=1
            current.Title=strings.TrimPrefix(line,"#")
            current.Content=""
        } else if strings.HasPrefix(line,"## "){
            if current.Content!="" {
            sections=append(sections,current)
            }
        current.Level=2
            current.Title=strings.TrimPrefix(line,"##")
            current.Content=""
        } else if strings.HasPrefix(line,"### ") {
            if current.Content!="" {
            sections=append(sections,current)
        }
        current.Level=3
            current.Title=strings.TrimPrefix(line,"###")
            current.Content=""
        } else {
            if current.Content==""{
            current.Content = line
            } else {
                current.Content = current.Content + "\n" + line
            }
        }
    }
     if current.Content!="" {
            sections=append(sections,current)
     }
         return sections
    }
    func splitSentences(text string) []string { //режу на предложения
    r:= regexp.MustCompile(`[.!?]\s+`)
    
    parts := r.Split(text, -1) //режу
    out:=[]string{}

    for i := 0; i < len(parts); i++ {
        s:= strings.TrimSpace(parts[i])
        if s == "" {
            continue
            }
            if s[len(s)-1]!='.' && s[len(s)-1] != '!' && s[len(s)-1] != '?' { //ставлю точку
            s = s + "."
        }
        out = append(out, s)
    }

    return out
}


func TestIntelligent() {
    fmt.Println("Тест")

    text := `# Глава 1
    Тут текст первой главы.
    ## Раздел 1.1
    Тут текст раздела.
    ### Пункт 1.1.1
    Тут текст пункта.
    # Глава 2
    Тут текст второй главы.`

    sections := parseSections(text)

    fmt.Println("Нашла разделов:", len(sections))
    for i := 0; i < len(sections); i++ {
        s := sections[i]
        fmt.Println("Уровень:", s.Level, "Заголовок:", s.Title)
        fmt.Println("Текст:", s.Content[:30]+"...")
    }

    fmt.Println("Работает")
}


