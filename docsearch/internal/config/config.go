package config

import (
	"os"
	"gopkg.in/yaml.v2"
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
		Provider string `yaml:"provider"`
		Model    string `yaml:"model"`
		BaseURL  string `yaml:"base_url"`
	} `yaml:"embeddings"`

	Retrieval struct {
		TopK     int     `yaml:"top_k"`
		MinScore float64 `yaml:"min_score"`
	} `yaml:"retrieval"`

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

	return &cfg, nil
}