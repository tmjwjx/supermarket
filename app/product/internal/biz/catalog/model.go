package catalog

// 商品状态与库里的数字一致
const (
	StatusDraft    int32 = 1
	StatusPending  int32 = 2
	StatusRejected int32 = 3
	StatusApproved int32 = 4
	StatusOnSale   int32 = 5
	StatusOff      int32 = 6
)

// Card 是列表和收藏里用的商品卡片
type Card struct {
	ID          string
	Name        string
	MainImage   string
	MinPrice    int64
	MarketPrice int64
	Sales       int64
	Status      int32
}
