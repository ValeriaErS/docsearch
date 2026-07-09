package chunk
import(
    "strings"
    "github.com/pkoukk/tiktoken-go"
)
func isHeading(line string)bool{   //проверяет начинается ли строка с # ## или ###
    if strings.HasPrefix(line,"# "){
        return true
    }
    if strings.HasPrefix(line,"## "){
        return true
    }
    if strings.HasPrefix(line,"### "){
        return true
    }
    return false
}

func getHeadingLevel(line string) int{ //возвращает уровень заголовка (1, 2 или 3)
	if strings.HasPrefix(line,"### "){
		return 3
	}
    if strings.HasPrefix(line,"## "){
		return 2
	}
    if strings.HasPrefix(line,"# "){
		return 1
	}
	return 0
}

func getHeadingTitle(line string) string{   // убирает # оставляет только название
	if strings.HasPrefix(line, "### "){
		return strings.TrimPrefix(line, "### ")
	}
    if strings.HasPrefix(line, "## "){
		return strings.TrimPrefix(line, "## ")
	}
	if strings.HasPrefix(line, "# "){
		return strings.TrimPrefix(line, "# ")
	}
	return line
}

type Section struct{  //один раздел
    Level int
    Title string
    Content string
}

func parseSections(text string) []Section{  // разбирает текст на разделы по заголовкам
	var sections []Section
	lines:=strings.Split(text, "\n")

	var current Section
	current.Level=1
	current.Title="root"

	for i:=0;i<len(lines);i++{
		line:=strings.TrimSpace(lines[i])
    if isHeading(line){      // если в текущем разделе есть текст сохраняю его
		if current.Content!=""{
			sections=append(sections,current)
		}
		current.Level=getHeadingLevel(line)  // начинаю новый раздел
		current.Title=getHeadingTitle(line)
		current.Content=""
	}else{
		if current.Content==""{  // обычный текст добавляю к текущему разделу
			current.Content=line
		}else{
			current.Content=current.Content+"\n"+line
		}
	}
}
if current.Content!=""{
	sections=append(sections,current)
}
return sections

}

type IntelligentChunk struct {  // один чанк
	Text string
	Document string
	Section string
	Level int
	Index int
	TokenCount int 
}
func SplitIntelligent(text string, docName string, maxTokens int)[]IntelligentChunk{ //режу
	enc,_:=tiktoken.GetEncoding ("cl100k_base")
	var chunks []IntelligentChunk

	sections:=parseSections(text)
	chunkIndex:=0

	for _, section:=range sections{
		sentences:=strings.Split(section.Content, ". ")

		var current string
		currentTokens:=0

		for i:=0;i<len(sentences);i++{
			s:=sentences[i]
			if i<len(sentences)-1{
				s=s+"."
			}
		tokenCount:=len(enc.Encode(s,nil,nil))
		if currentTokens+tokenCount<=maxTokens{
			if current!=""{
				current=current+" "+s
			} else {
				current=s
			}
			currentTokens=currentTokens+tokenCount
		} else {
			if current!=""{
				chunks=append(chunks,IntelligentChunk{
			    Text:current,
				Document:docName,
				Section:section.Title,
				Level: section.Level,
				Index:chunkIndex,
				TokenCount:currentTokens,
				})
	            chunkIndex++
		}
		current=s
		currentTokens=tokenCount
		}
	}
	if current!=""{     // сохраняю последний чанк в разделе
		chunks=append(chunks, IntelligentChunk{
			Text:current,
			Document:docName,
			Section:section.Title,
			Level:section.Level,
			Index:chunkIndex,
			TokenCount:currentTokens,
		})
		chunkIndex++
	}
}
return chunks
}