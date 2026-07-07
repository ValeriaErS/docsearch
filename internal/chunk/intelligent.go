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
func SplitIntelligent(text string, docName string, maxTokens int, overlapTokens int) []IntelligentChunk { // эта функция режет текст на куски
    enc := tiktoken.GetEncoding("cl100k_base")  // создаю токенизатор, чтобы считать токены
    var chunks []IntelligentChunk
    var currentText strings.Builder
    currentTokens := 0
    chunkIndex := 0
    return chunks
}

type Section struct {  // один раздел 
    Level int
    Title string
    Content string
}


func parseSections(text string) []Section {  // разбираю текст на главы и разделы
    var sections []Section
    lines := strings.Split(text, "\n")
    var current Section
    current.Level = 1
    current.Title = "root"

    return sections
}

