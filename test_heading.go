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

func main() {
  
    lines := []string{
        "# Глава 1",
        "## Раздел 1.1",
        "### Пункт 1.1.1",
        "Обычный текст",
        "#  Без пробела",
    }

    for i := 0; i < len(lines); i++ {
        result := isHeading(lines[i])
        fmt.Printf("Строка: %q заголовок? %v\n", lines[i], result)
    }
}