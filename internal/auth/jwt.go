package auth
import(
	"fmt"
	"os"
	"time"
    "github.com/golang-jwt/jwt/v4"
)

func MakeToken(username string)(string,error){  // Создаю токен
secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        return "", fmt.Errorf("JWT_SECRET не задан в .env файле")
    }

	claims := jwt.MapClaims{
	"user":username,
	"exp":time.Now().Add(time.Hour * 24).Unix(),
}	
token:=jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
return token.SignedString([]byte(secret))
}


func CheckToken(tokenString string)(string,error){ // Проверяю токен и достаю имя пользователя
	secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        return "", fmt.Errorf("JWT_SECRET не задан в .env файле")
    }
	
	token,err:=jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
})
if err!=nil{
	return "",err
}
claims,ok:=token.Claims.(jwt.MapClaims)
if !ok || !token.Valid{
	 return "", fmt.Errorf("неверный токен")
}
username,ok:=claims["user"].(string) // Беру имя пользователя
if !ok{
	 return "", fmt.Errorf("неверный токен: нет поля user")
}
return username,nil
}