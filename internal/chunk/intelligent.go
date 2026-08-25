package chunk

import (
	"fmt"
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
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	if overlapTokens >= maxTokens {
		overlapTokens = maxTokens / 4
	}

	// КОРОТКИЙ ДОКУМЕНТ → один чанк со ВСЕМ текстом
	if len([]rune(text)) < 3000 {
		return []IntelligentChunk{
			{
				Text:        text,
				Document:    docName,
				Section:     "full",
				Level:       0,
				Index:       0,
				TokenCount:  len(strings.Fields(text)),
				OverlapFrom: -1,
				StartPos:    0,
			},
		}
	}

	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return []IntelligentChunk{
			{
				Text:        text,
				Document:    docName,
				Section:     "full",
				Level:       0,
				Index:       0,
				TokenCount:  len(strings.Fields(text)),
				OverlapFrom: -1,
				StartPos:    0,
			},
		}
	}

	var chunks []IntelligentChunk
	sections := parseSections(text)
	chunkIndex := 0
	globalPos := 0

	for _, section := range sections {
		// разбиваем по точкам, вопросительным и восклицательным знакам
		sentences := splitIntoSentences(section.Content)

		var current string
		var currentStartPos int
		var overlapBuffer string
		var overlapBufferStartPos int
		currentTokens := 0
		overlapTokensCount := 0

		for i := 0; i < len(sentences); i++ {
			s := sentences[i]
			sentPos := strings.Index(text[globalPos:], s)
			if sentPos == -1 {
				sentPos = strings.Index(text, s)
				if sentPos == -1 {
					continue
				}
			} else {
				sentPos += globalPos
			}

			tokenCount := len(enc.Encode(s, nil, nil))

			// Проверяем, что предложение не слишком маленькое (фильтруем мусор)
			if tokenCount < 2 && len(strings.TrimSpace(s)) < 3 {
				globalPos = sentPos + len(s)
				continue
			}

			if currentTokens+tokenCount <= maxTokens {
				if current != "" {
					current = current + " " + s
				} else {
					current = s
					currentStartPos = sentPos
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
						OverlapFrom: -1,
						StartPos:    currentStartPos,
					})
					chunkIndex++
				}

				overlapBuffer = ""
				overlapBufferStartPos = 0
				overlapTokensCount = 0

				if overlapTokens > 0 && current != "" {
					prevSentences := splitIntoSentences(current)

					for j := len(prevSentences) - 1; j >= 0; j-- {
						s2 := prevSentences[j]
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
					if overlapTokensCount+tokenCount > maxTokens {
						overlapTokensCount = maxTokens - tokenCount
						if overlapTokensCount < 0 {
							overlapTokensCount = 0
							overlapBuffer = ""
						} else {
							overlapBuffer = truncateToTokens(overlapBuffer, overlapTokensCount, enc)
						}
					}
					if overlapBuffer != "" {
						current = overlapBuffer + " " + s
					} else {
						current = s
					}
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

		if current != "" && len(current) > 10 {
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

	// Если чанков нет или они слишком маленькие - используем простое разбиение
	totalLen := 0
	for _, ch := range chunks {
		totalLen += len(ch.Text)
	}

	if len(chunks) == 0 || totalLen < len(text)/2 || len(chunks) < 5 {
		return splitBySize(text, docName, maxTokens, overlapTokens)
	}

	return chunks
}

// разбивает текст на предложения
func splitIntoSentences(text string) []string {
	var result []string
	var current strings.Builder

	for _, r := range text {
		current.WriteRune(r)
		// Проверяем конец предложения: . ! ?
		if r == '.' || r == '!' || r == '?' {
			// Проверяем, что это не часть числа (например, 1. или 2.)
			str := current.String()
			trimmed := strings.TrimSpace(str)
			// Если предложение длинное или содержит буквы - сохраняем
			if len(trimmed) > 3 && strings.ContainsAny(trimmed, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZабвгдеёжзийклмнопрстуфхцчшщъыьэюя") {
				result = append(result, trimmed)
				current.Reset()
			}
		}
	}

	// Добавляем остаток
	if current.Len() > 0 {
		trimmed := strings.TrimSpace(current.String())
		if len(trimmed) > 3 {
			result = append(result, trimmed)
		}
	}

	return result
}

func truncateToTokens(text string, maxTokens int, enc *tiktoken.Tiktoken) string {
	if text == "" || maxTokens <= 0 {
		return ""
	}

	words := strings.Fields(text)
	var result strings.Builder
	tokens := 0

	for _, word := range words {
		wordTokens := len(enc.Encode(word, nil, nil))
		if tokens+wordTokens > maxTokens {
			break
		}
		if result.Len() > 0 {
			result.WriteString(" ")
		}
		result.WriteString(word)
		tokens += wordTokens
	}

	return result.String()
}

// splitBySize - простое разбиение текста на чанки по размеру (в символах)
// Используется как fallback для больших документов без структуры
func splitBySize(text string, docName string, maxTokens int, overlapTokens int) []IntelligentChunk {
	if len(text) == 0 {
		return nil
	}

	if len(text) > 100000 {
		fmt.Printf("⚠️ splitBySize: текст слишком большой (%d байт), пропускаю обработку\n", len(text))
		return nil
	}

	// Оцениваем размер чанка в символах (примерно 4 символа на токен)
	maxChars := maxTokens * 4
	overlapChars := overlapTokens * 4

	// Если текст короткий - один чанк
	if len(text) <= maxChars {
		return []IntelligentChunk{
			{
				Text:        text,
				Document:    docName,
				Section:     "full",
				Level:       0,
				Index:       0,
				TokenCount:  len(strings.Fields(text)),
				OverlapFrom: -1,
				StartPos:    0,
			},
		}
	}

	var chunks []IntelligentChunk
	start := 0
	chunkIndex := 0

	for start < len(text) {
		end := start + maxChars
		if end > len(text) {
			end = len(text)
		}

		// Ищем конец предложения или абзаца в пределах 500 символов
		cutPos := end
		searchStart := end - 500
		if searchStart < start {
			searchStart = start
		}

		for i := end - 1; i >= searchStart; i-- {
			if text[i] == '.' || text[i] == '?' || text[i] == '!' || text[i] == '\n' {
				if i > 0 && text[i-1] >= '0' && text[i-1] <= '9' {
					continue
				}
				cutPos = i + 1
				break
			}
		}

		if cutPos == end || cutPos <= start {
			for i := end - 1; i >= searchStart; i-- {
				if text[i] == ' ' || text[i] == '\t' || text[i] == '\n' {
					cutPos = i + 1
					break
				}
			}
		}

		if cutPos <= start || cutPos > len(text) {
			cutPos = end
			if cutPos > len(text) {
				cutPos = len(text)
			}
		}

		chunkText := text[start:cutPos]
		if len(chunkText) > 20 {
			chunks = append(chunks, IntelligentChunk{
				Text:        chunkText,
				Document:    docName,
				Section:     fmt.Sprintf("part_%d", chunkIndex+1),
				Level:       0,
				Index:       chunkIndex,
				TokenCount:  len(strings.Fields(chunkText)),
				OverlapFrom: -1,
				StartPos:    start,
			})
			chunkIndex++
		}

		start = cutPos - overlapChars
		if start < 0 {
			start = 0
		}
		if start >= len(text) {
			break
		}
		if start == cutPos {
			start = cutPos + 1
		}
	}

	return chunks
}