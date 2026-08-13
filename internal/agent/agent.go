package agent

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/rag"
	"docsearch/internal/vector"
	"fmt"
	"strings"
)

type Agent struct {
	cfg          *config.Config
	vectorClient vector.VectorStore
	planner      *Planner
}

type TaskResult struct { // результат выполнения шага
	Query   string   `json:"query"`
	Answer  string   `json:"answer"`
	Sources []string `json:"sources"`
	Pages   []int    `json:"pages"`
	Step    int      `json:"step"`
}

func NewAgent(cfg *config.Config, vc vector.VectorStore) *Agent { // создает нового агента
	return &Agent{
		cfg:          cfg,
		vectorClient: vc,
		planner:      NewPlanner(cfg),
	}
}

func (a *Agent) Ask(ctx context.Context, question string, userID string, history []map[string]string) (string, []string, []int, error) {
	// 1. Пробуем LLM-планировщик
	plan, err := a.planner.CreatePlan(ctx, question) // создаю план
	if err == nil && len(plan.Steps) > 1 {
		return a.executePlan(ctx, question, userID, history, plan)
	}

	// 2. Если LLM не сработал — используем локальное разбиение
	subQueries := a.splitQueryLocal(question)
	if len(subQueries) > 1 {
		fmt.Printf("Локальное разбиение: %d подзапросов\n", len(subQueries))
		results := []TaskResult{}
		for i, subQuery := range subQueries {
			fmt.Printf("Шаг %d: %s\n", i+1, subQuery)
			_, docs, _, answer, pages, _, _, _ := rag.Ask(
				ctx,
				*a.cfg,
				subQuery,
				userID,
				history,
				a.vectorClient,
			)
			if isValidAnswer(answer) && !strings.Contains(answer, "нет информации") {
				results = append(results, TaskResult{
					Query:   subQuery,
					Answer:  answer,
					Sources: docs,
					Pages:   pages,
					Step:    i + 1,
				})
			}
		}
		if len(results) > 0 {
			return a.synthesizeAnswer(question, results)
		}
	}

	// 3. Fallback: обычный RAG
	return a.fallbackRAG(ctx, question, userID, history)
}

// executePlan выполняет план от LLM
func (a *Agent) executePlan(ctx context.Context, question string, userID string, history []map[string]string, plan *Plan) (string, []string, []int, error) {
	fmt.Printf("Выполняю план из %d шагов\n", len(plan.Steps))
	results := []TaskResult{}
	for _, step := range plan.Steps {
		fmt.Printf("Шаг %d: %s\n", step.Step, step.Description)
		_, docs, _, answer, pages, _, _, _ := rag.Ask(
			ctx,
			*a.cfg,
			step.Query,
			userID,
			history,
			a.vectorClient,
		)
		if isValidAnswer(answer) && !strings.Contains(answer, "нет информации") {
			results = append(results, TaskResult{
				Query:   step.Query,
				Answer:  answer,
				Sources: docs,
				Pages:   pages,
				Step:    step.Step,
			})
		}
	}
	if len(results) == 0 {
		return a.fallbackRAG(ctx, question, userID, history)
	}
	return a.synthesizeAnswer(question, results)
}

// splitQueryLocal разбивает сложный запрос на подзапросы (без LLM)
func (a *Agent) splitQueryLocal(query string) []string {
	lower := strings.ToLower(query)

	// Сравнение: "Сравни X и Y", "Сравнение X и Y"
	if strings.Contains(lower, "сравни") || strings.Contains(lower, "сравнение") || strings.Contains(lower, "разница") {
		// Пробуем найти "X и Y"
		parts := strings.Split(query, "сравни")
		if len(parts) >= 2 {
			rest := strings.TrimSpace(parts[1])
			if strings.Contains(rest, "и") {
				items := strings.Split(rest, "и")
				if len(items) >= 2 {
					obj1 := strings.TrimSpace(items[0])
					obj2 := strings.TrimSpace(items[1])
					return []string{
						fmt.Sprintf("Что такое %s?", obj1),
						fmt.Sprintf("Что такое %s?", obj2),
						fmt.Sprintf("Сравнение %s и %s", obj1, obj2),
					}
				}
			}
		}
		// Альтернативный вариант: "Сравнение RAG и FileAuditor"
		if strings.Contains(lower, "сравнение") {
			rest := strings.TrimPrefix(query, "Сравнение")
			rest = strings.TrimPrefix(rest, "сравнение")
			rest = strings.TrimSpace(rest)
			if strings.Contains(rest, "и") {
				items := strings.Split(rest, "и")
				if len(items) >= 2 {
					obj1 := strings.TrimSpace(items[0])
					obj2 := strings.TrimSpace(items[1])
					return []string{
						fmt.Sprintf("Что такое %s?", obj1),
						fmt.Sprintf("Что такое %s?", obj2),
						fmt.Sprintf("Сравнение %s и %s", obj1, obj2),
					}
				}
			}
		}
	}

	// Запрос с "и" (не сравнение)
	if strings.Contains(lower, " и ") && !strings.Contains(lower, "сравни") {
		parts := strings.Split(query, " и ")
		if len(parts) >= 2 {
			subQueries := []string{}
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if !strings.HasSuffix(part, "?") {
					part = part + " — что это?"
				}
				subQueries = append(subQueries, part)
			}
			return subQueries
		}
	}

	// Запрос с запятыми
	if strings.Contains(lower, ",") && !strings.Contains(lower, "сравни") {
		parts := strings.Split(query, ",")
		if len(parts) >= 2 {
			subQueries := []string{}
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if !strings.HasSuffix(part, "?") {
					part = part + " — что это?"
				}
				subQueries = append(subQueries, part)
			}
			return subQueries
		}
	}

	// Простой запрос — возвращаем как есть
	return []string{query}
}

func isValidAnswer(answer string) bool { // проверяет что нет галлюцинаций
	forbidden := []string{
		"User Safety: safe",
		"Response Safety: safe",
	}
	for _, f := range forbidden {
		if strings.Contains(answer, f) {
			return false
		}
	}
	return len(answer) > 10
}

func (a *Agent) fallbackRAG(ctx context.Context, query string, userID string, history []map[string]string) (string, []string, []int, error) { // обычный rag
	_, docs, _, answer, pages, _, _, _ := rag.Ask(ctx, *a.cfg, query, userID, history, a.vectorClient)
	return answer, docs, pages, nil
}

func (a *Agent) synthesizeAnswer(originalQuery string, results []TaskResult) (string, []string, []int, error) { // объединение результатов
	if len(results) == 0 {
		return "Не удалось найти информацию по вашему запросу.", []string{}, []int{}, nil
	}

	var answer strings.Builder
	allSources := []string{}
	allPages := []int{}

	answer.WriteString(fmt.Sprintf("По запросу: %s\n\n", originalQuery))

	for _, result := range results {
		answer.WriteString(fmt.Sprintf("Шаг %d: %s\n", result.Step, result.Query))
		answer.WriteString(result.Answer)
		answer.WriteString("\n\n")
		allSources = append(allSources, result.Sources...)
		allPages = append(allPages, result.Pages...)
	}

	answer.WriteString("\n---\n")
	answer.WriteString("Источники: ")
	answer.WriteString(strings.Join(allSources, ", "))

	return answer.String(), allSources, allPages, nil
}