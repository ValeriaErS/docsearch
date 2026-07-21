package corpus 
import(
	"os"
	"fmt"
	"path/filepath"
	"github.com/ledongthuc/pdf"
	"strings"  
)

func readPDF(path string) (string, map[int]string, error){   // читаю PDF файл и достаю из него текст
	file,reader,err:=pdf.Open(path)
	if err!=nil{
		return "",nil,err
	}
	defer file.Close()

	var fullText strings.Builder
	pages := make(map[int]string)

	for i:=1;i<=reader.NumPage();i++{         // прохожу по всем страницам
	page:=reader.Page(i)
	if page.V.IsNull(){
		continue
	}
	content,err:=page.GetPlainText(nil)
	if err!=nil{
		continue
	}
	pages[i] = content
		fullText.WriteString(content)
		fullText.WriteString("\n")
	}
	return fullText.String(), pages, nil
}

func LoadDocuments(path string)([]Document,error) { //читаем док,возврат списка
	files,err:=os.ReadDir(path)
	if err!=nil{
		return nil,err
	}
	var docs []Document  //Пустой список 
	
	for _,file:=range files {
		if file.IsDir () { 
			continue
		}
	name:=file.Name()
	ext:=filepath.Ext(name)

	if ext!=".md" && ext!=".txt" && ext!=".pdf"{
		continue
	}
fullPath:=filepath.Join(path,name)
var text string
var pages map[int]string

if ext==".pdf"{
	text, pages, err = readPDF(fullPath) 
	fmt.Printf("Документ %s: %d страниц\n", name, len(pages))  
}else {
	data,err:=os.ReadFile(fullPath)
	if err!=nil{
		return nil,err
	}
	text=string(data)
	pages = nil
}
if err!=nil{
	return nil,err
}
doc:=Document{  // создаю документ и нормализую текст
	Name:name,
	Text:NormalizeNext(text),
	Pages: pages,
}
docs=append(docs,doc)

}
return docs,nil
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
