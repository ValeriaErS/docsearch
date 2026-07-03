package corpus 
import(
	"fmt"
	"os"
	"path/filepath"
)
func LoadDocuments(path string)([]Document,error) { //читаем док,возврат списка
	files,err:=os.ReadDir(path)
	if err!=nil{
		return nil,err
	}
	var documents []Document//Пустой список 
	
	for _,file:=range files {
		if filepath.Ext(file.Name()) !=".md"&& filepath.Ext(file.Name()) !=".txt"{ 
			continue
		}
			text,err:=os.ReadFile(filepath.Join(path,file.Name())) //читаю
			if err!=nil{
				return nil,err
			
	}
	document:=Document{
		Name:file.Name(),
		Text:NormalizeNext(string(text)),
	}
	fmt.Println("Загружен документ:", document.Name)
	documents=append(documents,document)
}
 return documents, nil
}

/*package loader 
func LoadDocuments(path string)([]Document,error)//загрузка документов
return nil,nil
}*/
/*
func LoadDocuments(path string) ([]string, error) {
    files, _ := os.ReadDir(path)
    var texts []string
    for _, f := range files {
        data, _ := os.ReadFile(f.Name())
        texts = append(texts, string(data))
    }
    return texts, nil
}
*/
