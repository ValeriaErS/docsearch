package agent

import (
    "context"
    "fmt"
	"docsearch/internal/rag"
)

type Tool interface { //  интерфейс для инструментов агента
    Name() string
    Description() string
    Execute(ctx context.Context, input string) (string, error)
}

type SearchTool struct {  //  инструмент для поиска
    agent *Agent
}

func (t *SearchTool) Name() string {
    return "search"
}

func (t *SearchTool) Description() string {
    return "Поиск информации в документации"
}

func (t *SearchTool) Execute(ctx context.Context, input string) (string, error) {
    _, _, _, answer, _, _, _, _ := rag.Ask(ctx, *t.agent.cfg, input, "", []map[string]string{}, t.agent.vectorClient)
    return answer, nil
}

type CalculatorTool struct{}  //  инструмент для вычислений

func (t *CalculatorTool) Name() string {
    return "calculate"
}

func (t *CalculatorTool) Description() string {
    return "Выполнение математических вычислений"
}

func (t *CalculatorTool) Execute(ctx context.Context, input string) (string, error) {
    return fmt.Sprintf("Результат: %s", input), nil
}

type SummaryTool struct{}  //  инструмент для суммаризации

func (t *SummaryTool) Name() string {
    return "summarize"
}

func (t *SummaryTool) Description() string {
    return "Суммаризация текста"
}

func (t *SummaryTool) Execute(ctx context.Context, input string) (string, error) {
    if len(input) > 200 {
        return input[:200] + "...", nil
    }
    return input, nil
}

type ToolManager struct {
    tools map[string]Tool
}

func NewToolManager(agent *Agent) *ToolManager {
    tm := &ToolManager{
        tools: make(map[string]Tool),
    }
   
    tm.Register(&SearchTool{agent: agent})
    tm.Register(&CalculatorTool{})
    tm.Register(&SummaryTool{})
    return tm
}

func (tm *ToolManager) Register(tool Tool) {
    tm.tools[tool.Name()] = tool
}

func (tm *ToolManager) GetTool(name string) (Tool, bool) {
    tool, ok := tm.tools[name]
    return tool, ok
}

func (tm *ToolManager) ListTools() []string {
    names := []string{}
    for name := range tm.tools {
        names = append(names, name)
    }
    return names
}

func (tm *ToolManager) ExecuteTool(ctx context.Context, name, input string) (string, error) {  //  выполняет инструмент
    tool, ok := tm.GetTool(name)
    if !ok {
        return "", fmt.Errorf("инструмент '%s' не найден", name)
    }
    return tool.Execute(ctx, input)
}