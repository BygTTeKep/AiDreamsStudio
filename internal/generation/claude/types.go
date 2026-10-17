package claude

// Пределы длительности кадра, в которых генерирует видео-модель.
const (
	MinSceneDurationSec = 3
	MaxSceneDurationSec = 10
)

// ScenarioJob — задача на генерацию сценариев для проекта.
type ScenarioJob struct {
	ProjectID string
	UserID    int

	ProductDescription string
	TargetAudience     string
	PainPoints         []string // боли ЦА; если пусто — Claude подберёт сам
	Count              int      // сколько сценариев нужно
	ScenesPerScenario  int      // кадров в сценарии
	SceneDurationSec   int      // средняя длительность кадра; общий хронометраж = ScenesPerScenario * SceneDurationSec
	Language           string   // язык текста сценариев

	Images []ProductImage // картинки продукта, уходят в запрос как base64
}

type ProductImage struct {
	MediaType string // image/jpeg | image/png | image/webp | image/gif
	Data      []byte
}

// GeneratedScenario — один сценарий из ответа Claude.
// JSON-теги совпадают со схемой scenarioSchema.
type GeneratedScenario struct {
	PainPoint string           `json:"pain_point"`
	Title     string           `json:"title"`
	Scenes    []GeneratedScene `json:"scenes"`
}

type GeneratedScene struct {
	Position    int    `json:"position"`
	DurationSec int    `json:"duration_sec"`
	Description string `json:"description"`
}

type generationResponse struct {
	Scenarios []GeneratedScenario `json:"scenarios"`
}

// scenarioSchema — JSON Schema для structured outputs: ответ модели
// гарантированно ей соответствует, поэтому его можно сразу парсить в generationResponse.
var scenarioSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"scenarios": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pain_point": map[string]any{
						"type":        "string",
						"description": "Боль целевой аудитории, которую закрывает сценарий",
					},
					"title": map[string]any{
						"type":        "string",
						"description": "Короткое название сценария",
					},
					"scenes": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"position": map[string]any{
									"type":        "integer",
									"description": "Порядковый номер кадра, начиная с 1",
								},
								"duration_sec": map[string]any{
									"type":        "integer",
									"description": "Длительность кадра в секундах, от 3 до 10",
								},
								"description": map[string]any{
									"type":        "string",
									"description": "Что происходит в кадре: действие, ракурс, движение камеры, обстановка, текст на экране",
								},
							},
							"required":             []string{"position", "duration_sec", "description"},
							"additionalProperties": false,
						},
					},
				},
				"required":             []string{"pain_point", "title", "scenes"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"scenarios"},
	"additionalProperties": false,
}
