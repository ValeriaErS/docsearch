package auth
import(
	"time"
    "github.com/golang-jwt/jwt/v4"
)
var secretKey=[]byte("my-secret-key")
func MakeToken(username string)(string,error){  // Создаю токен
data:=jwt.MapClaims{
	"user":username,
	"exp":time.Now().Add(time.Hour * 24).Unix(),
}	
token:=jwt.NewWithClaims(jwt.SigningMethodHS256,data)
return token.SignedString(secretKey)
}
func CheckToken(tokenString string)(string,error){ // Проверяю токен и достаю имя пользователя
	token,err:=jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
        return secretKey, nil
})
if err!=nil{
	return "",err
}
claims,ok:=token.Claims.(jwt.MapClaims)
if !ok || !token.Valid{
	 return "", jwt.ErrInvalidKey
}
name,ok:=claims["user"].(string) // Беру имя пользователя
if !ok{
	 return "", jwt.ErrInvalidKey
}
return name,nil
}