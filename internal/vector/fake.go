package vector
import "context" 

type FakeVectorStore struct{
	Points []map[string]interface{} //чанки храним в памяти
}
func NewFakeVectorStore() *FakeVectorStore{ //фейк клиент
	return &FakeVectorStore{
		Points:[]map[string]interface{}{},
	}
}
func (f *FakeVectorStore) Search(ctx context.Context, name string, vec []float32, limit int, userID string)([] map[string]interface{},error){ //похожие чанки
	if len(f.Points)==0{
		return []map[string]interface{}{},nil
	}
	result:=[]map[string]interface{}{} //возрат первых штук
	for i:=0;i<limit && i<len(f.Points);i++{
		result=append(result,f.Points[i])
	}
	return result,nil
}
func (f *FakeVectorStore) Save(ctx context.Context, name string, id string, vec []float32, data map[string] interface{}) error {  //в память сохраняю
f.Points=append(f.Points,map[string]interface{}{
	"id": id,
    "vector": vec,
    "payload": data,
    "score": 0.95,
})
return nil
}

func (f *FakeVectorStore) Delete(ctx context.Context, name string, filter map[string]interface{}) error {
    f.Points = []map[string]interface{}{}
    return nil
}

func (f *FakeVectorStore) CreateCollection(ctx context.Context, name string) error {
	return nil
}

func (f *FakeVectorStore) Ping(ctx context.Context) error {
    return nil
}