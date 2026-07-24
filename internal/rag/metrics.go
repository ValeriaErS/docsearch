package rag
import (
	"regexp"
	"strings"
)
type CitationMetrics struct{  //хранит информацию о цитатах в ответе
	TotalCitations int  // сколько всего ссылок
	UniqueSources int   // сколько источников
	Sources map[string]bool   // список источников
}
func CountCitations(answer string)CitationMetrics{   // ищет ссылки
	metrics:=CitationMetrics{
		Sources:make(map[string]bool),
	}
	re:=regexp.MustCompile(`\[источник:\s*([^\]]+)\]`)
	matches:=re.FindAllStringSubmatch(answer,-1)  // все совпадения
    
	metrics.TotalCitations=len(matches)
	for _, match:=range matches{   // проход по каждой найденной ссылке
		if len (match)>1{
			source:=strings.TrimSpace(match[1])
			metrics.Sources[source]=true //запоминаю
		}
	}
	metrics.UniqueSources=len(metrics.Sources)
	return metrics
}