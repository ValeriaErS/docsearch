package retrieve

import "math"

func Search(texts []string, vectors [][]float64, query []float64, TopK int) ([]string, []float64){
    scores := []float64{} //находим topK самых похожих+даем оценку

    for i := 0; i < len(vectors); i++ {
        scores = append(scores, similarity(query, vectors[i]))
    }

    for i := 0; i < len(scores); i++ {  //сортировка по убыв.
        for j := i + 1; j < len(scores); j++ {
            if scores[i] < scores[j] {
                scores[i], scores[j] = scores[j], scores[i]
                texts[i], texts[j] = texts[j], texts[i]
            }
        }
    }

   resultTexts := []string{}
   resultScores := []float64{}
    for i := 0; i < TopK && i < len(texts); i++ {
        resultTexts = append(resultTexts, texts[i])
        resultScores = append(resultScores, scores[i])
    }
    return resultTexts, resultScores
}

func similarity(a []float64, b []float64) float64 { //считаю косинус.близ.между векторами
    if len(a) != len(b) {
        return 0
    }

    dot := 0.0 //скалярное произведение
    for i := 0; i < len(a); i++ {
        dot = dot + a[i]*b[i]
    }

    lenA := 0.0 //длина векторов
    lenB := 0.0
    for i := 0; i < len(a); i++ {
        lenA = lenA + a[i]*a[i]
        lenB = lenB + b[i]*b[i]
    }

    lenA = math.Sqrt(lenA)
    lenB = math.Sqrt(lenB)

    if lenA == 0 || lenB == 0 {
        return 0
    }

    return dot / (lenA * lenB)
}

/*func Search(texts []string, vectors [][]float64, query []float64) []string {
    var result []string
    for i := 0; i < len(vectors); i++ {
        sim := similarity(query, vectors[i])
        if sim > 0.5 {
            result = append(result, texts[i])
        }
    }
    return result
}
*/
/*func Search(texts []string, vectors [][]float64, query []float64, TopK int) []string {
    scores := []float64{}
    for i := 0; i < len(vectors); i++ {
        scores = append(scores, similarity(query, vectors[i]))
    }

    for i := 0; i < len(scores); i++ {
        for j := i + 1; j < len(scores); j++ {
            if scores[i] < scores[j] {
                scores[i], scores[j] = scores[j], scores[i]
                texts[i], texts[j] = texts[j], texts[i]
            }
        }
    }

    result := []string{}
    for i := 0; i < TopK && i < len(texts); i++ {
        result = append(result, texts[i])
    }
    return result
}
*/