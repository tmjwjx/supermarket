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
		{"both set", []Key{{"jwt_secret", "a"}, {"admin_jwt_secret", "b"}}, ""},
		{"placeholder only warns", []Key{{"jwt_secret", "change-me-a"}, {"admin_jwt_secret", "change-me-b"}}, ""},
		{"single key", []Key{{"admin_jwt_secret", "b"}}, ""},
		{"buyer empty", []Key{{"jwt_secret", " "}, {"admin_jwt_secret", "b"}}, "jwt_secret is empty"},
		{"admin empty", []Key{{"jwt_secret", "a"}, {"admin_jwt_secret", ""}}, "admin_jwt_secret is empty"},
		{"same", []Key{{"jwt_secret", "x"}, {"admin_jwt_secret", "x"}}, "admin_jwt_secret must differ from jwt_secret"},
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
