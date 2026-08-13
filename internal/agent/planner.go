package agent

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"encoding/json"
	"fmt"
	"strings"
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

	systemPrompt := `Ты — планировщик запросов для системы поиска документов.
Твоя задача — разбить сложный запрос пользователя на простые шаги.

Правила:
1. Если запрос простой — создай 1 шаг с тем же запросом.
2. Если запрос содержит сравнение — разбей на 3 шага:
   - Шаг 1: найти информацию об объекте А
   - Шаг 2: найти информацию об объекте Б
   - Шаг 3: сравнить А и Б
3. Если запрос содержит несколько вопросов — каждый вопрос отдельный шаг.
4. Каждый шаг должен быть самостоятельным поисковым запросом.
5. Ответь ТОЛЬКО JSON в формате:
{"steps": [{"step": 1, "description": "описание", "query": "поисковый запрос"}]}`

	messages := []map[string]string{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": fmt.Sprintf("Запрос: %s", query)},
	}

	tempCfg := *p.cfg
	tempCfg.LLM.Temperature = 0.0
	tempCfg.LLM.MaxTokens = 300

	response, _, err := llm.GetAnswerWithHistory(ctx, query, []string{}, []string{}, []int{}, messages, &tempCfg)
	if err != nil {
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