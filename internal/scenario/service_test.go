package scenario

import (
	"context"
	"errors"
	"testing"

	"github.com/BygTTeKep/AiDreamsStudio/internal/generation/claude"
	"github.com/BygTTeKep/AiDreamsStudio/internal/projects"
)

const (
	testProject  = "11111111-1111-1111-1111-111111111111"
	testScenario = "22222222-2222-2222-2222-222222222222"
)

type fakeRepo struct {
	scenario   Scenario
	approveErr error
	created    CreateScenarioRequest
}

func (f *fakeRepo) GetScenariosByStatus(context.Context, int, string, Status) ([]Scenario, error) {
	return nil, nil
}
func (f *fakeRepo) ProjectOwnedBy(context.Context, int, string) (bool, error) { return true, nil }
func (f *fakeRepo) ApproveScenario(context.Context, string, string, int) error {
	if f.approveErr != nil {
		return f.approveErr
	}
	f.scenario.Status = StatusApproved
	return nil
}
func (f *fakeRepo) CreateScenario(_ context.Context, sc CreateScenarioRequest) (string, error) {
	f.created = sc
	return testScenario, nil
}
func (f *fakeRepo) GetById(context.Context, string, string, int) (Scenario, error) {
	return f.scenario, nil
}
func (f *fakeRepo) SearchScenarios(ctx context.Context, limit, offset, userId int, projectId string) ([]Scenario, error) {
	return nil, nil
}

func TestApproveScenario(t *testing.T) {
	repo := &fakeRepo{scenario: Scenario{ID: testScenario, Status: StatusPendingReview}}
	svc := NewService(repo, nil, nil)

	got, err := svc.ApproveScenario(context.Background(), testProject, testScenario, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusApproved {
		t.Fatalf("want approved, got %s", got.Status)
	}
}

func TestApproveScenarioErrors(t *testing.T) {
	cases := []struct {
		name, project, scenario string
		repoErr, want           error
	}{
		{"bad project id", "x", testScenario, nil, projects.ErrProjectNotFound},
		{"bad scenario id", testProject, "x", nil, ErrScenarioNotFound},
		{"not found", testProject, testScenario, ErrScenarioNotFound, ErrScenarioNotFound},
		{"not pending", testProject, testScenario, ErrScenarioNotPending, ErrScenarioNotPending},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(&fakeRepo{approveErr: tc.repoErr}, nil, nil)
			_, err := svc.ApproveScenario(context.Background(), tc.project, tc.scenario, 1)
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestSaveScenarioSetsServerFields(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil, nil)

	_, err := svc.SaveScenario(context.Background(), testProject, CreateScenarioRequest{
		Title:  "t",
		Scenes: []SceneRequest{{Position: 5, DurationSec: 4, Description: "b"}, {Position: 2, DurationSec: 6, Description: "a"}},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	c := repo.created
	if c.ProjectID != testProject || c.Source != SourceUser || c.Status != StatusPendingReview {
		t.Fatalf("server fields not set: %+v", c)
	}
	if c.Scenes[0].Description != "a" || c.Scenes[0].Position != 1 || c.Scenes[1].Position != 2 {
		t.Fatalf("scenes not normalized: %+v", c.Scenes)
	}
}

func TestSaveGeneratedScenariosDuration(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil, nil)

	err := svc.SaveGeneratedScenarios(context.Background(), claude.ScenarioJob{ProjectID: testProject, SceneDurationSec: 5},
		[]claude.GeneratedScenario{{Title: "t", Scenes: []claude.GeneratedScene{
			{DurationSec: 4, Description: "a"},
			{DurationSec: 0, Description: "b"},  // не указана — средняя из задачи
			{DurationSec: 30, Description: "c"}, // больше максимума
			{DurationSec: 1, Description: "d"},  // меньше минимума
		}}})
	if err != nil {
		t.Fatal(err)
	}
	want := []int{4, 5, MaxSceneDurationSec, MinSceneDurationSec}
	for i, sc := range repo.created.Scenes {
		if sc.DurationSec != want[i] {
			t.Fatalf("scene %d: want %d, got %d", i, want[i], sc.DurationSec)
		}
	}
}

func TestScenarioResponseTotalDuration(t *testing.T) {
	resp := toScenarioResponse(Scenario{Scenes: []Scene{{DurationSec: 3}, {DurationSec: 7}}})
	if resp.DurationSec != 10 || resp.Scenes[1].DurationSec != 7 {
		t.Fatalf("unexpected durations: %+v", resp)
	}
}
