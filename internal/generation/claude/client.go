package claude

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

var (
	ErrRefusal       = errors.New("claude refused the request")
	ErrTruncated     = errors.New("claude response truncated by max_tokens")
	ErrEmptyResponse = errors.New("claude returned empty response")
	ErrInvalidResult = errors.New("claude returned invalid scenarios")
)

const systemPrompt = `Ты — креативный директор и сценарист коротких рекламных видео для маркетплейсов и e-commerce.
По описанию и фотографиям продукта ты пишешь покадровые сценарии вертикальных роликов, каждый из которых закрывает одну конкретную боль целевой аудитории.

Требования к сценарию:
- Первый кадр — хук: за 1–2 секунды показывает боль или ситуацию, в которой зритель узнаёт себя.
- Далее продукт решает эту боль; последний кадр — результат и призыв к действию.
- Каждый кадр — самостоятельный визуальный эпизод, который можно сгенерировать как отдельное видео: опиши, кто/что в кадре, действие, обстановку, освещение, ракурс и движение камеры, текст на экране (если нужен).
- Продукт должен выглядеть так же, как на фотографиях: не меняй форму, цвет, упаковку и логотип.
- Сценарии в одном ответе должны закрывать разные боли и не повторять друг друга.
- Не обещай того, чего нет в описании продукта.`

type Config struct {
	APIKey    string
	Model     string
	MaxTokens int64
	Effort    string // low | medium | high | xhigh | max
}

type Client struct {
	api       anthropic.Client
	model     string
	maxTokens int64
	effort    anthropic.BetaOutputConfigEffort
}

func NewClient(cfg Config) *Client {
	var opts []option.RequestOption
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey)) // иначе SDK возьмёт ANTHROPIC_API_KEY
	}
	if cfg.Model == "" {
		cfg.Model = "claude-opus-5-5"
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 16000
	}
	if cfg.Effort == "" {
		cfg.Effort = "high"
	}
	return &Client{
		api:       anthropic.NewClient(opts...),
		model:     cfg.Model,
		maxTokens: cfg.MaxTokens,
		effort:    anthropic.BetaOutputConfigEffort(cfg.Effort),
	}
}

// GenerateScenarios отправляет один запрос в Claude и возвращает распарсенные сценарии.
func (c *Client) GenerateScenarios(ctx context.Context, job ScenarioJob) ([]GeneratedScenario, error) {
	content := make([]anthropic.BetaContentBlockParamUnion, 0, len(job.Images)+1)
	for _, img := range job.Images {
		content = append(content, anthropic.NewBetaImageBlock(anthropic.BetaBase64ImageSourceParam{
			Data:      base64.StdEncoding.EncodeToString(img.Data),
			MediaType: anthropic.BetaBase64ImageSourceMediaType(img.MediaType),
		}))
	}
	content = append(content, anthropic.NewBetaTextBlock(buildUserPrompt(job)))

	resp, err := c.api.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    []anthropic.BetaTextBlockParam{{Text: systemPrompt}},
		Messages:  []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(content...)},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: c.effort,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: scenarioSchema},
		},
		// при отказе по политикам запрос переобслужит fallback-модель в рамках того же вызова
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	})
	if err != nil {
		return nil, fmt.Errorf("claude request: %w", err)
	}

	switch resp.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return nil, fmt.Errorf("%w: %s", ErrRefusal, resp.StopDetails.Explanation)
	case anthropic.BetaStopReasonMaxTokens:
		return nil, ErrTruncated
	}

	var raw strings.Builder
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			raw.WriteString(text.Text)
		}
	}
	if raw.Len() == 0 {
		return nil, ErrEmptyResponse
	}

	var out generationResponse
	if err := json.Unmarshal([]byte(raw.String()), &out); err != nil {
		return nil, fmt.Errorf("parse claude json: %w", err)
	}
	if err := validate(out.Scenarios); err != nil {
		return nil, err
	}
	return out.Scenarios, nil
}

func buildUserPrompt(job ScenarioJob) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Напиши %d сценари(я/ев) рекламного видео.\n", job.Count)
	fmt.Fprintf(&b, "В каждом сценарии ровно %d кадров, общая длительность ролика около %d секунд.\n",
		job.ScenesPerScenario, job.ScenesPerScenario*job.SceneDurationSec)
	fmt.Fprintf(&b, "Для каждого кадра укажи duration_sec — целое число от %d до %d секунд. "+
		"Подбирай длительность под содержание кадра: хук короче, демонстрация продукта может быть дольше; "+
		"в среднем около %d секунд на кадр.\n", MinSceneDurationSec, MaxSceneDurationSec, job.SceneDurationSec)
	fmt.Fprintf(&b, "Язык текста сценариев: %s.\n\n", job.Language)

	b.WriteString("<product>\n")
	b.WriteString(job.ProductDescription)
	b.WriteString("\n</product>\n")

	if job.TargetAudience != "" {
		b.WriteString("\n<target_audience>\n")
		b.WriteString(job.TargetAudience)
		b.WriteString("\n</target_audience>\n")
	}

	if len(job.PainPoints) > 0 {
		b.WriteString("\n<pain_points>\n")
		for _, p := range job.PainPoints {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		b.WriteString("</pain_points>\n")
		b.WriteString("Каждый сценарий строй вокруг одной из перечисленных болей.\n")
	} else {
		b.WriteString("\nБоли ЦА не заданы: определи самые сильные сам, исходя из продукта и аудитории.\n")
	}

	if len(job.Images) > 0 {
		b.WriteString("\nФотографии продукта приложены выше.\n")
	}
	b.WriteString("\nНумеруй кадры (position) с 1 по порядку.")
	return b.String()
}

func validate(scenarios []GeneratedScenario) error {
	if len(scenarios) == 0 {
		return fmt.Errorf("%w: no scenarios", ErrInvalidResult)
	}
	for i, sc := range scenarios {
		if strings.TrimSpace(sc.Title) == "" || len(sc.Scenes) == 0 {
			return fmt.Errorf("%w: scenario %d has empty title or scenes", ErrInvalidResult, i)
		}
		for j, s := range sc.Scenes {
			if strings.TrimSpace(s.Description) == "" {
				return fmt.Errorf("%w: scenario %d scene %d has empty description", ErrInvalidResult, i, j)
			}
		}
	}
	return nil
}
