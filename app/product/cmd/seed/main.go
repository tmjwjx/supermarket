package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	inventoryv1 "github.com/tmjwjx/supermarket/api/inventory/v1"
	productv1 "github.com/tmjwjx/supermarket/api/product/v1"

	"github.com/go-kratos/kratos/v3/metadata"
	mmd "github.com/go-kratos/kratos/v3/middleware/metadata"
	kgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	"google.golang.org/grpc"
)

// 种子以固定的超级管理员身份经 gRPC 走正常审核上架流程
const (
	seedAdminID   = "00000000-0000-0000-0000-000000000001"
	seedAdminRole = "super"
	seedStock     = 100
)

const (
	statusDraft    int32 = 1
	statusPending  int32 = 2
	statusRejected int32 = 3
	statusApproved int32 = 4
	statusOnSale   int32 = 5
	statusOff      int32 = 6
)

var (
	productAddr   string
	inventoryAddr string
)

func init() {
	flag.StringVar(&productAddr, "product", "127.0.0.1:9001", "product gRPC address")
	flag.StringVar(&inventoryAddr, "inventory", "127.0.0.1:9002", "inventory gRPC address")
}

type brandSeed struct {
	name    string
	initial string
	sort    int32
}

type categorySeed struct {
	parent string
	name   string
	sort   int32
}

type attrSeed struct {
	name        string
	kind        int32
	options     []string
	allowCustom bool
	sort        int32
}

type productSeed struct {
	name     string
	category string
	brand    string
	keywords string
	price    int64
}

type clients struct {
	brands          productv1.BrandServiceClient
	categories      productv1.CategoryServiceClient
	attributes      productv1.AttributeServiceClient
	products        productv1.ProductServiceClient
	recommendations productv1.RecommendationServiceClient
	stocks          inventoryv1.StockServiceClient
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	productConn, err := dial(ctx, productAddr)
	if err != nil {
		return err
	}
	defer productConn.Close()
	inventoryConn, err := dial(ctx, inventoryAddr)
	if err != nil {
		return err
	}
	defer inventoryConn.Close()
	c := clients{
		brands:          productv1.NewBrandServiceClient(productConn),
		categories:      productv1.NewCategoryServiceClient(productConn),
		attributes:      productv1.NewAttributeServiceClient(productConn),
		products:        productv1.NewProductServiceClient(productConn),
		recommendations: productv1.NewRecommendationServiceClient(productConn),
		stocks:          inventoryv1.NewStockServiceClient(inventoryConn),
	}
	adminCtx := metadata.AppendToClientContext(ctx,
		"x-md-global-admin-id", seedAdminID,
		"x-md-global-admin-role", seedAdminRole,
	)
	return seed(adminCtx, c)
}

func dial(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return kgrpc.NewClient(ctx,
		kgrpc.WithEndpoint(addr),
		kgrpc.WithTimeout(10*time.Second),
		kgrpc.WithMiddleware(mmd.Client()),
	)
}

func seed(ctx context.Context, c clients) error {
	brandID, err := seedBrands(ctx, c.brands, []brandSeed{
		{name: "鲜选", initial: "X", sort: 1},
		{name: "家清", initial: "J", sort: 2},
		{name: "田园", initial: "T", sort: 3},
	})
	if err != nil {
		return fmt.Errorf("brands: %w", err)
	}
	childID, err := seedCategories(ctx, c.categories,
		[]categorySeed{{name: "食品", sort: 1}, {name: "日用", sort: 2}, {name: "生鲜", sort: 3}},
		[]categorySeed{
			{parent: "食品", name: "乳品", sort: 1},
			{parent: "食品", name: "休闲", sort: 2},
			{parent: "日用", name: "纸品", sort: 1},
			{parent: "日用", name: "清洁", sort: 2},
			{parent: "生鲜", name: "水果", sort: 1},
			{parent: "生鲜", name: "蔬菜", sort: 2},
		},
	)
	if err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	templateID, err := seedTemplate(ctx, c.attributes, "超市通用", []attrSeed{
		{name: "规格", kind: 1, options: []string{"默认"}, allowCustom: true, sort: 1},
		{name: "产地", kind: 2, allowCustom: true, sort: 2},
	})
	if err != nil {
		return fmt.Errorf("attributes: %w", err)
	}
	if err := bindTemplate(ctx, c.categories, childID, templateID); err != nil {
		return fmt.Errorf("bind template: %w", err)
	}
	rows := []productSeed{
		{name: "鲜牛奶", category: "乳品", brand: "鲜选", keywords: "牛奶 乳品", price: 3990},
		{name: "原味酸奶", category: "乳品", brand: "鲜选", keywords: "酸奶 乳品", price: 4590},
		{name: "儿童奶酪", category: "乳品", brand: "鲜选", keywords: "奶酪 乳品", price: 1990},
		{name: "全脂奶粉", category: "乳品", brand: "鲜选", keywords: "奶粉 乳品", price: 5990},
		{name: "原味薯片", category: "休闲", brand: "鲜选", keywords: "薯片 零食", price: 890},
		{name: "苏打饼干", category: "休闲", brand: "鲜选", keywords: "饼干 零食", price: 690},
		{name: "黑巧克力", category: "休闲", brand: "鲜选", keywords: "巧克力 零食", price: 1590},
		{name: "混合坚果", category: "休闲", brand: "鲜选", keywords: "坚果 零食", price: 2990},
		{name: "抽纸", category: "纸品", brand: "家清", keywords: "抽纸 纸品", price: 1290},
		{name: "卷纸", category: "纸品", brand: "家清", keywords: "卷纸 纸品", price: 1990},
		{name: "湿巾", category: "纸品", brand: "家清", keywords: "湿巾 纸品", price: 990},
		{name: "洗衣液", category: "清洁", brand: "家清", keywords: "洗衣液 清洁", price: 3290},
		{name: "洗洁精", category: "清洁", brand: "家清", keywords: "洗洁精 清洁", price: 890},
		{name: "牙膏", category: "清洁", brand: "家清", keywords: "牙膏 清洁", price: 1590},
		{name: "苹果", category: "水果", brand: "田园", keywords: "苹果 水果", price: 799},
		{name: "香蕉", category: "水果", brand: "田园", keywords: "香蕉 水果", price: 499},
		{name: "橙子", category: "水果", brand: "田园", keywords: "橙子 水果", price: 699},
		{name: "番茄", category: "蔬菜", brand: "田园", keywords: "番茄 蔬菜", price: 599},
		{name: "黄瓜", category: "蔬菜", brand: "田园", keywords: "黄瓜 蔬菜", price: 399},
		{name: "青菜", category: "蔬菜", brand: "田园", keywords: "青菜 蔬菜", price: 299},
	}
	existing, err := productIDsByName(ctx, c.products)
	if err != nil {
		return fmt.Errorf("list products: %w", err)
	}
	products := make([]*productv1.Product, 0, len(rows))
	for i, row := range rows {
		p, err := ensureProduct(ctx, c.products, existing, row, childID[row.category], brandID[row.brand], i)
		if err != nil {
			return fmt.Errorf("product %s: %w", row.name, err)
		}
		products = append(products, p)
	}
	if err := seedRecommendations(ctx, c.recommendations, products); err != nil {
		return fmt.Errorf("recommendations: %w", err)
	}
	stocked, err := seedStocks(ctx, c.stocks, products)
	if err != nil {
		return fmt.Errorf("stocks: %w", err)
	}
	fmt.Printf("seed ok brands=%d categories=%d products=%d stocked_skus=%d\n", len(brandID), len(childID), len(products), stocked)
	return nil
}

func seedBrands(ctx context.Context, cli productv1.BrandServiceClient, rows []brandSeed) (map[string]string, error) {
	list, err := cli.AdminListBrands(ctx, &productv1.AdminListBrandsRequest{})
	if err != nil {
		return nil, err
	}
	ids := map[string]string{}
	for _, b := range list.GetBrands() {
		ids[b.GetName()] = b.GetId()
	}
	for _, row := range rows {
		if _, ok := ids[row.name]; ok {
			continue
		}
		created, err := cli.CreateBrand(ctx, &productv1.CreateBrandRequest{
			Name: row.name, Initial: row.initial, Description: row.name, Visible: true, Sort: row.sort,
		})
		if err != nil {
			return nil, err
		}
		ids[row.name] = created.GetBrand().GetId()
	}
	return ids, nil
}

// 返回二级分类名到 id 的映射 种子里二级分类名不重复
func seedCategories(ctx context.Context, cli productv1.CategoryServiceClient, roots, children []categorySeed) (map[string]string, error) {
	tree, err := cli.AdminListCategories(ctx, &productv1.AdminListCategoriesRequest{})
	if err != nil {
		return nil, err
	}
	rootID := map[string]string{}
	childID := map[string]string{}
	for _, root := range tree.GetCategories() {
		rootID[root.GetName()] = root.GetId()
		for _, child := range root.GetChildren() {
			childID[root.GetName()+"/"+child.GetName()] = child.GetId()
		}
	}
	for _, row := range roots {
		if _, ok := rootID[row.name]; ok {
			continue
		}
		created, err := cli.CreateCategory(ctx, &productv1.CreateCategoryRequest{Name: row.name, Sort: row.sort, Visible: true})
		if err != nil {
			return nil, err
		}
		rootID[row.name] = created.GetCategory().GetId()
	}
	out := map[string]string{}
	for _, row := range children {
		key := row.parent + "/" + row.name
		if id, ok := childID[key]; ok {
			out[row.name] = id
			continue
		}
		created, err := cli.CreateCategory(ctx, &productv1.CreateCategoryRequest{
			ParentId: rootID[row.parent], Name: row.name, Sort: row.sort, Visible: true,
		})
		if err != nil {
			return nil, err
		}
		out[row.name] = created.GetCategory().GetId()
	}
	return out, nil
}

func seedTemplate(ctx context.Context, cli productv1.AttributeServiceClient, name string, attrs []attrSeed) (string, error) {
	list, err := cli.ListAttributeTemplates(ctx, &productv1.ListAttributeTemplatesRequest{})
	if err != nil {
		return "", err
	}
	var id string
	for _, tpl := range list.GetTemplates() {
		if tpl.GetName() == name {
			id = tpl.GetId()
		}
	}
	if id == "" {
		created, err := cli.CreateAttributeTemplate(ctx, &productv1.CreateAttributeTemplateRequest{Name: name})
		if err != nil {
			return "", err
		}
		id = created.GetTemplate().GetId()
	}
	have, err := cli.ListAttributes(ctx, &productv1.ListAttributesRequest{TemplateId: id})
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	for _, attr := range have.GetAttributes() {
		seen[attr.GetName()] = true
	}
	for _, attr := range attrs {
		if seen[attr.name] {
			continue
		}
		if _, err := cli.CreateAttribute(ctx, &productv1.CreateAttributeRequest{
			TemplateId: id, Name: attr.name, Kind: attr.kind, Options: attr.options,
			AllowCustom: attr.allowCustom, Sort: attr.sort,
		}); err != nil {
			return "", err
		}
	}
	return id, nil
}

func bindTemplate(ctx context.Context, cli productv1.CategoryServiceClient, childID map[string]string, templateID string) error {
	tree, err := cli.AdminListCategories(ctx, &productv1.AdminListCategoriesRequest{})
	if err != nil {
		return err
	}
	bound := map[string]string{}
	for _, root := range tree.GetCategories() {
		for _, child := range root.GetChildren() {
			bound[child.GetId()] = child.GetAttributeTemplateId()
		}
	}
	for _, id := range childID {
		if bound[id] == templateID {
			continue
		}
		if _, err := cli.UpdateCategory(ctx, &productv1.UpdateCategoryRequest{Id: id, AttributeTemplateId: &templateID}); err != nil {
			return err
		}
	}
	return nil
}

func productIDsByName(ctx context.Context, cli productv1.ProductServiceClient) (map[string]string, error) {
	out := map[string]string{}
	token := ""
	for {
		page, err := cli.AdminListProducts(ctx, &productv1.AdminListProductsRequest{PageSize: 50, PageToken: token})
		if err != nil {
			return nil, err
		}
		for _, card := range page.GetProducts() {
			out[card.GetName()] = card.GetId()
		}
		token = page.GetNextPageToken()
		if token == "" {
			return out, nil
		}
	}
}

// 按当前状态补齐 提交 通过 上架 已在售直接返回
func ensureProduct(ctx context.Context, cli productv1.ProductServiceClient, existing map[string]string, row productSeed, categoryID, brandID string, index int) (*productv1.Product, error) {
	var p *productv1.Product
	if id, ok := existing[row.name]; ok {
		got, err := cli.AdminGetProduct(ctx, &productv1.AdminGetProductRequest{Id: id})
		if err != nil {
			return nil, err
		}
		p = got.GetProduct()
	} else {
		created, err := cli.CreateProduct(ctx, &productv1.CreateProductRequest{
			CategoryId: categoryID,
			BrandId:    brandID,
			Name:       row.name,
			Subtitle:   row.name,
			Keywords:   row.keywords,
			Unit:       "件",
			WeightGram: 500,
			Images:     []string{fmt.Sprintf("https://img.local/product/%02d.jpg", index+1)},
			DetailHtml: "<p>" + row.name + "</p>",
			Skus: []*productv1.SkuInput{{
				SpecsJson: `{"规格":"默认"}`, Price: row.price, MarketPrice: row.price + 100, Enabled: true,
			}},
			Params: []*productv1.ProductParam{{Name: "产地", Value: "本地"}},
		})
		if err != nil {
			return nil, err
		}
		p = created.GetProduct()
	}
	id := p.GetId()
	for step := 0; p.GetStatus() != statusOnSale; step++ {
		if step >= 3 {
			return nil, fmt.Errorf("stuck at status %d", p.GetStatus())
		}
		var err error
		switch p.GetStatus() {
		case statusDraft, statusRejected:
			var out *productv1.SubmitProductResponse
			out, err = cli.SubmitProduct(ctx, &productv1.SubmitProductRequest{Id: id})
			p = out.GetProduct()
		case statusPending:
			var out *productv1.ApproveProductResponse
			out, err = cli.ApproveProduct(ctx, &productv1.ApproveProductRequest{Id: id})
			p = out.GetProduct()
		case statusApproved, statusOff:
			var out *productv1.PublishProductResponse
			out, err = cli.PublishProduct(ctx, &productv1.PublishProductRequest{Id: id})
			p = out.GetProduct()
		default:
			return nil, fmt.Errorf("unexpected status %d", p.GetStatus())
		}
		if err != nil {
			return nil, err
		}
	}
	got, err := cli.AdminGetProduct(ctx, &productv1.AdminGetProductRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return got.GetProduct(), nil
}

// 每个位置放两个商品 同位置同商品已存在就跳过
func seedRecommendations(ctx context.Context, cli productv1.RecommendationServiceClient, products []*productv1.Product) error {
	have, err := cli.AdminListRecommendations(ctx, &productv1.AdminListRecommendationsRequest{})
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, rec := range have.GetRecommendations() {
		seen[rec.GetSlot()+"/"+rec.GetProductId()] = true
	}
	for i, slot := range []string{"new", "hot", "topic"} {
		for j := 0; j < 2; j++ {
			id := products[i*2+j].GetId()
			if seen[slot+"/"+id] {
				continue
			}
			if _, err := cli.CreateRecommendation(ctx, &productv1.CreateRecommendationRequest{
				Slot: slot, ProductId: id, Sort: int32(j + 1),
			}); err != nil {
				return fmt.Errorf("%s: %w", slot, err)
			}
		}
	}
	return nil
}

// 只给还没有库存记录或实际数量为 0 的启用 SKU 入库 重复执行不会叠加
func seedStocks(ctx context.Context, cli inventoryv1.StockServiceClient, products []*productv1.Product) (int, error) {
	var ids []string
	for _, p := range products {
		for _, sku := range p.GetSkus() {
			if sku.GetEnabled() && sku.GetId() != "" {
				ids = append(ids, sku.GetId())
			}
		}
	}
	have, err := cli.AdminGetStocks(ctx, &inventoryv1.AdminGetStocksRequest{SkuIds: ids})
	if err != nil {
		return 0, err
	}
	onHand := map[string]int64{}
	found := map[string]bool{}
	for _, row := range have.GetStocks() {
		onHand[row.GetSkuId()] = row.GetOnHand()
		found[row.GetSkuId()] = true
	}
	stocked := 0
	for _, id := range ids {
		if !found[id] {
			if _, err := cli.CreateStock(ctx, &inventoryv1.CreateStockRequest{SkuId: id}); err != nil {
				return 0, err
			}
		}
		if onHand[id] > 0 {
			continue
		}
		if _, err := cli.AdjustStock(ctx, &inventoryv1.AdjustStockRequest{SkuId: id, Delta: seedStock, Reason: "seed"}); err != nil {
			return 0, err
		}
		stocked++
	}
	return stocked, nil
}
