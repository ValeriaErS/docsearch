package chunk
type Chunk struct{
	Text string
}
	func SplitText(text string,size int,overlap int)[]Chunk{ //Разбиваем текст на куски
		var result[]Chunk

		if text==""{ //если пустой,меньше 0 текст-на выход
			return result
		}
		if size<=0{
				return result
		}
		if len(text)<=size{
			result=append(result,Chunk{Text:text})
			return result
		}
		start:=0
			 
		for start < len(text){
				end:=start+size

			if end>len(text) { //обрезка при выходе за границу
				end=len(text)
			}
			part:=text[start:end]
			result=append(result,Chunk{Text:part})

			start=start+size-overlap
			if start>=len(text){
				break
			}
	
	}
	return result
}
/* func SplitText(text string, size int) []string {
    var result []string
    for i := 0; i < len(text); i += size {
        end := i + size
        if end > len(text) {
            end = len(text)
        }
        result = append(result, text[i:end])
    }
    return result
}
*/
/*func SplitText(text string, size int, overlap int) []string {
    var result []string
    start := 0
    for start < len(text) {
        end := start + size
        if end > len(text) {
            end = len(text)
        }
        result = append(result, text[start:end])
        start = start + size - overlap
    }
    return result
}
*/