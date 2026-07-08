package main

import (
    "fmt"
    "strings"
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

func getHeadingTitle(line string) string{
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
type Section struct{  //одн раздел
    Level int
    Title string
    Content string
}

func parseSections(text string) []Section{
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
func main(){
	text:=`# Глава 1
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
        fmt.Println("---")
        fmt.Println("Уровень:", s.Level)
        fmt.Println("Заголовок:", s.Title)
        if len(s.Content) > 30 {
            fmt.Println("Текст:", s.Content[:30]+"...")
        } else {
            fmt.Println("Текст:", s.Content)
        }
    }
}















