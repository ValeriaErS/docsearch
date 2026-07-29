package chunk

import (
	"github.com/pkoukk/tiktoken-go"
	"strings"
)

func isHeading(line string) bool { //проверяет начинается ли строка с # ## или ###
	if strings.HasPrefix(line, "# ") {
		return true
	}
	if strings.HasPrefix(line, "## ") {
		return true
	}
	if strings.HasPrefix(line, "### ") {
		return true
	}
	return false
}

func getHeadingLevel(line string) int { //возвращает уровень заголовка (1, 2 или 3)
	if strings.HasPrefix(line, "### ") {
		return 3
	}
	if strings.HasPrefix(line, "## ") {
		return 2
	}
	if strings.HasPrefix(line, "# ") {
		return 1
	}
	return 0
}

func getHeadingTitle(line string) string { // убирает # оставляет только название
	if strings.HasPrefix(line, "### ") {
		return strings.TrimPrefix(line, "### ")
	}
	if strings.HasPrefix(line, "## ") {
		return strings.TrimPrefix(line, "## ")
	}
	if strings.HasPrefix(line, "# ") {
		return strings.TrimPrefix(line, "# ")
	}
	return line
}

type Section struct { //один раздел
	Level   int
	Title   string
	Content string
}

func parseSections(text string) []Section { // разбирает текст на разделы по заголовкам
	var sections []Section
	lines := strings.Split(text, "\n")

	var current Section
	current.Level = 1
	current.Title = "root"

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if isHeading(line) { // если в текущем разделе есть текст сохраняю его
			if current.Content != "" {
				sections = append(sections, current)
			}
			current.Level = getHeadingLevel(line) // начинаю новый раздел
			current.Title = getHeadingTitle(line)
			current.Content = ""
		} else {
			if current.Content == "" { // обычный текст добавляю к текущему разделу
				current.Content = line
			} else {
				current.Content = current.Content + "\n" + line
			}
		}
	}
	if current.Content != "" {
		sections = append(sections, current)
	}
	return sections

}

type IntelligentChunk struct { // один чанк
	Text        string
	Document    string
	Section     string
	Level       int
	Index       int
	TokenCount  int
	Page        int
	OverlapFrom int
	StartPos    int
}

func SplitIntelligent(text string, docName string, maxTokens int, overlapTokens int) []IntelligentChunk {
	if overlapTokens >= maxTokens { //если перекрытие слишком большое уменьшаю его
		overlapTokens = maxTokens / 4
	}

	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return []IntelligentChunk{} // нет энкодер, то пустой результат
	}

	var chunks []IntelligentChunk
	sections := parseSections(text)
	chunkIndex := 0
	globalPos := 0 //позиция в общем тексте

	for _, section := range sections {
		sentences := strings.Split(section.Content, ". ")

		var current string
		var currentStartPos int
		var overlapBuffer string
		var overlapBufferStartPos int
		currentTokens := 0
		overlapTokensCount := 0

		for i := 0; i < len(sentences); i++ {
			s := sentences[i]
			if i < len(sentences)-1 {
				s = s + "."
			}
			 sentPos := strings.Index(text[globalPos:], s)
			 if sentPos==-1{
				sentPos=strings.Index(text,s)
				if sentPos==-1{
					continue
				}
			 } else{
				sentPos+=globalPos
			 }

			tokenCount := len(enc.Encode(s, nil, nil))

			if currentTokens+tokenCount <= maxTokens { // влезет ли предложение в текущий чанк
				if current != "" {
					current = s
					currentStartPos = sentPos
				}else {
                    current = current + " " + s
				}
				currentTokens = currentTokens + tokenCount
			} else {

				if current != "" {
					chunks = append(chunks, IntelligentChunk{
						Text:        current,
						Document:    docName,
						Section:     section.Title,
						Level:       section.Level,
						Index:       chunkIndex,
						TokenCount:  currentTokens,
						OverlapFrom: -1, //нет перекрытия
						StartPos:    currentStartPos,
					})
					chunkIndex++
				}

					overlapBuffer = ""
                    overlapBufferStartPos = 0
					overlapTokensCount = 0

					if overlapTokens > 0 && current != "" {
					prevSentences := strings.Split(current, ". ")

					for j := len(prevSentences) - 1; j >= 0; j-- { //с конца собираю
						s2 := prevSentences[j]
						if j < len(prevSentences)-1 {
							s2 = s2 + "."
						}
						tCount := len(enc.Encode(s2, nil, nil))
						if overlapTokensCount+tCount <= overlapTokens {
							if overlapBuffer != "" {
								overlapBuffer = s2 + ". " + overlapBuffer
							} else {
								overlapBuffer = s2
								overlapBufferStartPos = len(current) - len(s2) - 1
                                if overlapBufferStartPos < 0 {
                                overlapBufferStartPos = 0
							}
						}
							overlapTokensCount = overlapTokensCount + tCount
						} else {
							break
						}
					}
				}
if overlapBuffer != "" {
			
				
					current = overlapBuffer + " " + s
					currentStartPos = overlapBufferStartPos
					currentTokens = overlapTokensCount + tokenCount
				} else {
					current = s
					currentStartPos = sentPos
					currentTokens = tokenCount
				}
				
			}
			globalPos = sentPos + len(s)
		}

		if current != "" {
			chunks = append(chunks, IntelligentChunk{
				Text:        current,
				Document:    docName,
				Section:     section.Title,
				Level:       section.Level,
				Index:       chunkIndex,
				TokenCount:  currentTokens,
				OverlapFrom: -1,
				StartPos:    currentStartPos,
			})
			chunkIndex++
		}
	}
	return chunks
}
