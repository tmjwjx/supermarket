package server

import "testing"

func TestAllowAdmin(t *testing.T) {
	cases := []struct {
		path string
		role string
		ok   bool
	}{
		{"/v1/admin/orders/1:ship", "order", true},
		{"/v1/admin/payments/1", "order", true},
		{"/v1/admin/orders", "product", false},
		{"/v1/admin/users", "super", true},
		{"/v1/admin/users", "order", false},
		{"/v1/admin/products", "product", true},
		{"/v1/admin/me", "product", true},
		{"/v1/admin/me", "super", true},
		{"/v1/admin/me", "order", true},
		{"/v1/admin/me", "guest", false},
		{"/v1/admin/products", "order", false},
	}
	for _, tc := range cases {
		if got := allowAdmin(tc.path, tc.role); got != tc.ok {
			t.Errorf("allowAdmin(%q, %q) = %v", tc.path, tc.role, got)
		}
	}
}
