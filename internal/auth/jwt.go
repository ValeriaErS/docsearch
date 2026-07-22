package auth
import(
	"os"
	"time"
    "github.com/golang-jwt/jwt/v4"
)

func MakeToken(username string)(string,error){  // Создаю токен
secret := []byte(os.Getenv("JWT_SECRET"))
    if len(secret) == 0 {
        secret = []byte("fallback-secret-key")
    }

	claims := jwt.MapClaims{
	"user":username,
	"exp":time.Now().Add(time.Hour * 24).Unix(),
}	
token:=jwt.NewWithClaims(jwt.SigningMethodHS256,claims)
return token.SignedString(secret)
}


func CheckToken(tokenString string)(string,error){ // Проверяю токен и достаю имя пользователя
	secret := []byte(os.Getenv("JWT_SECRET"))
    if len(secret) == 0 {
        secret = []byte("fallback-secret-key")
    }
	
	token,err:=jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
        return secret, nil
})
if err!=nil{
	return "",err
}
claims,ok:=token.Claims.(jwt.MapClaims)
if !ok || !token.Valid{
	 return "", jwt.ErrInvalidKey
}
username,ok:=claims["user"].(string) // Беру имя пользователя
if !ok{
	 return "", jwt.ErrInvalidKey
}
return username,nil
}