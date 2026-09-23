package recommendation

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateRejectsEndBeforeStart(t *testing.T) {
	uc := NewRecommendationUsecase(&fakeRepo{})
	_, err := uc.Create(context.Background(), &Recommendation{Slot: "hot", ProductID: "p1", StartAt: 200, EndAt: 100})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestPublicListFiltersByNowAndOnSale(t *testing.T) {
	repo := &fakeRepo{}
	uc := NewRecommendationUsecase(repo)
	uc.now = func() time.Time { return time.Unix(1000, 0) }
	if _, err := uc.List(context.Background(), "hot"); err != nil {
		t.Fatal(err)
	}
	if repo.last.ActiveAt != 1000 || repo.last.Slot != "hot" {
		t.Fatalf("filter %+v", repo.last)
	}
	if _, err := uc.AdminList(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if repo.last.ActiveAt != 0 {
		t.Fatalf("admin filter %+v", repo.last)
	}
}

func TestUpdateKeepsUnsetFields(t *testing.T) {
	repo := &fakeRepo{rows: map[string]*Recommendation{"r1": {ID: "r1", Slot: "hot", ProductID: "p1", Sort: 2, StartAt: 10}}}
	uc := NewRecommendationUsecase(repo)
	end := int64(20)
	got, err := uc.Update(context.Background(), Patch{ID: "r1", EndAt: &end})
	if err != nil {
		t.Fatal(err)
	}
	if got.Slot != "hot" || got.Sort != 2 || got.StartAt != 10 || got.EndAt != 20 {
		t.Fatalf("got %+v", got)
	}
	early := int64(5)
	if _, err := uc.Update(context.Background(), Patch{ID: "r1", EndAt: &early}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

type fakeRepo struct {
	rows map[string]*Recommendation
	last ListFilter
}

func (r *fakeRepo) Save(_ context.Context, in *Recommendation) (*Recommendation, error) {
	in.ID = "new"
	return in, nil
}

func (r *fakeRepo) Find(context.Context, string, string) (*Recommendation, error) { return nil, nil }

func (r *fakeRepo) Get(_ context.Context, id string) (*Recommendation, error) {
	row, ok := r.rows[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *row
	return &cp, nil
}

func (r *fakeRepo) Update(_ context.Context, in *Recommendation) (*Recommendation, error) {
	r.rows[in.ID] = in
	return in, nil
}

func (r *fakeRepo) Delete(context.Context, string) error { return nil }

func (r *fakeRepo) List(_ context.Context, f ListFilter) ([]*Recommendation, error) {
	r.last = f
	return nil, nil
}
