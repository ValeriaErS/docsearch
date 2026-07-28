package corpus

import (
	"regexp"
	"strings"
)

func NormalizeNext(text string) string {
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.TrimSpace(text) //убираю пробелы, каретку
	return text
}
func RemoveHTMLTags(text string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(text, "")
}
