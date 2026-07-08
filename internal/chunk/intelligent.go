package chunk

import (
    "regexp"
    "strings"
    "github.com/pkoukk/tiktoken-go"
)
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
    enc := tiktoken.GetEncoding("cl100k_base")  // создаю токенизатор, чтобы считать токены
    var chunks []IntelligentChunk
    var currentText strings.Builder
    currentTokens := 0
    chunkIndex := 0
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
        if strings.HasPrefix(line,"#"){
            if current.Content!=""{
                sections=append(sections,current)
            }
            current.Level=1
            current.Title=strings.TrimPrefix(line,"#")
            current.Content=""
        } else if strings.HasPrefix(line,"##"){
            if current,Content!="" {
            sections=append(sections,current)
            }
        current.Level=2
            current.Title=strings.TrimPrefix(line,"##")
            current.Content=""
        } else if strings.HasPrefix(line,"###") {
            if current,Content!="" {
            sections=append(sections,current)
        }
        current.Level=3
            current.Title=strings.TrimPrefix(line,"###")
            current.Content=""
        } else {
            if current,Content!=""{
           current.Content = line
            } else {
                current.Content = current.Content + "\n" + line
            }
        }
    }
     if current,Content!="" {
            sections=append(sections,current)
     }
         return sections
    }
    func splitSentences(text string) []string { //режу на предложения
    r:= regexp.MustCompile(`[.!?]\s+`)
    
    parts := re.Split(text, -1) //режу
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


