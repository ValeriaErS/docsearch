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
}
func NewAgent(cfg *config.Config,vc vector.VectorStore) *Agent{ //создает нового агента
	return &Agent{
		cfg: cfg,
		vectorClient: vc,
	}
}
type TaskResult struct{ //результат выполнения задачи
	Query    string   `json:"query"`
	Answer   string   `json:"answer"`
	Sources  []string `json:"sources"`
	Pages    []int    `json:"pages"`
	Step     int      `json:"step"`
}
func (a *Agent) Ask(ctx context.Context,question string,userID string,history []map[string]string)(string, []string,[]int,error){
	if a.isComplexQuery(question){ //сложный ли вопросик
		subQueries:=a.splitQuery(question) //разбивка на подвопросы
		result:=[]TaskResult{} //выполняю каждый подвопрос
		for i,subQuery:=range subQueries{
			fmt.Printf("Шаг %d: %s\n", i+1, subQuery)
			texts,docs,scores,answer,pages,_,_,_:=rag.Ask(
				ctx,
				*a,cfg,
				subQuery,
				userID,
				history,
				a.vectorClient,
			)
			if answer!=""{
				result=append(results,TaskResult{
					Query:subQuery,
					Answer:answer,
					Sources:docs,
					Pages:pages,
					Step:i+1,
				})
			}
		}
		return a.synthesizeAnswer(question, results)
	}
	texts,docs,scores,answer,pages,_,_,_:=rag.Ask( //если простой запрос то обычный rag
		        ctx,
				*a,cfg,
				subQuery,
				userID,
				history,
				a.vectorClient,	
	)
	return answer,docs,pages,nil
}
func(a *Agent) isComplexQuery(query string)bool{
	complexKeywords:=[]string{
		"сравни", "сравнение", "разница", "отличие",
		"проанализируй", "анализ",
		"объясни по шагам",
		"и", "а также",
	}
	lower:=strings.ToLower(query)
	for _,keyword:=range complexKeywords{
		if strings.Contains(lower,keyword){
			return true
		}
	}
	if strings.Count(lower,"?")>1{
		return true
	}
	return false
}
func (a *Agent) splitQuery(query string) []string{
	lower:=strings.ToLower(query)
	if strings.Contains(lower, "сравни") || strings.Contains(lower, "сравнение"){ //если есть сравни то разбивка на две части

		parts:=strings.Split(query, "сравни")
		if len(parts)>=2{
			rest:=strings.TrimSpace(parts[1])
			if strings.Contains(resr, "и"){
				items:=strings.Split(rest,"и")
				if len(items)>=2{
					return []string{
						strings.TrimSpace(items[0])+" — что это?",
						strings.TrimSpace(items[1])+" — что это?",
						"Сравни"+strings.TrimSpace(items[0])+" и " + strings.TrimSpace(items[1]),

					}
				}
			}
		}
	}
	if strings.Contains(lower,"и")&& !strings.Contains(lower,"сравни"){
		parts:=strings.Split(query,"и")
		if len(parts)>=2{
			subQueries:=[]string{}
			for _,part:=range parts{
				subQueries=append(subQueries,strings.TrimSpace(part))
			}
			return subQueries
		}
	}
	return []string{query}
}
func (a *Agent) synthesizeAnswer(originalQuery string, results []TaskResult) (string, []string, []int, error) {
	if len(results)==0{
		return "Не удалось найти информацию по вашему запросу.", []string{},[]int{},nil
	}
	var answer strings.Builder
	allSources:=[]string{}
	allPages:=[]int{}
	answer.WriteString(fmt.Sprintf("По вашему запросу: %s\n\n", originalQuery))
	for _,result:=range result{
		answer.WriteString(fmt.Sprintf("Шаг %d: %s\n", result.Step, result.Query))
		answer.WriteString(fmt.Sprintf("%s\n\n", result.Answer))
		allSources = append(allSources, result.Sources...)
		allPages = append(allPages, result.Pages...)
	}
	answer.WriteString("\n---\n")
	answer.WriteString("Источники: ")
	answer.WriteString(strings.Join(allSources, ", "))
	
	return answer.String(), allSources, allPages, nil
}