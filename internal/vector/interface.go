package vector

import "context"

type VectorStore interface {
	Search(ctx context.Context, name string, vec []float32, limit int, userID string) ([]map[string]interface{}, error)
	Save(ctx context.Context, name string, id string, vec []float32, data map[string]interface{}) error
	Delete(ctx context.Context, name string, filter map[string]interface{}) error
	CreateCollection(ctx context.Context, name string) error
	Ping(ctx context.Context) error
	GetAllVectors(ctx context.Context, name string, userID string) ([]map[string]interface{}, error)
	SearchText(ctx context.Context, name string, query string, limit int, userID string) ([]map[string]interface{}, error) 
}
