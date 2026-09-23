//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	notificationv1 "github.com/tmjwjx/supermarket/api/notification/v1"
	orderv1 "github.com/tmjwjx/supermarket/api/order/v1"
	paymentv1 "github.com/tmjwjx/supermarket/api/payment/v1"
	productv1 "github.com/tmjwjx/supermarket/api/product/v1"
	userv1 "github.com/tmjwjx/supermarket/api/user/v1"

	"google.golang.org/protobuf/proto"
)

const orderStatusPaid = 2

func gateway() string {
	if v := strings.TrimSpace(os.Getenv("E2E_GATEWAY")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8080"
}

type client struct {
	t     *testing.T
	http  *http.Client
	token string
}

// do 发请求并按 gateway 的 JSON 形态解析到 proto 结构 非 2xx 直接让用例失败
func (c *client) do(method, path string, body proto.Message, out proto.Message) {
	c.t.Helper()
	if code, raw := c.send(method, path, body, out); code/100 != 2 {
		c.t.Fatalf("%s %s status %d body %s", method, path, code, raw)
	}
}

func (c *client) send(method, path string, body proto.Message, out proto.Message) (int, string) {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, gateway()+path, reader)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode/100 == 2 && out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("%s %s decode %v body %s", method, path, err, raw)
		}
	}
	return res.StatusCode, string(raw)
}

// eventually 每 500ms 重试一次 超时让用例失败
func eventually(t *testing.T, within time.Duration, what string, fn func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(within)
	last := ""
	for time.Now().Before(deadline) {
		ok, detail := fn()
		if ok {
			return
		}
		last = detail
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("%s 超时 最后状态 %s", what, last)
}

// 经 gateway 走完买家下单支付全链路 需要先启动全部服务并跑过种子
func TestCheckoutThroughGateway(t *testing.T) {
	c := &client{t: t, http: &http.Client{Timeout: 10 * time.Second}}
	phone := fmt.Sprintf("139%08d", time.Now().UnixNano()%100000000)
	password := "e2e-pass-123"

	var reg userv1.RegisterResponse
	c.do(http.MethodPost, "/v1/users/register", &userv1.RegisterRequest{Phone: phone, Password: password, Nickname: "e2e"}, &reg)
	t.Logf("注册 user=%s phone=%s", reg.GetUser().GetId(), phone)

	var login userv1.LoginResponse
	c.do(http.MethodPost, "/v1/users/login", &userv1.LoginRequest{Phone: phone, Password: password}, &login)
	if login.GetAccessToken() == "" {
		t.Fatal("登录没有返回令牌")
	}
	c.token = login.GetAccessToken()
	t.Logf("登录 token 长度 %d", len(c.token))

	var list productv1.ListProductsResponse
	c.do(http.MethodGet, "/v1/products?page_size=20", nil, &list)
	if len(list.GetProducts()) == 0 {
		t.Fatal("商品列表为空 先跑 app/product/cmd/seed")
	}
	t.Logf("列表 %d 个在售商品", len(list.GetProducts()))

	var sku *productv1.Sku
	var product *productv1.Product
	for _, card := range list.GetProducts() {
		var detail productv1.GetProductResponse
		c.do(http.MethodGet, "/v1/products/"+card.GetId(), nil, &detail)
		for _, s := range detail.GetProduct().GetSkus() {
			if s.GetEnabled() && s.GetPrice() > 0 {
				sku, product = s, detail.GetProduct()
				break
			}
		}
		if sku != nil {
			break
		}
	}
	if sku == nil {
		t.Fatal("没有可买的 SKU")
	}
	t.Logf("详情 product=%s sku=%s price=%d", product.GetName(), sku.GetId(), sku.GetPrice())

	var cart orderv1.AddCartItemResponse
	c.do(http.MethodPost, "/v1/cart/items", &orderv1.AddCartItemRequest{SkuId: sku.GetId(), Quantity: 1}, &cart)
	t.Logf("加购 sku=%s", cart.GetItem().GetSkuId())

	var addr userv1.CreateAddressResponse
	c.do(http.MethodPost, "/v1/addresses", &userv1.CreateAddressRequest{
		Receiver: "张三", Phone: phone, Province: "上海市", City: "上海市", District: "浦东新区", Detail: "世纪大道 1 号",
	}, &addr)
	t.Logf("地址 id=%s", addr.GetAddress().GetId())

	var created orderv1.CreateOrderResponse
	c.do(http.MethodPost, "/v1/orders", &orderv1.CreateOrderRequest{
		Lines:     []*orderv1.OrderLine{{SkuId: sku.GetId(), Quantity: 1}},
		AddressId: addr.GetAddress().GetId(),
		RequestId: fmt.Sprintf("e2e-%d", time.Now().UnixNano()),
	}, &created)
	orderID := created.GetOrder().GetId()
	t.Logf("下单 order=%s pay_amount=%d status=%d", orderID, created.GetOrder().GetPayAmount(), created.GetOrder().GetStatus())

	var pay paymentv1.CreatePaymentResponse
	c.do(http.MethodPost, "/v1/payments", &paymentv1.CreatePaymentRequest{OrderId: orderID}, &pay)
	payID := pay.GetPayment().GetId()
	t.Logf("创建支付 payment=%s amount=%d", payID, pay.GetPayment().GetAmount())

	var sim paymentv1.SimulatePaymentResponse
	c.do(http.MethodPost, "/v1/payments/"+payID+":simulate", &paymentv1.SimulatePaymentRequest{Success: true}, &sim)
	t.Logf("模拟支付成功 status=%d", sim.GetPayment().GetStatus())

	eventually(t, 30*time.Second, "订单变为已支付", func() (bool, string) {
		var got orderv1.GetOrderResponse
		c.do(http.MethodGet, "/v1/orders/"+orderID, nil, &got)
		return got.GetOrder().GetStatus() == orderStatusPaid, fmt.Sprintf("status=%d", got.GetOrder().GetStatus())
	})
	t.Logf("订单已支付 order=%s", orderID)

	eventually(t, 60*time.Second, "收到通知", func() (bool, string) {
		var notes notificationv1.ListNotificationsResponse
		c.do(http.MethodGet, "/v1/notifications", nil, &notes)
		for _, n := range notes.GetNotifications() {
			t.Logf("通知 title=%s order=%s", n.GetTitle(), n.GetOrderId())
		}
		return len(notes.GetNotifications()) > 0, fmt.Sprintf("count=%d", len(notes.GetNotifications()))
	})
}
