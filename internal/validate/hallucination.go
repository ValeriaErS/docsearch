package validate

import (
	
	"strings"
)

type HallucinationReport struct {  //  содержит результаты проверки на галлюцинации
	TotalClaims   int               
	Verified      int               
	Unverified    int              
	UnverifiedList []string         
	HasHallucinations bool          
}

type HallucinationDetector struct {  //  проверяет ответ на выдуманные факты
	chunks       []string
	docNames     []string
}

func NewHallucinationDetector(chunks []string, docNames []string) *HallucinationDetector {
	return &HallucinationDetector{
		chunks:   chunks,
		docNames: docNames,
	}
}

func (d *HallucinationDetector) Detect(answer string) HallucinationReport {  //  проверяет ответ на галлюцинации
	report := HallucinationReport{
		UnverifiedList: []string{},
	}

	sentences := strings.Split(answer, ".")
	sentences = strings.Split(answer, "!")
	sentences = strings.Split(answer, "?")
	
	claims := []string{}   // собираю все утверждения
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if len(s) < 10 {
			continue
		}
		if strings.Contains(s, "[источник:") {
			continue
		}
		claims = append(claims, s)
	}

	report.TotalClaims = len(claims)

	for _, claim := range claims {
		if d.verifyClaim(claim) {
			report.Verified++
		} else {
			report.Unverified++
			report.UnverifiedList = append(report.UnverifiedList, claim)
		}
	}

	report.HasHallucinations = report.Unverified > 0
	return report
}

func (d *HallucinationDetector) verifyClaim(claim string) bool {  //  проверяет одно утверждение
	if len(d.chunks) == 0 {
		return false
	}

	claimLower := strings.ToLower(claim)
	words := strings.Fields(claimLower)
	
	keyWords := []string{}
	for _, w := range words {
		if len(w) > 3 {
			keyWords = append(keyWords, w)
		}
	}

	if len(keyWords) == 0 {
		return false
	}

	for _, chunk := range d.chunks {  // ищу совпадения в чанках
		chunkLower := strings.ToLower(chunk)
		matchCount := 0
		for _, word := range keyWords {
			if strings.Contains(chunkLower, word) {
				matchCount++
			}
		}
		if float64(matchCount) > float64(len(keyWords))*0.5 {
			return true
		}
	}
	return false
}

func CheckHallucinations(answer string, chunks []string, docNames []string) HallucinationReport {  //основной метод для проверки
	detector := NewHallucinationDetector(chunks, docNames)
	return detector.Detect(answer)
}