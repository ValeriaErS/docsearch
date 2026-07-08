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















func main() {
  
    lines := []string{
        "# Глава 1",
        "## Раздел 1.1",
        "### Пункт 1.1.1",
        "Обычный текст",
        "#  Без пробела",
    }

    for i := 0; i < len(lines); i++ {
		line:=lines[i]
        heading := isHeading(line)
		level := getHeadingLevel(line)
		title := getHeadingTitle(line)
        fmt.Printf("Строка: %q заголовок? %v, уровень: %d, название: %q\n", line, heading, level, title)
    }
}