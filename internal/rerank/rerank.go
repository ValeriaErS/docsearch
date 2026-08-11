package rerank

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"fmt"
	"strings"
	
)

type Reranker struct {
	cfg*config.Config
}

func NewReranker(cfg*config.Config) *Reranker {  // создает новый реранкер
	return &Reranker{cfg:cfg}
	}

func (r *Reranker) Rerank(ctx context.Context, query string, chunks []string, topK int) ([]int, []float64, error) {  //  переранжирует чанки с помощью маленькой ллм
	if len(chunks) <= topK {
		indices:=make([]int,len(chunks))
		for i:=range indices{
			indices[i]=i
		}
		return indices,nil,nil
	}

	if r.cfg.LLM.Provider == "mock" {
		indices:=make([]int,topK)
		for i:=range indices{
			indices[i]=i
		}
		return indices,nil,nil
	}

	systemPrompt := `Ты — оценщик релевантности документов.
Оцени, насколько каждый документ отвечает на вопрос пользователя.
Выведи список индексов документов в порядке убывания релевантности (самый релевантный — первым).

Формат ответа: [0, 3, 1, 4, 2]

Вопрос: ` + query + `

Документы:
`

for i,chunk:=range chunks{
	systemPrompt += fmt.Sprintf ("\n[%d] %s\n", i, chunk[:min(len(chunk), 300)])
}
messages:=[]map[string]string{
	{"role": "system", "content": "Ты — оценщик релевантности. Отвечай только списком индексов."},
		{"role": "user", "content": systemPrompt},
}
tempCfg:=*r.cfg
tempCfg.LLM.Temperature=0.0
tempCfg.LLM.MaxTokens=100

response,_,err:=llm.GetAnswerWithHistory(ctx,query, []string{}, []string{}, []int{}, messages, &tempCfg)
if err!=nil{
	indices:=make([]int,topK)
	for i:=range indices{
			indices[i]=i
	}
	return indices,nil,nil
}
var indices []int   // парсим ответ
response=strings.TrimSpace(response)
response=strings.Trim(response, "[]")
parts:=strings.Split(response,",")

for _,part:=range parts{
	part=strings.TrimSpace(part)
	var idx int
	if _,err:=fmt.Sscanf(part, "%d", &idx); err == nil && idx < len(chunks) {
		indices=append(indices,idx)
	}
}
if len(indices)>topK{
	indices=indices[:topK]
}
if len(indices)==0{
	indices=make([]int,topK)
	for i:=range indices{
		indices[i]=i
	}
}
return indices,nil,nil
}
func min(a,b int) int{
	if a < b{
		return a
	}
	return b
}