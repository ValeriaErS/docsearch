package embed
import (
    "testing"
)
const LongVector = 768

func TestLong(t *testing.T) {    // проверяю что константа не нулевая
    if LongVector <= 0 {
        t.Errorf("Ошибка, размер должен быть больше 0, сейчас он %d", LongVector)
    }
}

func TestVectorneNol(t *testing.T) {
    text:= "Привет"

    vec, err:= GetEmbedding(text)
    if err!= nil {
        t.Skip("Модель не запущена, пропускаем тест")
        return
    }
    if len(vec) == 0 {                 // проверка что вектор не пуст
        t.Error("вектор пуст")
    }
}

func TestLongVector(t *testing.T) {      // проверяю что размер вектора совпадает с ожидаемым
    text:= "проверка размера"

    vec, err:= GetEmbedding(text)
    if err!= nil {
        t.Skip("модель не запущена, пропускаем тест")
        return
    }

    if len(vec)!= LongVector {
        t.Errorf("размер вектора %d, должен быть %d", len(vec), LongVector)
    }
}

func TestDrygText(t *testing.T) {  // проверяю что для разных текстов размер вектора одинаковый
    texts:= []string{
        "короткий",
        "длинный текст для проверки",
        "еще текст",
    }

    for i:= 0; i < len(texts); i++ {
        vec, err:= GetEmbedding(texts[i])
        if err!= nil {
            t.Skip("модель не запущена, пропускаем тест")
            return
        }
        if len(vec)!= LongVector {          // проверяю, что размер всегда одинаковый
            t.Errorf("текст %d дал размер %d, ожидалось %d", i+1, len(vec), LongVector)
        }
    }
}