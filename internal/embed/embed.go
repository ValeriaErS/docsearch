package embed 
import ("strings")
func GetVector(text string,words []string) []float64{ //превращение текста в вектор
	vector:=make([]float64,len(words))
	parts:=strings.Split(text,"") //разбиваю на слова
	
	for i:=0;i<len(parts);i++{
	word:=strings.ToLower(parts[i])

	for j:=0;j<len(words);j++{
		if words[j]==word {
			vector[j]=vector[j]+1
			break
			}
		}
	}
	return vector
}
/*package embed

import ("fmt")
func GetEmbedding(text string) []float32 {
	fmt.Println("эмбеддинг:", len(text))

	vector := []float32{0.1, 0.2, 0.3, 0.4}

	return vector
}
	*/
	
