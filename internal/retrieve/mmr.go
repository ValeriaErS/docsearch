package retrieve

import (
	"math"
	"fmt"
)

// По формуле из статьи DataTalks.ru:
// MMR(dᵢ) = λ·Sim(dᵢ, query) − (1−λ)·maxⱼ∈S Sim(dᵢ, dⱼ)
//
// Параметры:
//   - results: результаты поиска (с полем "score" как релевантность)
//   - lambda: баланс между релевантностью (1.0) и разнообразием (0.0)
//   - topK: сколько документов вернуть
//   - fetchK: сколько кандидатов брать до MMR
func MMRSelect(results []map[string]interface{}, lambda float64, topK int, fetchK int) []map[string]interface{} {
	fmt.Printf("[MMR] Выбор разнообразных чанков\n")
	fmt.Printf("Кандидатов: %d, λ=%.2f\n", len(results), lambda)

	if len(results) == 0 {
		return results
	}

	if len(results) <= topK {
		return results
	}

	if lambda <= 0 {
		lambda = 0.5
	}
	if lambda > 1 {
		lambda = 1.0
	}
	if topK <= 0 {
		topK = 5
	}

	if fetchK <= 0 || fetchK > len(results) {
		fetchK = len(results)
	}

	candidates := results[:fetchK]

	scores := make([]float64, len(candidates))
	for i, r := range candidates {
		s, ok := r["score"].(float64)
		if !ok {
			s = 0.0
		}
		scores[i] = s
	}

	similarityMatrix := make([][]float64, len(candidates)) // вычисляю попарное сходство между кандидатами
	for i := 0; i < len(candidates); i++ {
		similarityMatrix[i] = make([]float64, len(candidates))
		for j := 0; j < len(candidates); j++ {
			diff := math.Abs(scores[i] - scores[j])
			similarityMatrix[i][j] = 1.0 / (1.0 + diff*10) // нормализую в [0,1]
		}
	}

	selected := []map[string]interface{}{}
	selectedIndices := []int{}

	for len(selected) < topK && len(selectedIndices) < len(candidates) {
		bestIdx := -1
		bestScore := -math.MaxFloat64

		for i := 0; i < len(candidates); i++ {
			alreadySelected := false
			for _, idx := range selectedIndices {
				if idx == i {
					alreadySelected = true
					break
				}
			}
			if alreadySelected {
				continue
			}

			relevance := lambda * scores[i] // релевантность

			diversity := 0.0 // разнообразие
			if len(selectedIndices) > 0 {
				maxSim := -math.MaxFloat64
				for _, idx := range selectedIndices {
					if similarityMatrix[i][idx] > maxSim {
						maxSim = similarityMatrix[i][idx]
					}
				}
				diversity = (1 - lambda) * maxSim
			}

			mmrScore := relevance - diversity

			if mmrScore > bestScore {
				bestScore = mmrScore
				bestIdx = i
			}
		}

		if bestIdx == -1 {
			break
		}

		selected = append(selected, candidates[bestIdx])
		selectedIndices = append(selectedIndices, bestIdx)
	}

	fmt.Printf("[MMR] Оставлено: %d чанков\n", len(selected))
	return selected
}