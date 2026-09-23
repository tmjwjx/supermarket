package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Address struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Receiver  string
	Phone     string
	Province  string
	City      string
	District  string
	Detail    string
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AddressRepo interface {
	Create(ctx context.Context, address *Address) (*Address, error)
	List(ctx context.Context, userID uuid.UUID) ([]*Address, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	Get(ctx context.Context, userID, id uuid.UUID) (*Address, error)
	Count(ctx context.Context, userID uuid.UUID) (int64, error)
	ClearDefault(ctx context.Context, userID uuid.UUID) error
	Update(ctx context.Context, address *Address) (*Address, error)
	SetDefault(ctx context.Context, userID, id uuid.UUID) (*Address, error)
}

type AddressUsecase struct {
	repo AddressRepo
}

func NewAddressUsecase(repo AddressRepo) *AddressUsecase {
	return &AddressUsecase{repo: repo}
}

func (uc *AddressUsecase) Create(ctx context.Context, address *Address) (*Address, error) {
	if address.Receiver == "" || address.Detail == "" {
		return nil, ErrUserInvalidArgument
	}
	if err := ValidatePhone(address.Phone); err != nil {
		return nil, err
	}
	n, err := uc.repo.Count(ctx, address.UserID)
	if err != nil {
		return nil, err
	}
	if n >= 20 {
		return nil, ErrUserInvalidArgument
	}
	if n == 0 {
		address.IsDefault = true
	}
	if address.IsDefault {
		if err := uc.repo.ClearDefault(ctx, address.UserID); err != nil {
			return nil, err
		}
	}
	return uc.repo.Create(ctx, address)
}

func (uc *AddressUsecase) List(ctx context.Context, userID uuid.UUID) ([]*Address, error) {
	return uc.repo.List(ctx, userID)
}

func (uc *AddressUsecase) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return uc.repo.Delete(ctx, userID, id)
}

func (uc *AddressUsecase) Get(ctx context.Context, userID, id uuid.UUID) (*Address, error) {
	return uc.repo.Get(ctx, userID, id)
}

func (uc *AddressUsecase) Update(ctx context.Context, address *Address) (*Address, error) {
	if address == nil || address.Receiver == "" || address.Detail == "" {
		return nil, ErrUserInvalidArgument
	}
	if err := ValidatePhone(address.Phone); err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, address)
}

func (uc *AddressUsecase) SetDefault(ctx context.Context, userID, id uuid.UUID) (*Address, error) {
	if userID == uuid.Nil || id == uuid.Nil {
		return nil, ErrUserInvalidArgument
	}
	return uc.repo.SetDefault(ctx, userID, id)
}
