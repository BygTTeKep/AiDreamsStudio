package projects

import "time"

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type CreateProjectRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateProjectResponse struct {
	Id string `json:"projectId"`
}

type GetProjectByIdRequest struct {
	Id string `json:"projectId"`
}

type GetProjectByIdReponse struct {
	Id        string    `json:"projectId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type SearchProjectRequest struct {
	Pagination Pagination `json:"pagination"`
}

type ProjectResponse struct {
	ID          string    `json:"projectId"`
	Name        string    `json:"name"`
	UserCreated int       `json:"userCreated"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SearchProjectResponse struct {
	Projects []ProjectResponse `json:"projects"`
}

func toProjectResponse(p Project) ProjectResponse {
	return ProjectResponse{ID: p.ID, Name: p.Name, CreatedAt: p.CreatedAt}
}

func toProjectResponses(ps []Project) []ProjectResponse {
	out := make([]ProjectResponse, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProjectResponse(p))
	}
	return out
}
