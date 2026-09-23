package brand

import (
	"context"
	"errors"
	"testing"
)

func TestCreateBrandDuplicateName(t *testing.T) {
	repo := newFakeBrandRepo()
	uc := NewBrandUsecase(repo)
	ctx := context.Background()
	if _, err := uc.Create(ctx, &Brand{Name: "伊利", Initial: "Y", Visible: true}); err != nil {
		t.Fatal(err)
	}
	_, err := uc.Create(ctx, &Brand{Name: "伊利", Initial: "Y", Visible: true})
	if !errors.Is(err, ErrNameExists) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateBrandKeepsFieldsNotInPatch(t *testing.T) {
	repo := newFakeBrandRepo()
	uc := NewBrandUsecase(repo)
	ctx := context.Background()
	created, err := uc.Create(ctx, &Brand{Name: "伊利", Initial: "Y", Visible: true, Sort: 3})
	if err != nil {
		t.Fatal(err)
	}
	name := "蒙牛"
	got, err := uc.Update(ctx, Patch{ID: created.ID, Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "蒙牛" || !got.Visible || got.Sort != 3 || got.Initial != "Y" {
		t.Fatalf("got %+v", got)
	}
}

type fakeBrandRepo struct {
	items []*Brand
}

func newFakeBrandRepo() *fakeBrandRepo {
	return &fakeBrandRepo{}
}

func (r *fakeBrandRepo) Save(_ context.Context, b *Brand) (*Brand, error) {
	for _, item := range r.items {
		if item.Name == b.Name {
			return nil, ErrNameExists
		}
	}
	cp := *b
	cp.ID = "b1"
	if len(r.items) > 0 {
		cp.ID = "b2"
	}
	r.items = append(r.items, &cp)
	return &cp, nil
}

func (r *fakeBrandRepo) Update(_ context.Context, b *Brand) (*Brand, error) {
	for i, item := range r.items {
		if item.ID == b.ID {
			cp := *b
			r.items[i] = &cp
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (r *fakeBrandRepo) Delete(context.Context, string) error { return nil }

func (r *fakeBrandRepo) Find(_ context.Context, id string) (*Brand, error) {
	for _, item := range r.items {
		if item.ID == id {
			cp := *item
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (r *fakeBrandRepo) FindByName(_ context.Context, name string) (*Brand, error) {
	for _, item := range r.items {
		if item.Name == name {
			cp := *item
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (r *fakeBrandRepo) List(_ context.Context, visibleOnly bool) ([]*Brand, error) {
	var out []*Brand
	for _, item := range r.items {
		if visibleOnly && !item.Visible {
			continue
		}
		cp := *item
		out = append(out, &cp)
	}
	return out, nil
}

func (r *fakeBrandRepo) Used(context.Context, string) (bool, error) { return false, nil }
