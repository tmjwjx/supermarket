package user

import (
	"context"

	v1 "github.com/tmjwjx/supermarket/api/user/v1"
	bizuser "github.com/tmjwjx/supermarket/app/user/internal/biz/user"
)

type AddressService struct {
	v1.UnimplementedAddressServiceServer
	uc *bizuser.AddressUsecase
}

func NewAddressService(uc *bizuser.AddressUsecase) *AddressService {
	return &AddressService{uc: uc}
}

func (s *AddressService) CreateAddress(ctx context.Context, req *v1.CreateAddressRequest) (*v1.CreateAddressResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.uc.Create(ctx, &bizuser.Address{
		UserID: uid, Receiver: req.GetReceiver(), Phone: req.GetPhone(),
		Province: req.GetProvince(), City: req.GetCity(), District: req.GetDistrict(), Detail: req.GetDetail(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateAddressResponse{Address: toAddress(created)}, nil
}

func (s *AddressService) ListAddresses(ctx context.Context, _ *v1.ListAddressesRequest) (*v1.ListAddressesResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.uc.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := &v1.ListAddressesResponse{}
	for _, row := range rows {
		out.Addresses = append(out.Addresses, toAddress(row))
	}
	return out, nil
}

func (s *AddressService) DeleteAddress(ctx context.Context, req *v1.DeleteAddressRequest) (*v1.DeleteAddressResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseUserID(req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.uc.Delete(ctx, uid, id); err != nil {
		return nil, err
	}
	return &v1.DeleteAddressResponse{}, nil
}

func (s *AddressService) UpdateAddress(ctx context.Context, req *v1.UpdateAddressRequest) (*v1.UpdateAddressResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseUserID(req.GetId())
	if err != nil {
		return nil, err
	}
	updated, err := s.uc.Update(ctx, &bizuser.Address{
		ID: id, UserID: uid, Receiver: req.GetReceiver(), Phone: req.GetPhone(),
		Province: req.GetProvince(), City: req.GetCity(), District: req.GetDistrict(), Detail: req.GetDetail(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAddressResponse{Address: toAddress(updated)}, nil
}

func (s *AddressService) SetDefaultAddress(ctx context.Context, req *v1.SetDefaultAddressRequest) (*v1.SetDefaultAddressResponse, error) {
	uid, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseUserID(req.GetId())
	if err != nil {
		return nil, err
	}
	updated, err := s.uc.SetDefault(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	return &v1.SetDefaultAddressResponse{Address: toAddress(updated)}, nil
}

func (s *AddressService) GetAddress(ctx context.Context, req *v1.GetAddressRequest) (*v1.GetAddressResponse, error) {
	uid, err := parseUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}
	id, err := parseUserID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.uc.Get(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetAddressResponse{Address: toAddress(row)}, nil
}

func toAddress(in *bizuser.Address) *v1.Address {
	if in == nil {
		return nil
	}
	return &v1.Address{
		Id: in.ID.String(), Receiver: in.Receiver, Phone: in.Phone, Province: in.Province,
		City: in.City, District: in.District, Detail: in.Detail, IsDefault: in.IsDefault,
	}
}
