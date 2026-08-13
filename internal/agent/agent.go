package agent
import(
	"fmt"
	"docsearch/internal/config"
	"docsearch/internal/rag"
	"docsearch/internal/vector"
	"context"
	"strings"
)
type Agent struct{
	cfg *config.Config
	vectorClient vector.VectorStore
	planner *Planner
}
type TaskResult struct {
	Query   string   `json:"query"`
	Answer  string   `json:"answer"`
	Sources []string `json:"sources"`
	Pages   []int    `json:"pages"`
	Step    int      `json:"step"`
}

func NewAgent(cfg *config.Config,vc vector.VectorStore) *Agent{ //создает нового агента
	return &Agent{
		cfg: cfg,
		vectorClient: vc,
		planner: NewPlanner(cfg),
	}
}

func (a *Agent) Ask(ctx context.Context, question string, userID string, history []map[string]string) (string, []string, []int, error) {
	plan, err := a.planner.CreatePlan(ctx, question)  //создаю план
	if err != nil {
		_, docs, _, answer, pages, _, _, _ := rag.Ask(ctx, *a.cfg, question, userID, history, a.vectorClient)  //обычный rag
		return answer, docs, pages, nil
	}

	if len(plan.Steps) == 1 {
		_, docs, _, answer, pages, _, _, _ := rag.Ask(ctx, *a.cfg, plan.Steps[0].Query, userID, history, a.vectorClient)
		return answer, docs, pages, nil
	}

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

		if answer != "" && answer != "В документации нет информации по этому вопросу" {
			results = append(results, TaskResult{
				Query:   step.Query,
				Answer:  answer,
				Sources: docs,
				Pages:   pages,
				Step:    step.Step,
			})
		}
	}

	return a.synthesizeAnswer(question, results)
}

func (a *Agent) synthesizeAnswer(originalQuery string, results []TaskResult) (string, []string, []int, error) {  //объединение результатов
	if len(results) == 0 {
		return "Не удалось найти информацию по вашему запросу.", []string{}, []int{}, nil
	}

	var answer strings.Builder
	allSources := []string{}
	allPages := []int{}

	for _, result := range results {
		answer.WriteString(fmt.Sprintf("Шаг %d: %s\n\n", result.Step, result.Query))
		answer.WriteString(result.Answer)
		answer.WriteString("\n\n")
		allSources = append(allSources, result.Sources...)
		allPages = append(allPages, result.Pages...)
	}

	return answer.String(), allSources, allPages, nil
}