package indexer
import(
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

type EmbeddingCache struct{
	CacheDir string
}
func NewEmbeddingCache() *EmbeddingCache{
	cacheDir:=".cache/embeddings"
	os.MkdirAll(cacheDir,0755)
	return &EmbeddingCache{CacheDir:cacheDir}
}
func (c *EmbeddingCache) Get(text string) ([]float64,bool){
	key:=c.hashKey(text)
	path:=filepath.Join(c.CacheDir, key+".json")

	data,err:=os.ReadFile(path)
	if err!=nil{
		return nil,false
	}
	var embedding []float64
	if err:=json.Unmarshal(data, &embedding); err!=nil{
		return nil,false
	}
	return embedding,true
}
func (c *EmbeddingCache) Save(text string,embedding []float64) error{
	key:=c.hashKey(text)
	path:=filepath.Join(c.CacheDir,key+".json")

	data,err:=json.Marshal(embedding)
	if err!=nil{
		return err
	}
	return os.WriteFile(path,data,0644)

}
func (c *EmbeddingCache) hashKey(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}