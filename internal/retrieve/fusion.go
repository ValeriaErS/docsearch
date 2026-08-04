package retrieve

import (
	"sort"
)

func ReciprocalRankFusion(resultsLists ...[]map[string]interface{}) []map[string]interface{} {  //  объединяет результаты нескольких поисков,использует формулу: score = sum(1 / (k + rank))
	const k = 60 // Константа RRF

	fusionMap := make(map[string]*FusionResult)

	for _, list := range resultsLists {
		for rank, item := range list {
			id, ok := item["id"].(string)
			if !ok {
				continue
			}

			payload, _ := item["payload"].(map[string]interface{})
			score, _ := item["score"].(float64)

			if _, exists := fusionMap[id]; !exists {
				fusionMap[id] = &FusionResult{
					ID:      id,
					Score:   0,
					Payload: payload,
				}
			}

			fusionMap[id].Score += 1.0/float64(k+rank+1) + score*0.01
		}
	}

	results := make([]map[string]interface{}, 0, len(fusionMap))
	for _, item := range fusionMap {
		results = append(results, map[string]interface{}{
			"id":      item.ID,
			"score":   item.Score,
			"payload": item.Payload,
		})
	}

	sort.Slice(results, func(i, j int) bool {   //  по убыванию оценки
		return results[i]["score"].(float64) > results[j]["score"].(float64)
	})

	return results
}

type FusionResult struct {
	ID      string
	Score   float64
	Payload map[string]interface{}
}