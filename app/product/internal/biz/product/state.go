package product

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	v1 "github.com/tmjwjx/supermarket/api/product/v1"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/attribute"
	"github.com/tmjwjx/supermarket/app/product/internal/biz/category"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

func statusName(status int32) string {
	switch status {
	case StatusDraft:
		return "draft"
	case StatusPending:
		return "pending"
	case StatusRejected:
		return "rejected"
	case StatusApproved:
		return "approved"
	case StatusOnSale:
		return "on_sale"
	case StatusOff:
		return "off"
	default:
		return "unknown"
	}
}

func invalidAttr(name string) error {
	return kerrors.BadRequest(v1.ErrorReason_PRODUCT_INVALID_ARGUMENT.String(), "attribute "+name)
}

func (uc *ProductUsecase) checkTemplate(ctx context.Context, cat *category.Category, p *Product) error {
	if cat == nil || strings.TrimSpace(cat.TemplateID) == "" {
		return nil
	}
	if uc.templates == nil {
		return ErrInvalid
	}
	rows, err := uc.templates.List(ctx, cat.TemplateID)
	if err != nil {
		return err
	}
	specs := map[string]attribute.Attribute{}
	params := map[string]attribute.Attribute{}
	for _, row := range rows {
		if row.Kind == attribute.KindSpec {
			specs[row.Name] = row
		}
		if row.Kind == attribute.KindParam {
			params[row.Name] = row
		}
	}
	for i := range p.Skus {
		var raw map[string]string
		if err := json.Unmarshal([]byte(p.Skus[i].SpecsJSON), &raw); err != nil {
			return ErrInvalid
		}
		for name, value := range raw {
			attr, ok := specs[name]
			if !ok {
				return invalidAttr(name)
			}
			if !attr.AllowCustom && !containsOption(attr.Options, value) {
				return invalidAttr(name)
			}
		}
	}
	for i := range p.Params {
		p.Params[i].Name = strings.TrimSpace(p.Params[i].Name)
		p.Params[i].Value = strings.TrimSpace(p.Params[i].Value)
		if p.Params[i].Name == "" {
			return ErrInvalid
		}
		if _, ok := params[p.Params[i].Name]; !ok {
			return invalidAttr(p.Params[i].Name)
		}
	}
	return nil
}

func containsOption(options []string, value string) bool {
	for _, option := range options {
		if option == value {
			return true
		}
	}
	return false
}

// Submit 把草稿或已驳回送去待审核并写审核记录
func (uc *ProductUsecase) Submit(ctx context.Context, id, operator string) (*Product, error) {
	if id == "" {
		return nil, ErrInvalid
	}
	if err := uc.repo.SetStatus(ctx, id, []int32{StatusDraft, StatusRejected}, StatusPending, &Trail{Operator: operator, Action: "submit", Audit: true}); err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, id)
}

// Approve 把待审核标成已通过并写审核记录
func (uc *ProductUsecase) Approve(ctx context.Context, id, operator string) (*Product, error) {
	if id == "" {
		return nil, ErrInvalid
	}
	if err := uc.repo.SetStatus(ctx, id, []int32{StatusPending}, StatusApproved, &Trail{Operator: operator, Action: "approve", Audit: true}); err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, id)
}

// Reject 把待审核标成已驳回并写审核记录
func (uc *ProductUsecase) Reject(ctx context.Context, id, operator, reason string) (*Product, error) {
	if id == "" {
		return nil, ErrInvalid
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrInvalid
	}
	if err := uc.repo.SetStatus(ctx, id, []int32{StatusPending}, StatusRejected, &Trail{Operator: operator, Action: "reject", Reason: reason, Audit: true}); err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, id)
}

// Delete 软删除商品 买家随后看不到
func (uc *ProductUsecase) Delete(ctx context.Context, id, operator string) error {
	if id == "" {
		return ErrInvalid
	}
	return uc.repo.SoftDelete(ctx, id, &Trail{Operator: operator, Action: "delete"})
}

// Restore 把回收站商品恢复为下架
func (uc *ProductUsecase) Restore(ctx context.Context, id, operator string) (*Product, error) {
	if id == "" {
		return nil, ErrInvalid
	}
	if err := uc.repo.Restore(ctx, id, &Trail{Operator: operator, Action: "restore", After: statusName(StatusOff)}); err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, id)
}

// Purge 物理删除回收站中的商品及其规格图片和详情
func (uc *ProductUsecase) Purge(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalid
	}
	return uc.repo.Purge(ctx, id)
}

// ChangePrice 修改单个规格售价并写操作日志
func (uc *ProductUsecase) ChangePrice(ctx context.Context, productID, skuID string, price int64, operator string) (*Product, error) {
	if productID == "" || skuID == "" || price <= 0 {
		return nil, ErrInvalid
	}
	return uc.repo.ChangePrice(ctx, productID, skuID, price, &Trail{Operator: operator, Action: "change_price"})
}

// BatchPublish 逐个上架 失败的留在结果里
func (uc *ProductUsecase) BatchPublish(ctx context.Context, ids []string, operator string) (published []string, failed []Fail) {
	for _, id := range ids {
		if _, err := uc.Publish(ctx, id, operator); err != nil {
			failed = append(failed, Fail{ID: id, Reason: explain(err)})
			continue
		}
		published = append(published, id)
	}
	return published, failed
}

// BatchUnpublish 逐个下架 失败的留在结果里
func (uc *ProductUsecase) BatchUnpublish(ctx context.Context, ids []string, operator string) (unpublished []string, failed []Fail) {
	for _, id := range ids {
		if _, err := uc.Unpublish(ctx, id, operator); err != nil {
			failed = append(failed, Fail{ID: id, Reason: explain(err)})
			continue
		}
		unpublished = append(unpublished, id)
	}
	return unpublished, failed
}

func explain(err error) string {
	if err == nil {
		return ""
	}
	if ke := kerrors.FromError(err); ke != nil && ke.Message != "" {
		return ke.Message
	}
	return err.Error()
}

// AcceptCompleted 消费订单完成事件 销量仍可由同步调用先记上
func (uc *ProductUsecase) AcceptCompleted(ctx context.Context, eventID, payload string) error {
	var body struct {
		OrderID string `json:"order_id"`
		Items   []struct {
			ProductID string `json:"product_id"`
			Count     int64  `json:"count"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(payload), &body); err != nil {
		return err
	}
	if body.OrderID == "" || len(body.Items) == 0 {
		return nil
	}
	for _, item := range body.Items {
		if item.ProductID == "" || item.Count <= 0 {
			continue
		}
		mark := eventID + "/" + item.ProductID
		if err := uc.AddSales(ctx, mark, body.OrderID, item.ProductID, item.Count); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}
