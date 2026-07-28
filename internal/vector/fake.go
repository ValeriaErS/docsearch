package vector
import (
    "context"
    "math"
    "fmt"
) 

type FakeVectorStore struct{
	Points []map[string]interface{} //чанки храним в памяти
}
func NewFakeVectorStore() *FakeVectorStore{ //фейк клиент
	return &FakeVectorStore{
		Points:[]map[string]interface{}{},
	}
}
func (f *FakeVectorStore) Search(ctx context.Context, name string, vec []float32, limit int, userID string) ([]map[string]interface{}, error) {
    
    if len(f.Points) == 0 {
        fmt.Println("FakeVectorStore пуст!")
        return []map[string]interface{}{}, nil
    }
    query := make([]float64, len(vec))
    for i, v := range vec {
        query[i] = float64(v)
    }

    type scoredPoint struct {  //косинусная близость
        point map[string]interface{}
        score float64
    }
    var scored []scoredPoint

    for _, point := range f.Points {
        payload, ok := point["payload"].(map[string]interface{})
        if !ok {
            continue
        }
        pointUserID, ok := payload["user_id"].(string)
        if !ok {
            continue
        }
        if pointUserID != userID {
            continue
        }

        vectorData, ok := point["vector"].([]float32) //получаю вектор чанка
        if !ok {
            continue
        }

        pointVec := make([]float64, len(vectorData))
        for i, v := range vectorData {
            pointVec[i] = float64(v)
        }

        score := cosineSimilarity(query, pointVec)  // считаю косинусную близость

        scored = append(scored, scoredPoint{
            point: point,
            score: score,
        })
    }

    for i := 0; i < len(scored); i++ {   // сортирую по убыванию оценки
        for j := i + 1; j < len(scored); j++ {
            if scored[i].score < scored[j].score {
                scored[i], scored[j] = scored[j], scored[i]
            }
        }
    }

    result := []map[string]interface{}{}   // первые limit результатов
    for i := 0; i < limit && i < len(scored); i++ {
    
        scored[i].point["score"] = scored[i].score
        result = append(result, scored[i].point)
    }
    return result, nil
}

func cosineSimilarity(a, b []float64) float64 {  //считает косинусную близость между двумя векторами
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

func (f *FakeVectorStore) Save(ctx context.Context, name string, id string, vec []float32, data map[string] interface{}) error {  //в память сохраняю
    f.Points=append(f.Points,map[string]interface{}{
	"id": id,
    "vector": vec,
    "payload": data,
    "score": 0.95,
})
return nil
}

func (f *FakeVectorStore) Delete(ctx context.Context, name string, filter map[string]interface{}) error {
    f.Points = []map[string]interface{}{}
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
        if !ok {
            continue
        }
        if pointUserID == userID {
            result = append(result, point)
        }
    }
    return result, nil
}