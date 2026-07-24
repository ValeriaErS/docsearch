package retrieve
import "math"

func Search(texts []string, docs []string, vectors [][]float64, query []float64, TopK int) ([]string, []string, []float64) { // возвращает тексты, имена документов и оценки
    scores := []float64{}
    resultTexts := []string{}
    resultDocs := []string{}
    resultScores := []float64{}


    for i := 0; i < len(vectors); i++ {  // считаю похожесть для каждого вектора
        sim := similarity(query, vectors[i])
        scores = append(scores, sim)
        resultTexts = append(resultTexts, texts[i])
        resultDocs = append(resultDocs, docs[i])
        resultScores = append(resultScores, sim)
    }

   
    for i := 0; i < len(scores); i++ {      // сортирую по убыванию самые похожие будут первыми, меняю местами
        for j := i + 1; j < len(scores); j++ {
            if scores[i] < scores[j] {
                scores[i], scores[j] = scores[j], scores[i]
                resultTexts[i], resultTexts[j] = resultTexts[j], resultTexts[i]
                resultDocs[i], resultDocs[j] = resultDocs[j], resultDocs[i]
                resultScores[i], resultScores[j] = resultScores[j], resultScores[i]
            }
        }
    }

    finalTexts := []string{}  // беру первые TopK
    finalDocs := []string{}
    finalScores := []float64{}
    for i := 0; i < TopK && i < len(resultTexts); i++ {
        finalTexts = append(finalTexts, resultTexts[i])
        finalDocs = append(finalDocs, resultDocs[i])
        finalScores = append(finalScores, resultScores[i])
    }

    return finalTexts, finalDocs, finalScores
}

func similarity(a []float64, b []float64) float64 {  // считает косинусную близость между двумя векторами
    if len(a) != len(b) {
        return 0
    }

    dot := 0.0   // скалярное произведение
    for i := 0; i < len(a); i++ {
        dot = dot + a[i]*b[i]
    }

    lenA := 0.0     // длины векторов
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
