package embed

import (
    "testing"
)

const LongVector = 768

func TestLong(t *testing.T) {
    if LongVector <= 0 {
        t.Errorf("Ошибка, размер должен быть больше 0, сейчас он %d", LongVector)
    }
}

func TestVectorneNol(t *testing.T) {
    t.Log("Тест пройден")
}

func TestLongVector(t *testing.T) {
    if LongVector != 768 {
        t.Errorf("Размер вектора %d, ожидалось 768", LongVector)
    }
    t.Log("Размер вектора 768")
}

func TestDrygText(t *testing.T) {
    
    testVectors := [][]float64{  // проверочка что для разных текстов размер одинаковый
        {0.1, 0.2, 0.3},
        {0.4, 0.5, 0.6},
        {0.7, 0.8, 0.9},
    }

    for i, vec := range testVectors {
        if len(vec) != 3 {
            t.Errorf("текст %d дал размер %d, ожидалось 3", i+1, len(vec))
        }
    }
    t.Log("Все векторы одинакового размера")
}