package db
import (
	"fmt"
    "os"
	"database/sql"
    "github.com/joho/godotenv"
	_"github.com/lib/pq" 
)
type DB struct{
	Conn *sql.DB
}
func NewDB() (*DB,error){
	godotenv.Load()
	connStr:=os.Getenv("DATABASE_URL")
	conn,err:=sql.Open("postgres",connStr)
	if err!=nil{
		return nil,err
	}
	err=conn.Ping()
	if err!=nil{
		return nil,err
	}
	fmt.Println("база работает")

	return &DB{Conn:conn},nil
}
func (d *DB) Close(){
	d.Conn.Close()
}