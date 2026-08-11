package verify
import(
	"context"
	"docsearch/internal/config"
	"docsearch/internal/llm"
	"encoding/json"
	"fmt"
	"strings"
)
type VerificationResult struct{   //  результат проверки ответа
	IsAccurate bool   `json:"is_accurate"`
	Reason     string `json:"reason"`
	Confidence float64 `json:"confidence"`
	FixedAnswer string `json:"fixed_answer,omitempty"`
}
func VerifyAnswer(ctx context.Context, question string, answer string, chunks []string, cfg *config.Config) (*VerificationResult, error){  //  проверяет ответ после генерации
	if len(chunks)==0 || answer==""{
		return &VerificationResult{
			IsAccurate: false,
			Reason:     "Нет контекста или ответа",
			Confidence: 0,
		}, nil
		
	}
	if cfg.LLM.Provider=="mock"{
		return &VerificationResult{
			IsAccurate: true,
			Reason:     "Mock режим",
			Confidence: 0.9,
		}, nil
	}
	context:=strings.Join(chunks,"\n---\n")
	if len(context)>2000{
		context=context[:2000]+"..."
	}
	systemPrompt := `Ты — проверяющий качество ответов.

Проверь, соответствует ли ответ на вопрос предоставленному контексту.

Критерии:
1. Ответ должен быть основан ТОЛЬКО на контексте
2. В ответе не должно быть выдуманных фактов
3. Все утверждения должны подтверждаться контекстом

Если ответ правильный — верни {"is_accurate": true}
Если ответ содержит ошибки или выдумки — верни {"is_accurate": false, "reason": "причина"}

Вопрос: ` + question + `

Контекст:
` + context + `

Ответ для проверки:
` + answer

	messages := []map[string]string{
		{"role": "system", "content": "Ты — проверяющий. Отвечай только JSON."},
		{"role": "user", "content": systemPrompt},
	}
	tempCfg:= *cfg
	tempCfg.LLM.Temperature=0.0
	tempCfg.LLM.MaxTokens=100

	response,_,err:=llm.GetAnswerWithHistory(ctx, question, []string{}, []string{}, []int{}, messages, &tempCfg)
	if err!=nil{
		return &VerificationResult{
			IsAccurate: true,
			Reason:     "Проверка недоступна",
			Confidence: 0.5,
		}, nil
	}
	var result VerificationResult   // парсим JSON
	response=strings.TrimSpace(response)
	start:=strings.Index(response,"{")
	end:=strings.LastIndex(response,"}")
	if start!=-1 && end!=-1 && end>start{
		jsonStr:=response[start:end+1]
		if err:=json.Unmarshal([]byte(jsonStr), &result);err==nil{
			return &result,nil
		}
	}
	return &VerificationResult{
		IsAccurate: true,
		Reason:     "Не удалось проверить",
		Confidence: 0.5,
	}, nil
}