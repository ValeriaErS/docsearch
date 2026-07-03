package rag
import(
	"fmt"
    "docsearch/internal/config"
    "docsearch/internal/corpus"
    "docsearch/internal/chunk"
    "docsearch/internal/embed"
    "docsearch/internal/store"
    "docsearch/internal/retrieve"
)
func Index(cfg config.Config) { 
	docs,err:=corpus.LoadDocuments(cfg.Corpus.Path) //загрузка доков
	if err!=nil {
		fmt.Println("Ошибка загрузки",err)
		return
	}
	fmt.Println("Количество документов",len(docs))
	allText:=[]string{}
	allVectors:=[][]float64{}
	 
	 for i:=0;i<len(docs);i++{
		doc:=docs[i]
		fmt.Println("Файл",doc.Name)

	parts:=chunk.SplitText(doc.Text,cfg.Chunking.MaxTokens,cfg.Chunking.OverlapTokens) //бьею на чанки
	fmt.Println("Количество чанков",len(parts))
	 
	for j:=0;j<len(parts);j++{
		one:=parts[j]
		vec,err:=embed.GetEmbedding(one.Text) //получаю вектор
		if err!=nil{
			fmt.Println("КОшибка",err)
			continue
		}
		allText,allVectors=store.Add(allText,allVectors,one.Text,vec) //хранилище
	}
	}
	fmt.Println("Всего чанков",len(allText))
	fmt.Println("Индексация настроена")
}
func Ask(cfg config.Config,question string)([]string,[]float64){  //поиск по вопросу
	docs,err:=corpus.LoadDocuments(cfg.Corpus.Path)
	
	if err!=nil{
			fmt.Println("КОшибка загрузки",err)
			return[]string{},[]float64{}
}
allText:=[]string{}
allVectors:=[][]float64{}
	
for i:=0;i<len(docs);i++{
		doc:=docs[i]
		parts:=chunk.SplitText(doc.Text,cfg.Chunking.MaxTokens,cfg.Chunking.OverlapTokens)
		
		for j:=0;j<len(parts);j++{
		one:=parts[j]
		vec,err:=embed.GetEmbedding(one.Text)
		
		if err!=nil{
			fmt.Println("Ошибка",err)
			continue
		}
		allText,allVectors=store.Add(allText,allVectors,one.Text,vec)
	}
}
questionVec,err:=embed.GetEmbedding(question)

if err!=nil{
			fmt.Println("Ошибка вектора вопроса",err)
			return[]string{},[]float64{}
}
foundTexts,foundScores:=retrieve.Search(allText,allVectors,questionVec,cfg.Retrieval.TopK) //поиск похожих чанков
return foundTexts,foundScores
}

