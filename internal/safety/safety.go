package safety

import (
    "fmt"
    "path/filepath"
    "strings"
    "os"
    "regexp"
)

func SanitizeAndValidateUser(username string) (string, error) {  //проверка имени пользователя и путей
    
    re := regexp.MustCompile(`[^a-zA-Zа-яА-Я0-9_ -]`)  //очистка имени
    safe := re.ReplaceAllString(strings.TrimSpace(username), "")
    if len(safe) == 0 || len(safe) > 100 {
        return "", fmt.Errorf("некорректное имя пользователя")
    }

    
    base, err := filepath.Abs("docs")  //проверка пути
    if err != nil {
        return "", err
    }
    userDir := filepath.Join(base, safe)
    absUserDir, err := filepath.Abs(userDir)
    if err != nil {
        return "", err
    }

    
    if !strings.HasPrefix(absUserDir, base+string(os.PathSeparator)) && absUserDir != base {
        return "", fmt.Errorf("попытка выйти за пределы docs")
    }

    return safe, nil
}