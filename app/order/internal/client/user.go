package client

import (
	"context"

	userv1 "github.com/tmjwjx/supermarket/api/user/v1"
	"github.com/tmjwjx/supermarket/app/order/internal/biz/order"
	"github.com/tmjwjx/supermarket/app/order/internal/conf"

	kerrors "github.com/go-kratos/kratos/v3/errors"
)

type userAPI struct {
	cli userv1.AddressServiceClient
}

func NewUserAPI(c *conf.Client) (*userAPI, func(), error) {
	conn, err := dial(c.User.Addr, "127.0.0.1:9000", c.User.Timeout())
	if err != nil {
		return nil, nil, err
	}
	return &userAPI{cli: userv1.NewAddressServiceClient(conn)}, func() { _ = conn.Close() }, nil
}

func NewAddresses(api *userAPI) order.Addresses { return userAddress{api} }

type userAddress struct{ api *userAPI }

func (u userAddress) Get(ctx context.Context, userID, addressID string) (*order.AddressSnap, error) {
	resp, err := u.api.cli.GetAddress(ctx, &userv1.GetAddressRequest{Id: addressID, UserId: userID})
	if err != nil {
		if kerrors.IsNotFound(err) || kerrors.IsBadRequest(err) {
			return nil, order.ErrOrderAddressNotFound
		}
		return nil, order.ErrUpstream
	}
	addr := resp.GetAddress()
	if addr == nil {
		return nil, order.ErrOrderAddressNotFound
	}
	return &order.AddressSnap{
		Receiver: addr.GetReceiver(), Phone: addr.GetPhone(), Province: addr.GetProvince(),
		City: addr.GetCity(), District: addr.GetDistrict(), Detail: addr.GetDetail(),
	}, nil
}
