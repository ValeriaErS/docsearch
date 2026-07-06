package chunk

import "testing"

func TestSplitText(t *testing.T) {            //проверка что режется текст
    text:= "Этот текст тестовый"
    size:= 20
    overlap:= 5
    docName:= "test.md"

    chunks:= SplitText(text, size, overlap, docName)

    if len(chunks) == 0 {
        t.Error("Первый чанк пустой, не годится")
    }

    if len(chunks[0].Text) == 0 {
        t.Error("Ошибка: первый чанк пустой")
    }
}

func TestSplitTextEmpty(t *testing.T) {              //проверяю что пустой текст не ломает ничего
    docName:= "test.md"
    chunks:= SplitText("", 20, 5, docName)

    if len(chunks) != 0 {
        t.Errorf("для пустого чанка должно быть 0, а у тебя %d", len(chunks))
    }
}

func TestSplitTextSizeZero(t *testing.T) {         //проверяю что размер нулевой не ломает ничего
    docName:= "test.md"
    text:= "маленький текст"
    size:= 0
    overlap:= 5
    docName:= "test.md"

    chunks:= SplitText(text, size, overlap, docName)

    if len(chunks) != 0 {
        t.Errorf("при размере 0, надо 0 чанков, а у тебя %d", len(chunks))
    }
}