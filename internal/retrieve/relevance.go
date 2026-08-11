package retrieve
import (
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"encoding/json"
	"strings"
)
func RelevanceCheck(ctx context.Context,question string,chunks []string,cfg *config.Config)(bool,float64,error){
	if len(chunks)==0{
		return false,0,nil
	}
	if cfg.LLM.Provider=="mock"{
		return true, 0.8, nil
	}
	context:=strings.Join(chunks,"\n---\n")
	if len(context)>2000{
		context=context[:2000]+"..."
	}
	systemPrompt := `Ты — оценщик релевантности контекста.
Определи, есть ли в предоставленном контексте ответ на вопрос пользователя.

Ответь ТОЛЬКО JSON:
{"has_answer": true/false, "confidence": 0.0-1.0}

Вопрос: ` + question + `

Контекст:
` + context

	messages := []map[string]string{
		{"role": "system", "content": "Ты — оценщик релевантности. Отвечай только JSON."},
		{"role": "user", "content": systemPrompt},
	}
	tempCfg:=*cfg
	tempCfg.LLM.Temperature=0.0
	tempCfg.LLM.MaxTokens=50

	response,_,err:=llm.GetAnswerWithHistory(ctx,question,[]string{}, []string{}, []int{}, messages, &tempCfg)
	if err!=nil{
		return true,0.5,nil
	}
	var result struct{
		HasAnswer bool    `json:"has_answer"`
		Confidence float64 `json:"confidence"`
	}
	response=strings.TrimSpace(response)
	start:=strings.Index(response,"{")
	end:=strings.LastIndex(response,"}")
	if start!=-1 && end!=-1 && end>start{
		jsonStr:=response[start:end+1]
		if err:=json.Unmarshal([]byte(jsonStr), &result); err == nil {
			return result.HasAnswer, result.Confidence, nil
	}
}
return true, 0.5, nil
}