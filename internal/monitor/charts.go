package monitor

import (
	"fmt"
	"os"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func (c *MetricCollector) GenerateCharts(outputDir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.metrics) == 0 {
		return fmt.Errorf("нет метрик для построения графиков")
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	if err := c.generateTimelineChart(outputDir); err != nil {
		return err
	}

	if err := c.generateBreakdownChart(outputDir); err != nil {
		return err
	}

	if err := c.generateTokensChart(outputDir); err != nil {
		return err
	}

	if err := c.generateSuccessChart(outputDir); err != nil {
		return err
	}

	if err := c.generateDashboard(outputDir); err != nil {
		return err
	}

	fmt.Printf("Графики сохранены в %s\n", outputDir)
	return nil
}

func (c *MetricCollector) generateTimelineChart(outputDir string) error {
	line := charts.NewLine()

	trueBool := true

	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "#ffffff",
			Theme:           "white",
			Width:           "100%",
			Height:          "400px",
		}),

		charts.WithColorsOpts(opts.Colors{
			"#D8A7B1",
		}),

		charts.WithTitleOpts(opts.Title{
			Title: "Время ответа по запросам",
			Left:  "center",
			TitleStyle: &opts.TextStyle{
				Color:      "#433845",
				FontSize:   18,
				FontWeight: "bold",
			},
		}),

		charts.WithTooltipOpts(opts.Tooltip{
			Show:            &trueBool,
			Trigger:         "axis",
			BackgroundColor: "#ffffff",
			BorderColor:     "#D9D4DA",
		}),

		charts.WithXAxisOpts(opts.XAxis{
			AxisLabel: &opts.AxisLabel{
				Show:     &trueBool,
				Color:    "#958899",
				FontSize: 12,
				Interval: "0",
			},
			AxisLine: &opts.AxisLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#D9D4DA",
				},
			},
		}),

		charts.WithYAxisOpts(opts.YAxis{
			AxisLabel: &opts.AxisLabel{
				Show:     &trueBool,
				Color:    "#958899",
				FontSize: 12,
			},
			AxisLine: &opts.AxisLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#D9D4DA",
				},
			},
			SplitLine: &opts.SplitLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#EDE8EF",
				},
			},
		}),

		charts.WithLegendOpts(opts.Legend{
			Show: &trueBool,
			Top:  "8%",
			Left: "center",
			TextStyle: &opts.TextStyle{
				Color: "#958899",
			},
		}),

		charts.WithGridOpts(opts.Grid{
			Left:         "15%",
			Right:        "8%",
			Top:          "22%",
			Bottom:       "18%",
			ContainLabel: &trueBool,
		}),
	)

	xAxis := []string{}
	totalData := []opts.LineData{}

	for i, m := range c.metrics {
		idx := i + 1
		xAxis = append(xAxis, fmt.Sprintf("#%d", idx))
		totalData = append(totalData, opts.LineData{
			Value: float64(m.TotalDurationMs) / 1000,
		})
	}

	line.SetXAxis(xAxis).
		AddSeries("Общее время", totalData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: &trueBool,
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#D8A7B1",
				Width: 3,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color: "rgba(216, 167, 177, 0.15)",
			}),
		)

	filename := fmt.Sprintf("%s/timeline.html", outputDir)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return line.Render(f)
}

func (c *MetricCollector) generateBreakdownChart(outputDir string) error {
	bar := charts.NewBar()

	trueBool := true

	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "#ffffff",
			Theme:           "white",
			Width:           "100%",
			Height:          "400px",
		}),

		charts.WithColorsOpts(opts.Colors{
			"#A89BB8",
			"#D8A7B1",
			"#C4B5C0",
		}),

		charts.WithTitleOpts(opts.Title{
			Title: "Разбивка времени по этапам",
			Left:  "center",
			TitleStyle: &opts.TextStyle{
				Color:      "#433845",
				FontSize:   18,
				FontWeight: "bold",
			},
		}),

		charts.WithTooltipOpts(opts.Tooltip{
			Show:            &trueBool,
			Trigger:         "axis",
			BackgroundColor: "#ffffff",
			BorderColor:     "#D9D4DA",
		}),

		charts.WithLegendOpts(opts.Legend{
			Show: &trueBool,
			Top:  "8%",
			Left: "center",
			TextStyle: &opts.TextStyle{
				Color: "#958899",
			},
		}),

		charts.WithXAxisOpts(opts.XAxis{
			AxisLabel: &opts.AxisLabel{
				Show:     &trueBool,
				Color:    "#958899",
				FontSize: 12,
				Interval: "0",
			},
			AxisLine: &opts.AxisLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#D9D4DA",
				},
			},
		}),

		charts.WithYAxisOpts(opts.YAxis{
			AxisLabel: &opts.AxisLabel{
				Show:     &trueBool,
				Color:    "#958899",
				FontSize: 12,
			},
			AxisLine: &opts.AxisLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#D9D4DA",
				},
			},
			SplitLine: &opts.SplitLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#EDE8EF",
				},
			},
		}),

		charts.WithGridOpts(opts.Grid{
			Left:         "15%",
			Right:        "8%",
			Top:          "22%",
			Bottom:       "18%",
			ContainLabel: &trueBool,
		}),
	)

	xAxis := []string{}
	embeddingData := []opts.BarData{}
	searchData := []opts.BarData{}
	llmData := []opts.BarData{}

	for i, m := range c.metrics {
		idx := i + 1
		xAxis = append(xAxis, fmt.Sprintf("#%d", idx))
		embeddingData = append(embeddingData, opts.BarData{
			Value: float64(m.EmbeddingDurationMs) / 1000,
		})
		searchData = append(searchData, opts.BarData{
			Value: float64(m.SearchDurationMs) / 1000,
		})
		llmData = append(llmData, opts.BarData{
			Value: float64(m.LLMDurationMs) / 1000,
		})
	}

	bar.SetXAxis(xAxis).
		AddSeries("Эмбеддинг", embeddingData).
		AddSeries("Поиск", searchData).
		AddSeries("LLM", llmData).
		SetSeriesOptions(
			charts.WithBarChartOpts(opts.BarChart{
				Stack: "stack",
			}),
		)

	filename := fmt.Sprintf("%s/breakdown.html", outputDir)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return bar.Render(f)
}

func (c *MetricCollector) generateTokensChart(outputDir string) error {
	line := charts.NewLine()

	trueBool := true

	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "#ffffff",
			Theme:           "white",
			Width:           "100%",
			Height:          "400px",
		}),

		charts.WithColorsOpts(opts.Colors{
			"#A89BB8",
		}),

		charts.WithTitleOpts(opts.Title{
			Title: "Использование токенов",
			Left:  "center",
			TitleStyle: &opts.TextStyle{
				Color:      "#433845",
				FontSize:   18,
				FontWeight: "bold",
			},
		}),

		charts.WithTooltipOpts(opts.Tooltip{
			Show:            &trueBool,
			Trigger:         "axis",
			BackgroundColor: "#ffffff",
			BorderColor:     "#D9D4DA",
		}),

		charts.WithXAxisOpts(opts.XAxis{
			AxisLabel: &opts.AxisLabel{
				Show:     &trueBool,
				Color:    "#958899",
				FontSize: 12,
				Interval: "0",
			},
			AxisLine: &opts.AxisLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#D9D4DA",
				},
			},
		}),

		charts.WithYAxisOpts(opts.YAxis{
			AxisLabel: &opts.AxisLabel{
				Show:     &trueBool,
				Color:    "#958899",
				FontSize: 12,
			},
			AxisLine: &opts.AxisLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#D9D4DA",
				},
			},
			SplitLine: &opts.SplitLine{
				Show: &trueBool,
				LineStyle: &opts.LineStyle{
					Color: "#EDE8EF",
				},
			},
		}),

		charts.WithLegendOpts(opts.Legend{
			Show: &trueBool,
			Top:  "8%",
			Left: "center",
			TextStyle: &opts.TextStyle{
				Color: "#958899",
			},
		}),

		charts.WithGridOpts(opts.Grid{
			Left:         "15%",
			Right:        "8%",
			Top:          "22%",
			Bottom:       "18%",
			ContainLabel: &trueBool,
		}),
	)

	xAxis := []string{}
	tokensData := []opts.LineData{}

	for i, m := range c.metrics {
		idx := i + 1
		xAxis = append(xAxis, fmt.Sprintf("#%d", idx))
		tokensData = append(tokensData, opts.LineData{
			Value: m.TokensUsed,
		})
	}

	line.SetXAxis(xAxis).
		AddSeries("Токены", tokensData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: &trueBool,
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#A89BB8",
				Width: 3,
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Color: "rgba(168, 155, 184, 0.15)",
			}),
		)

	filename := fmt.Sprintf("%s/tokens.html", outputDir)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return line.Render(f)
}

func (c *MetricCollector) generateSuccessChart(outputDir string) error {
	pie := charts.NewPie()

	trueBool := true

	pie.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "#ffffff",
			Theme:           "white",
			Width:           "100%",
			Height:          "400px",
		}),

		charts.WithColorsOpts(opts.Colors{
			"#D8A7B1",
			"#D9D4DA",
		}),

		charts.WithTitleOpts(opts.Title{
			Title: "Успешность запросов",
			Left:  "center",
			TitleStyle: &opts.TextStyle{
				Color:      "#433845",
				FontSize:   18,
				FontWeight: "bold",
			},
		}),

		charts.WithTooltipOpts(opts.Tooltip{
			Show:            &trueBool,
			BackgroundColor: "#ffffff",
			BorderColor:     "#D9D4DA",
		}),

		charts.WithLegendOpts(opts.Legend{
			Show:   &trueBool,
			Bottom: "2%",
			Left:   "center",
			TextStyle: &opts.TextStyle{
				Color: "#958899",
			},
		}),
	)

	successCount := 0
	failCount := 0
	for _, m := range c.metrics {
		if m.Success {
			successCount++
		} else {
			failCount++
		}
	}

	pieData := []opts.PieData{
		{Name: fmt.Sprintf("Успешно (%d)", successCount), Value: successCount},
	}
	if failCount > 0 {
		pieData = append(pieData, opts.PieData{
			Name:  fmt.Sprintf("Ошибка (%d)", failCount),
			Value: failCount,
		})
	}

	pie.AddSeries("Статус", pieData).
		SetSeriesOptions(
			charts.WithPieChartOpts(opts.PieChart{
				Radius: []string{"42%", "72%"},
			}),
			charts.WithLabelOpts(opts.Label{
				Show:      &trueBool,
				Position:  "inside",
				Formatter: "{d}%",
				Color:     "#433845",
				FontSize:  14,
				FontWeight: "bold",
			}),
		)

	filename := fmt.Sprintf("%s/success.html", outputDir)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return pie.Render(f)
}

func (c *MetricCollector) generateDashboard(outputDir string) error {
	htmlContent := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DocSearch — Дашборд метрик</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Arial, sans-serif;
            background: #faf8fb;
            color: #3f3744;
            padding: 32px;
        }
        .dashboard { max-width: 1400px; margin: 0 auto; }
        h1 { color: #332b36; font-size: 30px; font-weight: 650; margin-bottom: 8px; }
        .subtitle { color: #8b7d8f; font-size: 15px; margin-bottom: 30px; }
        .stats {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 18px;
            margin-bottom: 28px;
        }
        .stat-card {
            background: #ffffff;
            border: 1px solid #eee5ef;
            border-radius: 14px;
            padding: 20px 22px;
            box-shadow: 0 3px 12px rgba(100, 75, 100, 0.05);
        }
        .stat-card .label { color: #958899; font-size: 12px; text-transform: uppercase; letter-spacing: 0.7px; font-weight: 600; }
        .stat-card .value { color: #3c3340; font-size: 27px; font-weight: 650; margin-top: 7px; }
        .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
        .chart-container {
            background: #ffffff;
            border: 1px solid #eee5ef;
            border-radius: 14px;
            padding: 18px;
            min-height: 450px;
            box-shadow: 0 3px 12px rgba(100, 75, 100, 0.05);
        }
        .chart-container h3 { color: #433845; font-size: 16px; font-weight: 600; margin-bottom: 12px; }
        .chart-container iframe { display: block; width: 100%; height: 400px; border: none; background: #ffffff; }
        @media (max-width: 1000px) { .stats { grid-template-columns: repeat(2, 1fr); } .grid { grid-template-columns: 1fr; } }
        @media (max-width: 600px) { body { padding: 18px; } .stats { grid-template-columns: 1fr; } }
    </style>
</head>
<body>
    <div class="dashboard">
        <h1> DocSearch — Дашборд метрик</h1>
        <div class="subtitle">Мониторинг производительности RAG-системы</div>
        <div class="stats">
            <div class="stat-card">
                <div class="label">Всего запросов</div>
                <div class="value">` + fmt.Sprintf("%d", len(c.metrics)) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">Среднее время</div>
                <div class="value">` + fmt.Sprintf("%.2fс", c.getAvgDuration()) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">Среднее токенов</div>
                <div class="value">` + fmt.Sprintf("%d", c.getAvgTokens()) + `</div>
            </div>
            <div class="stat-card">
                <div class="label">Успешность</div>
                <div class="value">` + fmt.Sprintf("%.0f%%", c.getSuccessRate()) + `</div>
            </div>
        </div>
        <div class="grid">
            <div class="chart-container">
                <h3> Время ответа</h3>
                <iframe src="timeline.html"></iframe>
            </div>
            <div class="chart-container">
                <h3> Разбивка по этапам</h3>
                <iframe src="breakdown.html"></iframe>
            </div>
            <div class="chart-container">
                <h3> Токены</h3>
                <iframe src="tokens.html"></iframe>
            </div>
            <div class="chart-container">
                <h3> Успешность</h3>
                <iframe src="success.html"></iframe>
            </div>
        </div>
    </div>
</body>
</html>`

	filename := fmt.Sprintf("%s/dashboard.html", outputDir)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(htmlContent)
	return err
}

func (c *MetricCollector) getAvgDuration() float64 {
	if len(c.metrics) == 0 {
		return 0
	}
	var total int64
	for _, m := range c.metrics {
		total += m.TotalDurationMs
	}
	return float64(total) / float64(len(c.metrics)) / 1000
}

func (c *MetricCollector) getAvgTokens() int {
	if len(c.metrics) == 0 {
		return 0
	}
	var total int
	for _, m := range c.metrics {
		total += m.TokensUsed
	}
	return total / len(c.metrics)
}

func (c *MetricCollector) getSuccessRate() float64 {
	if len(c.metrics) == 0 {
		return 0
	}
	success := 0
	for _, m := range c.metrics {
		if m.Success {
			success++
		}
	}
	return float64(success) / float64(len(c.metrics)) * 100
}