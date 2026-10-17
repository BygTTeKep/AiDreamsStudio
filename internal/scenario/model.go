package scenario

import "time"

type Status string

const (
	StatusPendingReview Status = "pending_review"
	StatusApproved      Status = "approved"
	StatusRejected      Status = "rejected"
)

type Source string

const (
	SourceLLM  Source = "llm"
	SourceUser Source = "user"
)

type Scenario struct {
	ID        string
	ProjectID string
	PainPoint string
	Title     string
	Source    Source
	Status    Status
	Scenes    []Scene
	CreatedAt time.Time
}

// Допустимая длительность кадра: в таких пределах генерирует видео-модель.
const (
	MinSceneDurationSec = 3
	MaxSceneDurationSec = 10
)

type Scene struct {
	ID          string
	ScenarioID  string
	Position    int
	DurationSec int
	Description string
}
