package retrieve
import "testing"

func TestSimilaritySame(t *testing.T) { // проверяю что одинаковые векторы дают похожесть 1.0
    a := []float64{1, 0, 0}
    b := []float64{1, 0, 0}

    sim := similarity(a, b)

    if sim != 1.0 {
        t.Errorf("одинаковые векторы должны быть 1.0, вышло %f", sim)
    }
}

func TestSimilarityDifferent(t *testing.T) {  //проверяю что разные векторы дают похожесть 0.0
    a := []float64{1, 0, 0}
    b := []float64{0, 1, 0}

    sim := similarity(a, b)

    if sim != 0.0 {
        t.Errorf("разные векторы должны быть 0.0, вышло %f", sim)
    }
}

func TestSimilarityDifferentLong(t *testing.T) {   // проверяю что векторы разной длины дают 0.0
    a := []float64{1, 0}
    b := []float64{0, 1, 0}

    sim := similarity(a, b)

    if sim != 0.0 {
        t.Errorf("разные векторы должны быть 0.0, вышло %f", sim)
    }
}

func TestSimilarityHalf(t *testing.T) {   //проверяю что частично похожие векторы дают значение между 0 и 1
    a := []float64{1, 1, 0}
    b := []float64{1, 0, 0}

    sim := similarity(a, b)

    if sim <= 0 || sim >= 1 {
        t.Errorf("частично похожие векторы должны быть между 0 и 1, вышло %f", sim)
    }
}