package query

import (
	"strings"
)

type Complexity string

const (
	ComplexitySimple  Complexity = "simple"
	ComplexityMedium  Complexity = "medium"
	ComplexityComplex Complexity = "complex"
)
func ClassifyComplexity(query string) Complexity {   //  определяет сложность запроса
	lower := strings.ToLower(strings.TrimSpace(query))
	words := strings.Fields(lower)
	n := len(words)

	if n == 0 {
		return ComplexityMedium
	}

	complexMarkers := []string{
		"сравни", "сравнение", "разница", "отличие", "отличия",
		"в отличие", "по сравнению", "плюсы и минусы",
		"versus", " vs ", " vs", "vs ",
		"чем отличается",
	}
	for _, m := range complexMarkers {
		if strings.Contains(lower, m) {
			return ComplexityComplex
		}
	}

	if strings.Count(lower, "?") >= 2 {
		return ComplexityComplex
	}

	if n >= 15 {
		return ComplexityComplex
	}

	mediumMarkers := []string{
		"как ", "почему", "зачем", "объясни", "расскажи",
		"настрой", "установ", "инструкц", "пошагов",
		"опиши", "покажи",
	}
	for _, m := range mediumMarkers {
		if strings.Contains(lower, m) {
			return ComplexityMedium
		}
	}
	if n <= 5 && (strings.HasPrefix(lower, "что такое") ||
		strings.HasPrefix(lower, "кто ") ||
		strings.HasPrefix(lower, "что есть")) {
		return ComplexitySimple
	}

	if n <= 4 {
		return ComplexitySimple
	}


	return ComplexityMedium
}
type RetrievalStrategy struct { //стратегия поиска
    CandidateTopK  int
    RerankTopK     int
    FinalTopK      int
    UseRewriting   bool
    UseHybrid      bool
    UseRerank      bool
    UseMultiQuery  bool
    UseMMR         bool
    Description    string
}

func GetRetrievalStrategy(complexity Complexity) RetrievalStrategy {
    switch complexity {
    case ComplexitySimple:
        return RetrievalStrategy{
            CandidateTopK:  20,
            RerankTopK:     5,
            FinalTopK:      3,
            UseRewriting:   false,
            UseHybrid:      false,
            UseRerank:      false,
            UseMultiQuery:  false,
            UseMMR:         true,
            Description:    "простой поиск",
        }
    case ComplexityMedium:
        return RetrievalStrategy{
            CandidateTopK:  50,
            RerankTopK:     10,
            FinalTopK:      5,
            UseRewriting:   true,
            UseHybrid:      true,
            UseRerank:      true,
            UseMultiQuery:  false,
            UseMMR:         true,
            Description:    "гибридный поиск с реранкингом",
        }
    case ComplexityComplex:
        return RetrievalStrategy{
            CandidateTopK:  100,
            RerankTopK:     15,
            FinalTopK:      8,
            UseRewriting:   true,
            UseHybrid:      true,
            UseRerank:      true,
            UseMultiQuery:  true,
            UseMMR:         true,
            Description:    "полный пайплайн",
        }
    default:
        return RetrievalStrategy{
            CandidateTopK:  50,
            RerankTopK:     10,
            FinalTopK:      5,
            UseRewriting:   true,
            UseHybrid:      true,
            UseRerank:      true,
            UseMultiQuery:  false,
            UseMMR:         true,
            Description:    "стандартный поиск",
        }
    }
}