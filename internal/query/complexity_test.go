package query

import (
    "testing"
)

func TestClassifyComplexity(t *testing.T) {
    tests := []struct {
        name     string
        query    string
        expected Complexity
    }{
        {
            name:     "короткое определение",
            query:    "Что такое RAG?",
            expected: ComplexitySimple,
        },
        {
            name:     "очень короткий запрос",
            query:    "FileAuditor",
            expected: ComplexitySimple,
        },
        {
            name:     "короткий запрос с определением",
            query:    "Что такое FileAuditor?",
            expected: ComplexitySimple,
        },
        {
            name:     "запрос с кто",
            query:    "Кто создал DocSearch?",
            expected: ComplexitySimple,
        },
        {
            name:     "запрос с и",
            query:    "RAG и векторный поиск",
            expected: ComplexitySimple,
        },
        {
            name:     "вопрос как",
            query:    "Как установить программу?",
            expected: ComplexityMedium,
        },
        {
            name:     "вопрос почему",
            query:    "Почему не работает поиск?",
            expected: ComplexityMedium,
        },
        {
            name:     "запрос с настройка",
            query:    "Настройка подключения к базе данных",
            expected: ComplexityMedium,
        },
        {
            name:     "запрос с инструкция",
            query:    "Инструкция по установке компонента",
            expected: ComplexityMedium,
        },
        {
            name:     "запрос с объясни",
            query:    "Объясни, как работает поиск",
            expected: ComplexityMedium,
        },
        {
            name:     "сравнение двух объектов",
            query:    "Сравни RAG и FileAuditor",
            expected: ComplexityComplex,
        },
        {
            name:     "сравнение с отличие",
            query:    "В чем отличие RAG от FileAuditor?",
            expected: ComplexityComplex,
        },
        {
            name:     "сравнение с разница",
            query:    "Какая разница между векторным и текстовым поиском?",
            expected: ComplexityComplex,
        },
        {
            name:     "два вопроса",
            query:    "Что такое RAG? И как он работает?",
            expected: ComplexityComplex,
        },
        {
            name:     "длинный запрос",
            query:    "Объясни, как работает механизм поиска документов в системе, какие компоненты участвуют и как они взаимодействуют между собой",
            expected: ComplexityComplex,
        },
        {
            name:     "сравнение с versus",
            query:    "RAG versus SearchServer",
            expected: ComplexityComplex,
        },
        {
            name:     "сравнение с vs без пробела",
            query:    "RAG vsFileAuditor",
            expected: ComplexityComplex,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ClassifyComplexity(tt.query)
            if result != tt.expected {
                t.Errorf("ClassifyComplexity(%q) = %v, ожидалось %v", tt.query, result, tt.expected)
            }
        })
    }
}

func TestGetRetrievalStrategy(t *testing.T) {
    tests := []struct {
        complexity   Complexity
        expectedTopK int
    }{
        {ComplexitySimple, 20},
        {ComplexityMedium, 50},
        {ComplexityComplex, 100},
    }

    for _, tt := range tests {
        t.Run(string(tt.complexity), func(t *testing.T) {
            strategy := GetRetrievalStrategy(tt.complexity)
            if strategy.CandidateTopK != tt.expectedTopK {
                t.Errorf("CandidateTopK = %v, ожидалось %v", strategy.CandidateTopK, tt.expectedTopK)
            }
        })
    }
}

func TestClassifyComplexityEdgeCases(t *testing.T) {
    tests := []struct {
        name     string
        query    string
        expected Complexity
    }{
        {
            name:     "пустой запрос",
            query:    "",
            expected: ComplexityMedium,
        },
        {
            name:     "запрос с пробелами",
            query:    "   ",
            expected: ComplexityMedium,
        },
        {
            name:     "запрос с цифрами",
            query:    "12345",
            expected: ComplexitySimple,
        },
        {
            name:     "запрос с английскими словами",
            query:    "RAG system",
            expected: ComplexitySimple,
        },
        {
            name:     "запрос с что такое и 7 слов",
            query:    "Что такое RAG и как он работает?",
            expected: ComplexityMedium,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ClassifyComplexity(tt.query)
            if result != tt.expected {
                t.Errorf("ClassifyComplexity(%q) = %v, ожидалось %v", tt.query, result, tt.expected)
            }
        })
    }
}

func TestGetRetrievalStrategyEdgeCases(t *testing.T) {
    tests := []struct {
        complexity          Complexity
        expectedTopK        int
        expectedDescription string
    }{
        {ComplexitySimple, 20, "простой поиск"},
        {ComplexityMedium, 50, "гибридный поиск с реранкингом"},
        {ComplexityComplex, 100, "полный пайплайн"},
    }
    for _, tt := range tests {
        t.Run(string(tt.complexity), func(t *testing.T) {
            strategy := GetRetrievalStrategy(tt.complexity)
            if strategy.CandidateTopK != tt.expectedTopK {
                t.Errorf("CandidateTopK = %v, ожидалось %v", strategy.CandidateTopK, tt.expectedTopK)
            }
            if strategy.Description != tt.expectedDescription {
                t.Errorf("Description = %v, ожидалось %v", strategy.Description, tt.expectedDescription)
            }
        })
    }
}