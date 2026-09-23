package review

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/paging"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
)

var (
	ErrNotAllowed      = kerrors.Forbidden(v1.ErrorReason_REVIEW_NOT_ALLOWED.String(), "review not allowed")
	ErrExists          = kerrors.Conflict(v1.ErrorReason_REVIEW_EXISTS.String(), "review exists")
	ErrInvalid         = kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "invalid argument")
	ErrAlreadyReviewed = errors.New("order item already reviewed")
	ErrDenied          = errors.New("order item cannot be reviewed")
	ErrUnavailable     = kerrors.ServiceUnavailable(v1.ErrorReason_REVIEW_UNAVAILABLE.String(), "review unavailable")
)

// Review 是评价领域对象
type Review struct {
	ID          string
	ProductID   string
	OrderItemID string
	UserID      string
	DisplayName string
	Content     string
	Stars       int32
	Anonymous   bool
	SpecsJSON   string
}

// Profile 是评价展示名要用的会员资料
type Profile struct {
	Nickname string
	Phone    string
}

// ProfileReader 读取会员昵称和手机号
type ProfileReader interface {
	Profile(ctx context.Context, userID string) (Profile, error)
}

// ReviewRepo 是评价存取接口
type ReviewRepo interface {
	Add(ctx context.Context, review *Review) (*Review, error)
	List(ctx context.Context, productID string, size, offset int) ([]*Review, bool, error)
}

// OrderChecker 向订单确认订单项能否评价
type OrderChecker interface {
	Check(ctx context.Context, userID, orderItemID, productID string) (string, error)
	MarkReviewed(ctx context.Context, orderItemID string) error
}

// ReviewUsecase 持有评价规则
type ReviewUsecase struct {
	repo     ReviewRepo
	order    OrderChecker
	profiles ProfileReader
}

func NewReviewUsecase(repo ReviewRepo, order OrderChecker, profiles ProfileReader) *ReviewUsecase {
	return &ReviewUsecase{repo: repo, order: order, profiles: profiles}
}

// Create 写评价 业务不允许返回 403 订单超时或存储失败返回 503
func (uc *ReviewUsecase) Create(ctx context.Context, userID string, in *Review) (*Review, error) {
	if in == nil {
		return nil, ErrInvalid
	}
	userID = strings.TrimSpace(userID)
	in.OrderItemID = strings.TrimSpace(in.OrderItemID)
	in.ProductID = strings.TrimSpace(in.ProductID)
	in.Content = strings.TrimSpace(in.Content)
	if userID == "" || in.OrderItemID == "" || in.ProductID == "" {
		return nil, ErrInvalid
	}
	if in.Stars < 1 || in.Stars > 5 || len([]rune(in.Content)) > 500 {
		return nil, ErrInvalid
	}
	if uc.order == nil {
		return nil, ErrNotAllowed
	}
	specs, err := uc.order.Check(ctx, userID, in.OrderItemID, in.ProductID)
	if err != nil {
		if errors.Is(err, ErrAlreadyReviewed) {
			return nil, ErrExists
		}
		if errors.Is(err, ErrDenied) {
			return nil, ErrNotAllowed
		}
		log.Error("order check review", "user", userID, "item", in.OrderItemID, "err", err)
		return nil, ErrUnavailable
	}
	in.UserID = userID
	in.SpecsJSON = specs
	name, err := uc.displayName(ctx, userID, in.Anonymous)
	if err != nil {
		return nil, err
	}
	in.DisplayName = name
	created, err := uc.repo.Add(ctx, in)
	if err != nil {
		if errors.Is(err, ErrExists) {
			return nil, err
		}
		log.Error("save review", "item", in.OrderItemID, "err", err)
		return nil, ErrUnavailable
	}
	if err := uc.order.MarkReviewed(ctx, in.OrderItemID); err != nil {
		log.Error("mark order item reviewed", "item", in.OrderItemID, "err", err)
	}
	return created, nil
}

func (uc *ReviewUsecase) displayName(ctx context.Context, userID string, anonymous bool) (string, error) {
	if anonymous {
		return "匿名用户", nil
	}
	if uc.profiles == nil {
		return "用户", nil
	}
	prof, err := uc.profiles.Profile(ctx, userID)
	if err != nil {
		log.Error("review profile", "user", userID, "err", err)
		return "", ErrUnavailable
	}
	nickname := strings.TrimSpace(prof.Nickname)
	if nickname != "" {
		return nickname, nil
	}
	phone := strings.TrimSpace(prof.Phone)
	if len(phone) >= 4 {
		return "用户" + phone[len(phone)-4:], nil
	}
	return "用户", nil
}

// List 公开分页列出评价 匿名显示名固定为匿名用户
func (uc *ReviewUsecase) List(ctx context.Context, productID, token string, size int32) ([]*Review, string, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, "", ErrInvalid
	}
	off, err := paging.Offset(token)
	if err != nil {
		return nil, "", ErrInvalid
	}
	n := paging.Size(size)
	rows, hasMore, err := uc.repo.List(ctx, productID, n, off)
	if err != nil {
		return nil, "", err
	}
	for _, row := range rows {
		if row.Anonymous {
			row.DisplayName = "匿名用户"
		} else if row.DisplayName == "" {
			row.DisplayName = "用户"
		}
	}
	next := ""
	if hasMore {
		next = paging.Token(off + n)
	}
	return rows, next, nil
}
