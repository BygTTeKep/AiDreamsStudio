package scenario

import "time"

type Pagination struct {
	// required у int не пропускает 0, поэтому для offset только min
	Limit  int `json:"limit" binding:"required,min=1,max=100"`
	Offset int `json:"offset" binding:"min=0"`
}

type GetApprovedScenarioRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
}

type GetApprovedSceneResponse struct {
	ID          string `json:"id"`
	ScenarioID  string `json:"scenario_id"`
	Position    int    `json:"position"`
	DurationSec int    `json:"duration_sec"`
	Description string `json:"description"`
}

type CreateScenarioRequest struct {
	PainPoint string         `json:"pain_point"`
	Title     string         `json:"title" binding:"required"`
	Scenes    []SceneRequest `json:"scenes" binding:"required,min=1,dive"`

	// заполняются сервером, из тела запроса не читаются
	ProjectID string `json:"-"`
	Source    Source `json:"-"`
	Status    Status `json:"-"`
}
type CreateScenarioResponse struct {
	Id string `json:"id"`
}

type SceneRequest struct {
	Position    int    `json:"position"`
	DurationSec int    `json:"duration_sec" binding:"required,min=3,max=10"`
	Description string `json:"description" binding:"required"`
}

type ScenarioResponse struct {
	ID        string                     `json:"id"`
	ProjectID string                     `json:"project_id"`
	PainPoint string                     `json:"pain_point"`
	Title     string                     `json:"title"`
	Source    Source                     `json:"source"`
	Status    Status                     `json:"status"`
	Scenes    []GetApprovedSceneResponse `json:"scenes"`
	// суммарная длительность всех кадров
	DurationSec int       `json:"duration_sec"`
	CreatedAt   time.Time `json:"created_at"`
}

type ScenarioSearchRequest struct {
	Pagination Pagination `json:"pagination" binding:"required"`
}

func toScenarioResponse(sc Scenario) ScenarioResponse {
	scenes := make([]GetApprovedSceneResponse, len(sc.Scenes))
	total := 0
	for i, v := range sc.Scenes {
		scenes[i] = GetApprovedSceneResponse{
			ID: v.ID, ScenarioID: v.ScenarioID, Position: v.Position,
			DurationSec: v.DurationSec, Description: v.Description,
		}
		total += v.DurationSec
	}
	return ScenarioResponse{
		ID: sc.ID, ProjectID: sc.ProjectID, PainPoint: sc.PainPoint, Title: sc.Title,
		Source: sc.Source, Status: sc.Status, Scenes: scenes, DurationSec: total, CreatedAt: sc.CreatedAt,
	}
}

func toScenariosResponse(sc []Scenario) []ScenarioResponse {
	scenarios := make([]ScenarioResponse, len(sc))
	for i, v := range sc {
		scenarios[i] = toScenarioResponse(v)
	}
	return scenarios
}

type GenerateScenarioRequest struct {
	ProductDescription string   `json:"product_description" binding:"required"`
	TargetAudience     string   `json:"target_audience"`
	PainPoints         []string `json:"pain_points"`
	Count              int      `json:"count" binding:"omitempty,min=1,max=10"`
	ScenesPerScenario  int      `json:"scenes_per_scenario" binding:"omitempty,min=1,max=12"`
	// средняя длительность кадра: Claude может делать отдельные кадры короче или длиннее (3–10 с),
	// но общий хронометраж держит около scenes_per_scenario * scene_duration_sec
	SceneDurationSec int    `json:"scene_duration_sec" binding:"omitempty,min=3,max=10"`
	Language         string `json:"language"`
}

type GenerateScenarioResponse struct {
	Status string `json:"status"`
}
