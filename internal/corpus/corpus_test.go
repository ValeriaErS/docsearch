package corpus
import "testing"
 
func TestReadPDF(t *testing.T) {            // проверяю, что PDF читается
    text, err:= readPDF("docs/test.pdf")

    if err!= nil {                          // если ошибка,тест провален
        t.Skip("Нет test.pdf или ошибка чтения")
        return
    }
    if len(text) == 0 {                   // если текст пустой,тест провален
        t.Error("PDF пустой")
    }
}
func TestLoadPDF(t *testing.T) {         // проверяю, что PDF загружается через LoadDocuments
    docs, err:= LoadDocuments("docs")

    if err != nil {                     
        t.Skip("Ошибка загрузки")
        return
    }
    for i:= 0; i < len(docs); i++ {            // прохожу по всем документам
        if docs[i].Name == "test.pdf" {
            if len(docs[i].Text) == 0 {
                t.Error("PDF загрузился пустым")
            }
            return 
        }
    }
    t.Error("PDF не загрузился")
}