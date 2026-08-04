package vector

import (
	"context"
	"math"
	"sort"   
	"strings"
)

type FakeVectorStore struct {
	Points []map[string]interface{} //чанки храним в памяти
}

func NewFakeVectorStore() *FakeVectorStore { //фейк клиент
	return &FakeVectorStore{
		Points: []map[string]interface{}{},
	}
}

func (f *FakeVectorStore) Save(ctx context.Context, name string, id string, vec []float32, data map[string]interface{}) error { //в память сохраняю
	f.Points = append(f.Points, map[string]interface{}{
		"id":      id,
		"vector":  vec,
		"payload": data,
		"score":   0.95,
	})
	return nil
}

func (f *FakeVectorStore) Search(ctx context.Context, name string, vec []float32, limit int, userID string) ([]map[string]interface{}, error) {
	if len(f.Points) == 0 {
		return []map[string]interface{}{}, nil
	}

	query := make([]float64, len(vec))
	for i, v := range vec {
		query[i] = float64(v)
	}

	type scoredPoint struct { //косинусная близость
		point map[string]interface{}
		score float64
	}
	scored := []scoredPoint{}

	for _, point := range f.Points {
		payload, ok := point["payload"].(map[string]interface{})
		if !ok {
			continue
		}
		pointUserID, ok := payload["user_id"].(string)
		if !ok || pointUserID != userID {
			continue
		}

		vecData, ok := point["vector"].([]float32)
		if !ok {
			continue
		}

		pointVec := make([]float64, len(vecData))
		for i, v := range vecData {
			pointVec[i] = float64(v)
		}

		score := cosineSimilarity(query, pointVec) // считаю косинусную близость
		scored = append(scored, scoredPoint{point: point, score: score})
	}

	for i := 0; i < len(scored); i++ { // сортирую по убыванию оценки
		for j := i + 1; j < len(scored); j++ {
			if scored[i].score < scored[j].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	result := []map[string]interface{}{} // первые limit результатов
	for i := 0; i < limit && i < len(scored); i++ {
		pointCopy := make(map[string]interface{})
		for k, v := range scored[i].point {
			pointCopy[k] = v
		}
		pointCopy["score"] = scored[i].score
		result = append(result, pointCopy)
	}

	return result, nil
}

func (f *FakeVectorStore) Delete(ctx context.Context, name string, filter map[string]interface{}) error {
	must, ok := filter["must"].([]map[string]interface{}) // беру doc_id и user_id из фильтра
	if !ok || len(must) < 2 {
		f.Points = []map[string]interface{}{}
		return nil
	}

	var docID, userID string
	for _, cond := range must {
		key, ok := cond["key"].(string)
		if !ok {
			continue
		}
		match, ok := cond["match"].(map[string]interface{})
		if !ok {
			continue
		}
		value, ok := match["value"].(string)
		if !ok {
			continue
		}
		if key == "doc_id" {
			docID = value
		}
		if key == "user_id" {
			userID = value
		}
	}

	if docID == "" || userID == "" {
		f.Points = []map[string]interface{}{}
		return nil
	}

	newPoints := []map[string]interface{}{} // оставляю только те точки которые не совпадают с doc_id и user_id
	for _, point := range f.Points {
		payload, ok := point["payload"].(map[string]interface{})
		if !ok {
			newPoints = append(newPoints, point)
			continue
		}
		pointDocID, _ := payload["doc_id"].(string)
		pointUserID, _ := payload["user_id"].(string)
		if pointDocID == docID && pointUserID == userID {
			continue
		}
		newPoints = append(newPoints, point)
	}
	f.Points = newPoints
	return nil
}

func (f *FakeVectorStore) CreateCollection(ctx context.Context, name string) error {
	return nil
}

func (f *FakeVectorStore) Ping(ctx context.Context) error {
	return nil
}

func (f *FakeVectorStore) GetAllVectors(ctx context.Context, name string, userID string) ([]map[string]interface{}, error) {
	result := []map[string]interface{}{}
	for _, point := range f.Points {
		payload, ok := point["payload"].(map[string]interface{})
		if !ok {
			continue
		}
		pointUserID, ok := payload["user_id"].(string)
		if !ok || pointUserID != userID {
			continue
		}
		result = append(result, point)
	}
	return result, nil
}

func cosineSimilarity(a, b []float64) float64 { //считает косинусную близость между двумя векторами
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	dot := 0.0
	normA := 0.0
	normB := 0.0

	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
    }
	
func (f *FakeVectorStore) SearchText(ctx context.Context, name string, query string, limit int, userID string) ([]map[string]interface{}, error) { //имитация полнотекстового поиска
	queryWords := strings.Fields(strings.ToLower(query))

	var results []map[string]interface{}
	for _, point := range f.Points {
		payload, ok := point["payload"].(map[string]interface{})
		if !ok {
			continue
		}

		pointUserID, ok := payload["user_id"].(string)
		if !ok || pointUserID != userID {
			continue
		}

		chunkText, ok := payload["chunk_text"].(string)
		if !ok {
			continue
		}

		score := 0.0 // счет совпадения ключевых слов
		textLower := strings.ToLower(chunkText)
		for _, word := range queryWords {
			if len(word) > 2 && strings.Contains(textLower, word) {
				score += 1.0
			}
		}

		if score > 0 {
			score = score / float64(len(queryWords))
			results = append(results, map[string]interface{}{
				"id":      point["id"],
				"score":   score,
				"payload": payload,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {  // сортирует по оценке
		return results[i]["score"].(float64) > results[j]["score"].(float64)
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}
