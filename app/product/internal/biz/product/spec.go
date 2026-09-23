package product

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// SpecOption 是某个规格在启用 SKU 里出现过的全部取值 详情页据此画规格选择器
type SpecOption struct {
	Name   string
	Values []string
}

// SpecOptions 按规格名排序 取值保持 SKU 顺序里第一次出现的先后 只看启用的 SKU
func SpecOptions(skus []Sku) []SpecOption {
	values := map[string][]string{}
	seen := map[string]map[string]bool{}
	for _, sku := range skus {
		if !sku.Enabled {
			continue
		}
		var raw map[string]string
		if err := json.Unmarshal([]byte(sku.SpecsJSON), &raw); err != nil {
			continue
		}
		for name, value := range raw {
			if seen[name] == nil {
				seen[name] = map[string]bool{}
			}
			if seen[name][value] {
				continue
			}
			seen[name][value] = true
			values[name] = append(values[name], value)
		}
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]SpecOption, 0, len(names))
	for _, name := range names {
		out = append(out, SpecOption{Name: name, Values: values[name]})
	}
	return out
}

// CanonicalSpecs 把规格 JSON 按名称排序后取 SHA-256
func CanonicalSpecs(specsJSON string) (canonical, hash string, err error) {
	specsJSON = strings.TrimSpace(specsJSON)
	if specsJSON == "" {
		return "", "", ErrInvalid
	}
	var raw map[string]string
	if err := json.Unmarshal([]byte(specsJSON), &raw); err != nil || raw == nil {
		return "", "", ErrInvalid
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		if strings.TrimSpace(k) == "" {
			return "", "", ErrInvalid
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(raw[k])
	}
	sum := sha256.Sum256([]byte(b.String()))
	body, err := json.Marshal(raw)
	if err != nil {
		return "", "", ErrInvalid
	}
	return string(body), hex.EncodeToString(sum[:]), nil
}
