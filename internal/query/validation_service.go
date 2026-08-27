package query

import (
    "context"
    "docsearch/internal/config"
    "docsearch/internal/llm"
    "log"
	"time"
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

    if !llm.ShouldUseLLM(query) {
        return &ValidationResult{
            Status: StatusValid,
            Reason: "запрос не требует LLM-проверки",
        }, nil
    }

    ctxWithTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()

    valid, err := ClassifyQuery(ctxWithTimeout, query, s.cfg)

    if err != nil {
        log.Printf("LLM классификатор недоступен: %v", err)
        return &ValidationResult{
            Status: StatusValid,
            Reason: "LLM недоступен, запрос пропущен",
        }, nil
    }

    if !valid {
        return &ValidationResult{
            Status: StatusInvalid,
            Reason: "Запрос не распознан. Пожалуйста, задайте более конкретный вопрос.",
        }, nil
    }

    return &ValidationResult{Status: StatusValid}, nil
}