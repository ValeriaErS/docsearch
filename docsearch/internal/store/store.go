package store

type ChunkWithVector struct {
    Text string
    Document string
    Vector []float64
}

type Store struct {
    Items []ChunkWithVector    // список всех чанков с векторами
}

func Add(store Store, text string, docName string, vector []float64) Store {   // добавляет новый чанк с его вектором в хранилище
    store.Items = append(store.Items, ChunkWithVector{
        Text: text,
        Document: docName,
        Vector: vector,
    })
    return store
}

/*package store
import(
    "math"
)
type ChunkWithVector struct{   //хранение вектора и текста
    Text string
    Vector []float64
}
type Store struct{    //список структур
    Items []ChunkWithVector
}

package store
import "math"
func AddItem(texts []string,vectors [][]float64,text string,vector []float64) 
([]string,[][]float64) {
    texts = append(texts, text)
    vectors = append(vectors, vector)
    return texts, vectors
}
func FindSimilar(texts[]string,vectors[][]float64,query[]float64,TopK int)
[]string {
    score:=[]float64{}

        for i:=0;i<len(vectors);i++{
            scores=append(scores,similerity(query,vectors[i]))
        }

        for i:=0;i<len(scores);i++{
            for j:=i+1;j<len(scores);j++{
                if scores[i]<scores[j] {
                    scores[i],scores[j]= scores[j],scores[i]
                    texts[i], texts[j] = texts[j], texts[i]
                }
            }    
        }
        result:=[]string{}
            for i:=0;i<TopK && i<len(texts);i++{
                result=append(result,texts[i])
            }
            return result
        }
        func similerity(a []float64,b []float64)float64{
            if len(a)!=len(b){
                return 0
            }
            dot:=0.0
            for i:=0;i<len(a);i++{
                dot= dot+a[i]*b[i]
            }
            lenA:=0.0
            lenB:=0.0
            for i:=0;i<len(a);i++{
                lenA=lenA+a[i]*a[i]
                lenB=lenB+b[i]*b[i]

        }
        lenA = math.Sqrt(lenA)
    lenB = math.Sqrt(lenB)
    if lenA == 0 || lenB == 0 {
        return 0
    }
    return dot / (lenA * lenB)
}*/


