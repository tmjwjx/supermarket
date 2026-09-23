package secretcheck

import (
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		name string
		keys []Key
		want string
	}{
		{"both set", []Key{{"AUTH_JWT_SECRET", "a"}, {"ADMIN_JWT_SECRET", "b"}}, ""},
		{"placeholder only warns", []Key{{"AUTH_JWT_SECRET", "change-me-a"}, {"ADMIN_JWT_SECRET", "change-me-b"}}, ""},
		{"single key", []Key{{"ADMIN_JWT_SECRET", "b"}}, ""},
		{"buyer empty", []Key{{"AUTH_JWT_SECRET", " "}, {"ADMIN_JWT_SECRET", "b"}}, "AUTH_JWT_SECRET is empty"},
		{"admin empty", []Key{{"AUTH_JWT_SECRET", "a"}, {"ADMIN_JWT_SECRET", ""}}, "ADMIN_JWT_SECRET is empty"},
		{"same", []Key{{"AUTH_JWT_SECRET", "x"}, {"ADMIN_JWT_SECRET", "x"}}, "ADMIN_JWT_SECRET must differ from AUTH_JWT_SECRET"},
	}
	for _, tc := range cases {
		err := Check(tc.keys...)
		if tc.want == "" {
			if err != nil {
				t.Fatalf("%s: got %v", tc.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: got %v want %s", tc.name, err, tc.want)
		}
	}
}
