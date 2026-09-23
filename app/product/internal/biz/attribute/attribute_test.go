package attribute

import (
	"context"
	"errors"
	"testing"
)

func TestDeleteTemplateInUse(t *testing.T) {
	repo := &fakeRepo{templates: map[string]*Template{"t1": {ID: "t1", Name: "饮料"}}}
	uc := NewAttributeUsecase(repo, fakeBind{used: true})
	err := uc.DeleteTemplate(context.Background(), "t1")
	if !errors.Is(err, ErrInUse) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateAttributeRejectsKind(t *testing.T) {
	repo := &fakeRepo{templates: map[string]*Template{"t1": {ID: "t1", Name: "饮料"}}}
	uc := NewAttributeUsecase(repo, fakeBind{})
	_, err := uc.CreateAttribute(context.Background(), &Attribute{TemplateID: "t1", Name: "容量", Kind: 9})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestUpdateAttributeKeepsUnsetFields(t *testing.T) {
	repo := &fakeRepo{attrs: map[string]*Attribute{
		"a1": {ID: "a1", TemplateID: "t1", Name: "颜色", Kind: KindSpec, Options: []string{"红"}, Sort: 3},
	}}
	uc := NewAttributeUsecase(repo, fakeBind{})
	name := " 色系 "
	got, err := uc.UpdateAttribute(context.Background(), Patch{ID: "a1", Name: &name, Options: []string{" 蓝 ", ""}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "色系" || got.Sort != 3 || got.Kind != KindSpec || len(got.Options) != 1 || got.Options[0] != "蓝" {
		t.Fatalf("got %+v", got)
	}
	bad := int32(9)
	if _, err := uc.UpdateAttribute(context.Background(), Patch{ID: "a1", Kind: &bad}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("kind got %v", err)
	}
	cleared, err := uc.UpdateAttribute(context.Background(), Patch{ID: "a1", ClearOptions: true})
	if err != nil || len(cleared.Options) != 0 {
		t.Fatalf("clear got %+v %v", cleared, err)
	}
}

type fakeBind struct{ used bool }

func (f fakeBind) UsesTemplate(context.Context, string) (bool, error) { return f.used, nil }

type fakeRepo struct {
	templates map[string]*Template
	attrs     map[string]*Attribute
}

func (r *fakeRepo) SaveTemplate(context.Context, string) (*Template, error) {
	return &Template{ID: "t-new", Name: "x"}, nil
}

func (r *fakeRepo) DeleteTemplate(_ context.Context, id string) error {
	delete(r.templates, id)
	return nil
}

func (r *fakeRepo) FindTemplate(_ context.Context, id string) (*Template, error) {
	item, ok := r.templates[id]
	if !ok {
		return nil, ErrNotFound
	}
	return item, nil
}

func (r *fakeRepo) ListTemplates(context.Context) ([]*Template, error) { return nil, nil }

func (r *fakeRepo) SaveAttribute(_ context.Context, attr *Attribute) (*Attribute, error) {
	attr.ID = "a1"
	return attr, nil
}

func (r *fakeRepo) FindAttribute(_ context.Context, id string) (*Attribute, error) {
	item, ok := r.attrs[id]
	if !ok {
		return nil, ErrAttributeMissing
	}
	copied := *item
	return &copied, nil
}

func (r *fakeRepo) UpdateAttribute(_ context.Context, attr *Attribute) (*Attribute, error) {
	r.attrs[attr.ID] = attr
	return attr, nil
}

func (r *fakeRepo) DeleteAttribute(context.Context, string) error { return nil }

func (r *fakeRepo) ListAttributes(context.Context, string) ([]*Attribute, error) { return nil, nil }
