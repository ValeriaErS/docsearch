package corpus

type Document struct {
	Name  string
	Text  string
	Pages map[int]string
	PageOffsets []int
}
func (d *Document) GetPageByPosition(pos int) int{  //возвращает номер страницы для позиции в тексте
	if d.PageOffsets==nil || len(d.PageOffsets)==0{
		return 1
	}
	for i:=len(d.PageOffsets)-1;i>=0;i--{
		if pos>=d.PageOffsets[i]{
			return i+1
		}
	}
	return 1
}
