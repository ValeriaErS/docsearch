package query

import (
	"context"
	"docsearch/internal/config"
	"log"
)

type ValidationService struct {
	validator *QueryValidator
	cfg       *config.Config
}

func NewValidationService(cfg *config.Config) *ValidationService {
	return &ValidationService{
		validator: NewQueryValidator(),
		cfg:       cfg,
	}
}

func (s *ValidationService) Validate(ctx context.Context, query string) (*ValidationResult, error) {
	localResult := s.validator.Validate(query)

	if localResult.Status == StatusInvalid {
		return localResult, nil
	}

	if localResult.Status == StatusValid {
		return localResult, nil
	}

	valid, err := ClassifyQuery(ctx, query, s.cfg) //проверка через ллм

	if err != nil {
		log.Printf("LLM классификатор недоступен: %v, пропускаем запрос", err)
		return &ValidationResult{
			Status: StatusValid,
			Reason: "LLM классификатор недоступен",
		}, nil
	}

	if !valid {
		return &ValidationResult{
			Status: StatusInvalid,
			Reason: "Запрос не распознан как осмысленный запрос к документации. Пожалуйста, задайте более конкретный вопрос.",
		}, nil
	}

	return &ValidationResult{Status: StatusValid}, nil
}