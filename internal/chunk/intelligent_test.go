package chunk

import (
    "testing"
	"strings"
)

func TestSplitIntelligentWithOverlap(t *testing.T) {
    text := "# Заголовок\n" +
        "Первое предложение. Второе предложение. Третье предложение. " +
        "Четвертое предложение. Пятое предложение. Шестое предложение."
    
    docName := "test.md"
    maxTokens := 20 // маленький размер для проверки перекрытия
    overlapTokens := 10

    chunks := SplitIntelligent(text, docName, maxTokens, overlapTokens)

    if len(chunks) == 0 {
        t.Error("Не создано ни одного чанка")
    }

    for i := 1; i < len(chunks); i++ {  //есть ли перекрытие

        prevWords := strings.Split(chunks[i-1].Text, " ")
        currWords := strings.Split(chunks[i].Text, " ")
        
        overlapFound := false
        for _, pw := range prevWords {
            for _, cw := range currWords {
                if len(pw) > 3 && pw == cw { // проверяю слова длиннее 3 символов
                    overlapFound = true
                    break
                }
    }
            if overlapFound {
                break
        }
        }
        
        if !overlapFound {
            t.Logf("Внимание: перекрытие между чанками %d и %d не найдено", i-1, i)
        }
    }

    t.Logf("Создано %d чанков с перекрытием", len(chunks))
}

func TestSplitIntelligentNoOverlap(t *testing.T) {
    text := "# Тест\nПредложение 1. Предложение 2. Предложение 3."
    docName := "test.md"
    maxTokens := 100
    overlapTokens := 0

    chunks := SplitIntelligent(text, docName, maxTokens, overlapTokens)

    if len(chunks) != 1 {
        t.Errorf("Ожидался 1 чанк, получено %d", len(chunks))
    }
    
    t.Log("Чанк без перекрытия создан корректно")
}