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

func TestSimilarityDifferent(t *testing.T) { //проверяю что разные векторы дают похожесть 0.0
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}

	sim := similarity(a, b)

	if sim != 0.0 {
		t.Errorf("разные векторы должны быть 0.0, вышло %f", sim)
	}
}

func TestSimilarityDifferentLong(t *testing.T) { // проверяю что векторы разной длины дают 0.0
	a := []float64{1, 0}
	b := []float64{0, 1, 0}

	sim := similarity(a, b)

	if sim != 0.0 {
		t.Errorf("разные векторы должны быть 0.0, вышло %f", sim)
	}
}

func TestSimilarityHalf(t *testing.T) { //проверяю что частично похожие векторы дают значение между 0 и 1
	a := []float64{1, 1, 0}
	b := []float64{1, 0, 0}

	sim := similarity(a, b)

	if sim <= 0 || sim >= 1 {
		t.Errorf("частично похожие векторы должны быть между 0 и 1, вышло %f", sim)
	}
}
func TestSearch(t *testing.T) {
	texts := []string{"текст1", "текст2", "текст3"}
	docs := []string{"doc1.md", "doc2.md", "doc3.md"}
	vectors := [][]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
	query := []float64{1, 0, 0}
	topK := 2

	resultTexts, resultDocs, resultScores := Search(texts, docs, vectors, query, topK)

	if len(resultTexts) != 2 {
		t.Errorf("ожидалось 2 результата, получено %d", len(resultTexts))
	}

	if resultTexts[0] != "текст1" {
		t.Errorf("первый результат должен быть 'текст1', получено %s", resultTexts[0])
	}

	if resultDocs[0] != "doc1.md" {
		t.Errorf("первый документ должен быть 'doc1.md', получено %s", resultDocs[0])
	}

	if resultScores[0] != 1.0 {
		t.Errorf("первая оценка должна быть 1.0, получено %f", resultScores[0])
	}

	t.Log("Search работает корректно")
}

func TestSearchEmptyVectors(t *testing.T) {
	texts := []string{}
	docs := []string{}
	vectors := [][]float64{}
	query := []float64{1, 0, 0}
	topK := 5

	resultTexts, _, _ := Search(texts, docs, vectors, query, topK)

	if len(resultTexts) != 0 {
		t.Errorf("ожидалось 0 результатов, получено %d", len(resultTexts))
	}

	t.Log("Search с пустыми векторами работает")
}

func TestSimilarityZeroVectors(t *testing.T) {
	a := []float64{0, 0, 0}
	b := []float64{1, 0, 0}

	sim := similarity(a, b)

	if sim != 0.0 {
		t.Errorf("нулевой вектор должен давать 0.0, получено %f", sim)
	}

	t.Log("similarity с нулевым вектором работает")
}
