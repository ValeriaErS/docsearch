package corpus
import "strings"
func NormalizeNext(text string)string{
	text=strings.ReplaceAll(text,"\r","")
	text=strings.TrimSpace(text) //убираю пробелы, каретку
	return text
}
