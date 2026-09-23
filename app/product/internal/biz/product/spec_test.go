package product

import (
	"reflect"
	"testing"
)

func TestSpecOptionsSkipsDisabledAndDedupes(t *testing.T) {
	got := SpecOptions([]Sku{
		{SpecsJSON: `{"容量":"500ml","口味":"原味"}`, Enabled: true},
		{SpecsJSON: `{"容量":"1L","口味":"原味"}`, Enabled: true},
		{SpecsJSON: `{"容量":"2L","口味":"草莓"}`, Enabled: false},
	})
	want := []SpecOption{
		{Name: "口味", Values: []string{"原味"}},
		{Name: "容量", Values: []string{"500ml", "1L"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}
