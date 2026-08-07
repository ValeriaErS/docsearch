package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	Corpus struct {
		Path    string   `yaml:"path"`
		Formats []string `yaml:"formats"`
	} `yaml:"corpus"`

	Chunking struct {
		MaxTokens     int `yaml:"max_tokens"`
		OverlapTokens int `yaml:"overlap_tokens"`
	} `yaml:"chunking"`

	Embeddings struct {
		Provider   string `yaml:"provider"`
		Model      string `yaml:"model"`
		BaseURL    string `yaml:"base_url"`
		VectorSize int    `yaml:"vector_size"`
	} `yaml:"embeddings"`

	Retrieval struct {
		TopK            int     `yaml:"top_k"`
		MinScore        float64 `yaml:"min_score"`
		EnableRewriting bool    `yaml:"enable_rewriting"`
		EnableHyDE      bool    `yaml:"enable_hyde"`
		HybridSearch    bool    `yaml:"hybrid_search"`
		EnableRerank    bool    `yaml:"enable_rerank"`
		EnableMultiQuery bool    `yaml:"enable_multi_query"`
		EnableCompression bool    `yaml:"enable_compression"`
	} `yaml:"retrieval"`

	Cache struct {
		EnableEmbeddingCache bool `yaml:"enable_embedding_cache"`
		EnableSearchCache    bool `yaml:"enable_search_cache"`
		TTLHours             int  `yaml:"ttl_hours"`
	} `yaml:"cache"`

	Validation struct {
		EnableCitationValidator bool `yaml:"enable_citation_validator"`
		RemoveInvalidCitations  bool `yaml:"remove_invalid_citations"`
		EnableHallucinationDetection bool `yaml:"enable_hallucination_detection"`
		WarnOnHallucination      bool `yaml:"warn_on_hallucination"`
	} `yaml:"validation"`

	LLM struct {
		Provider    string  `yaml:"provider"`
		Model       string  `yaml:"model"`
		BaseURL     string  `yaml:"base_url"`
		Temperature float64 `yaml:"temperature"`
		MaxTokens   int     `yaml:"max_tokens"`
	} `yaml:"llm"`
}

func LoadConfig(path string) (*Config, error) { //читает файл config.yml
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Embeddings.VectorSize <= 0 {
		return nil, fmt.Errorf("embeddings.vector_size должен быть больше 0 (сейчас: %d)", cfg.Embeddings.VectorSize)
	}
	if cfg.Retrieval.TopK <= 0 {
		return nil, fmt.Errorf("retrieval.top_k должен быть больше 0 (сейчас: %d)", cfg.Retrieval.TopK)
	}
	if cfg.Retrieval.MinScore < 0 || cfg.Retrieval.MinScore > 1 {
		return nil, fmt.Errorf("retrieval.min_score должен быть в [0,1] (сейчас: %.2f)", cfg.Retrieval.MinScore)
	}

	if cfg.Chunking.MaxTokens <= 0 {
		return nil, fmt.Errorf("chunking.max_tokens должен быть больше 0 (сейчас: %d)", cfg.Chunking.MaxTokens)
	}
	if cfg.Chunking.OverlapTokens < 0 {
		return nil, fmt.Errorf("chunking.overlap_tokens не может быть отрицательным (сейчас: %d)", cfg.Chunking.OverlapTokens)
	}

	return &cfg, nil
}
