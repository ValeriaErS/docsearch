package agent

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Plan struct {
	Steps []PlanStep `json:"steps"`
}

type PlanStep struct {  //  один шаг плана
	Step        int    `json:"step"`
	Description string `json:"description"`
	Query       string `json:"query"`
}

type Planner struct {  //  планирует выполнение сложных запросов
	cfg *config.Config
}

func NewPlanner(cfg *config.Config) *Planner {
	return &Planner{cfg: cfg}
}

func (p *Planner) CreatePlan(ctx context.Context, query string) (*Plan, error) {
	if p.cfg.LLM.Provider == "mock" {
		return p.createSimplePlan(query), nil
	}

	systemPrompt := `Ты — планировщик запросов для поиска документов.
Разбей сложный запрос на простые шаги.

Правила:
1. Если запрос простой — 1 шаг.
2. Если запрос содержит "сравни", "сравнение", "разница" — 3 шага:
   - Шаг 1: описание первого объекта
   - Шаг 2: описание второго объекта
   - Шаг 3: сравнение
3. Если запрос содержит "и" — каждый объект отдельный шаг.
4. Каждый шаг — самостоятельный поисковый запрос.

Ответь ТОЛЬКО JSON:
{"steps": [{"step": 1, "description": "описание", "query": "запрос"}]}

Запрос: ` + query

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": query},
	}

	tempCfg := *p.cfg
	tempCfg.LLM.Temperature = 0.0
	tempCfg.LLM.MaxTokens = 300

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	response, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, &tempCfg)
	if err != nil {
		fmt.Printf("Ошибка планировщика: %v, использую локальное разбиение\n", err)
		return p.createSimplePlan(query), nil
	}

	response = strings.TrimSpace(response)  // парсим JSON
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || end <= start {
		return p.createSimplePlan(query), nil
	}

	jsonStr := response[start : end+1]
	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return p.createSimplePlan(query), nil
	}

	if len(plan.Steps) == 0 {
		return p.createSimplePlan(query), nil
	}

	fmt.Printf("Создан план из %d шагов\n", len(plan.Steps))
	for _, step := range plan.Steps {
		fmt.Printf("Шаг %d: %s\n", step.Step, step.Description)
	}

	return &plan, nil
}

func (p *Planner) createSimplePlan(query string) *Plan {  //план из одного шага
	return &Plan{
		Steps: []PlanStep{
			{
				Step: 1,
				Description: "Поиск информации по запросу",
				Query: query,
			},
		},
	}
}