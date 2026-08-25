package rerank

import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"fmt"
	"strings"
	 "sort" 
	
)

type Reranker struct {
	cfg*config.Config
}

func NewReranker(cfg*config.Config) *Reranker {  // создает новый реранкер
	return &Reranker{cfg:cfg}
	}

func (r *Reranker) Rerank(ctx context.Context, query string, chunks []string, topK int) ([]int, []float64, error) {  //  переранжирует чанки с помощью маленькой ллм
	n := len(chunks)
    if n == 0 {
        return nil, nil, nil
    }
    if topK <= 0 || topK > n {
        topK = n
    }
	if r.cfg.LLM.Provider == "mock" {
        qTokens := tokenize(query)
        type scored struct {
            idx   int
            score float64
        }
        arr := make([]scored, n)
        for i, ch := range chunks {
            arr[i] = scored{idx: i, score: overlapScore(qTokens, tokenize(ch))}
        }
        sort.Slice(arr, func(i, j int) bool { return arr[i].score > arr[j].score })

        indices := make([]int, topK)
        scores := make([]float64, topK)
        for i := 0; i < topK; i++ {
            indices[i] = arr[i].idx
            scores[i] = arr[i].score
        }
        return indices, scores, nil
    }
	
	
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
func tokenize(s string) []string {
    s = strings.ToLower(s)
    var out []string
    var b strings.Builder
    flush := func() {
        if b.Len() >= 2 {
            out = append(out, b.String())
        }
        b.Reset()
    }
    for _, r := range s {
        if (r >= 'a' && r <= 'z') || (r >= 'а' && r <= 'я') || (r >= '0' && r <= '9') || r == 'ё' {
            b.WriteRune(r)
        } else {
            flush()
        }
    }
    flush()
    return out
}

func overlapScore(query, doc []string) float64 {
    if len(query) == 0 {
        return 0
    }
    set := map[string]struct{}{}
    for _, t := range doc {
        set[t] = struct{}{}
    }
    hit := 0
    for _, t := range query {
        if _, ok := set[t]; ok {
            hit++
        }
    }
    return float64(hit) / float64(len(query))
}