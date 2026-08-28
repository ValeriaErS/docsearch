package retrieve

import (
	"sort"
	"fmt"
)
type FusionResult struct {  // результат объединения
	ID      string
	Score   float64
	Payload map[string]interface{}
}
func ReciprocalRankFusion(resultsLists ...[]map[string]interface{}) []map[string]interface{} {
	const k = 60
	fusionMap := make(map[string]*FusionResult)

	for _, list := range resultsLists {
		for rank, item := range list {
			id, ok := item["id"].(string)
			if !ok {
				continue
			}
			payload, _ := item["payload"].(map[string]interface{})

			if _, exists := fusionMap[id]; !exists {
				fusionMap[id] = &FusionResult{
					ID:      id,
					Score:   0,
					Payload: payload,
				}
			}
			fusionMap[id].Score += 1.0 / float64(k+rank+1)
		}
	}

	return convertToResults(fusionMap)
}
func WeightedReciprocalRankFusion(weights []float64, resultsLists ...[]map[string]interface{}) []map[string]interface{} {
	fmt.Printf("[RRF] Объединение результатов\n")
	fmt.Printf("Векторных: %d\n", len(resultsLists[0]))
	if len(resultsLists) > 1 {
		fmt.Printf("Текстовых: %d\n", len(resultsLists[1]))
	}
	fmt.Printf("Веса: Vector=%.1f, Text=%.1f\n", weights[0], weights[1])
	
	const k = 60

	if len(weights) != len(resultsLists) {
		return ReciprocalRankFusion(resultsLists...)
	}

	fusionMap := make(map[string]*FusionResult)

	for listIdx, list := range resultsLists {
		weight := weights[listIdx]
		for rank, item := range list {
			id, ok := item["id"].(string)
			if !ok {
				continue
			}
			payload, _ := item["payload"].(map[string]interface{})

			if _, exists := fusionMap[id]; !exists {
				fusionMap[id] = &FusionResult{
					ID:      id,
					Score:   0,
					Payload: payload,
				}
			}
			fusionMap[id].Score += weight * (1.0 / float64(k+rank+1))
		}
	}

	return convertToResults(fusionMap)
}
func convertToResults(fusionMap map[string]*FusionResult) []map[string]interface{} {  // конвертирует map в слайс и сортирует
	results := make([]map[string]interface{}, 0, len(fusionMap))
	for _, item := range fusionMap {
		results = append(results, map[string]interface{}{
			"id":      item.ID,
			"score":   item.Score,
			"payload": item.Payload,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i]["score"].(float64) > results[j]["score"].(float64)
	})
	fmt.Printf("[RRF] После объединения: %d результатов\n", len(results))

	return results
}