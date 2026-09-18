package data

import (
	"context"
	"errors"
	"time"

	"github.com/tmjwjx/supermarket/app/user/internal/biz"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TodoPO is the persistent shape of a todo.
type TodoPO struct {
	ID        string    `gorm:"type:char(36);primaryKey"`
	Title     string    `gorm:"size:128"`
	Content   string    `gorm:"size:1024"`
	Completed bool
	Status    int32     `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName pins the table name instead of the gorm default pluralization.
func (TodoPO) TableName() string { return "todos" }

// newTodoPO builds a PO from a DO for writes.
func newTodoPO(t *biz.Todo) *TodoPO {
	return &TodoPO{
		ID:        t.ID.String(),
		Title:     t.Title,
		Content:   t.Content,
		Completed: t.Completed,
		Status:    int32(t.Status),
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// toBiz converts a persisted todo into its domain representation.
func (p *TodoPO) toBiz() *biz.Todo {
	if p == nil {
		return nil
	}
	id, _ := uuid.Parse(p.ID)
	return &biz.Todo{
		ID:        id,
		Title:     p.Title,
		Content:   p.Content,
		Completed: p.Completed,
		Status:    biz.TodoStatus(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

type todoRepo struct {
	data *Data
}

// NewTodoRepo creates a new TodoRepo instance.
func NewTodoRepo(data *Data) biz.TodoRepo {
	return &todoRepo{data: data}
}

// active narrows every read/write to non-deleted rows; soft-deleted records
// stay on disk for auditability but are invisible to the domain.
func (r *todoRepo) active() *gorm.DB {
	return r.data.db.Where("status = ?", int32(biz.TodoStatusActive))
}

func (r *todoRepo) FindByID(ctx context.Context, id uuid.UUID) (*biz.Todo, error) {
	var po TodoPO
	if err := r.active().First(&po, "id = ?", id.String()).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrTodoNotFound
		}
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *todoRepo) ListTodos(ctx context.Context, opts ...biz.ListOption) ([]*biz.Todo, error) {
	options := biz.ListOptions{Limit: 20}
	for _, opt := range opts {
		opt(&options)
	}
	if options.Offset < 0 || options.Limit <= 0 {
		return nil, biz.ErrTodoInvalidArgument
	}
	// TODO(placeholder): the AIP filter/order translation is not ported from
	// ent yet — listing ignores them and is always id-ordered. Wire it when a
	// real domain needs it.
	var pos []*TodoPO
	if err := r.active().
		Order("id").
		Offset(options.Offset).
		Limit(options.Limit).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	todos := make([]*biz.Todo, 0, len(pos))
	for _, po := range pos {
		todos = append(todos, po.toBiz())
	}
	return todos, nil
}

func (r *todoRepo) CreateTodo(ctx context.Context, t *biz.Todo) (*biz.Todo, error) {
	// UUIDv7 ids are time-ordered, which keeps a plain `ORDER BY id` stable
	// across pages.
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	t.ID = id
	t.Status = biz.TodoStatusActive
	po := newTodoPO(t)
	if err := r.data.db.Create(po).Error; err != nil {
		return nil, err
	}
	return po.toBiz(), nil
}

func (r *todoRepo) UpdateTodo(ctx context.Context, t *biz.Todo) (*biz.Todo, error) {
	// status is not updatable here: it moves only through DeleteTodo.
	res := r.active().
		Where("id = ?", t.ID.String()).
		Updates(map[string]any{
			"title":     t.Title,
			"content":   t.Content,
			"completed": t.Completed,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, biz.ErrTodoNotFound
	}
	return r.FindByID(ctx, t.ID)
}

// DeleteTodo soft-deletes a todo by flipping its status to deleted.
func (r *todoRepo) DeleteTodo(ctx context.Context, id uuid.UUID) error {
	res := r.active().
		Where("id = ?", id.String()).
		Update("status", int32(biz.TodoStatusDeleted))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return biz.ErrTodoNotFound
	}
	return nil
}
