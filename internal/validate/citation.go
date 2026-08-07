package validate
import (
	"regexp"
	"strings"
)
type Citation struct{
	FullText string
	Source string
	Page string
	IsValid bool // подтверждена ли ссылка
	ErrorMessage string // причина если не подтверждена
}
type CitationValidator struct{

}
func NewCitationValidator() *CitationValidator{
	return &CitationValidator{}
}
func (v *CitationValidator) ExtractCitations(text string) []Citation{  //  извлекает все ссылки из текста
	re:=regexp.MustCompile(`\[источник:\s*([^,\]]+)(?:,\s*страница\s*(\d+))?\]`)
	matches:=re.FindAllStringSubmatch(text, -1)

	citations:= []Citation{}
	for _, match:=range matches{
		fullText := match[0]
		source := strings.TrimSpace(match[1])
		page:="1"
		if len(match)>2 && match[2]!=""{
			page=match[2]
		}
		citations=append(citations,Citation{
			FullText: fullText,
			Source:  source,
			Page:     page,
			IsValid:  false, 
		})
	}
	return citations
}
func (v *CitationValidator) ValidateCitations(answer string, chunks []string, docNames []string) (string, []Citation){ //  проверяет ссылки по документам
	citations:=v.ExtractCitations(answer)
	if len(citations)==0{
		return answer, citations
	}
	for i :=range citations{
		citations[i].IsValid=v.verifyCitation(citations[i], chunks, docNames)
	}
	validatedAnswer:=answer
	for _, cit:=range citations{
		if !cit.IsValid{
			validatedAnswer=strings.ReplaceAll(validatedAnswer, cit.FullText, "")
			cit.ErrorMessage="Информация не найдена в указанном источнике"
		}
	}
	return validatedAnswer,citations

}
func (v *CitationValidator) verifyCitation(cit Citation, chunks []string, docNames []string) bool{
	found:=false
	for i,doc:=range docNames{
		if strings.Contains(doc,cit.Source) || strings.Contains(cit.Source,doc){

			if i<len(chunks) && len(chunks[i])>20{
				found=true
				break
			}
		}
	}
	return found
}
func ValidateAnswer(answer string, chunks []string, docNames []string) (string, []Citation) {  //основной метод для проверки всего ответа
	validator := NewCitationValidator()
	return validator.ValidateCitations(answer, chunks, docNames)
}