// تست قرارداد JSON درخت مکان‌ها — فیلدهای struct جاسازی‌شده هم باید
// camelCase سریال شوند؛ وگرنه درخت فرانت‌اند نام/kind را گم می‌کند.
package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLocationNodeJSONKeys(t *testing.T) {
	parent := int64(1)
	node := &Node{
		Location: Location{ID: 1, ParentID: nil, Kind: KindBuilding, Name: "ساختمان مرکزی"},
		Children: []*Node{{Location: Location{ID: 2, ParentID: &parent, Kind: KindFloor, Name: "طبقه اول"}}},
	}
	b, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, key := range []string{`"id":`, `"parentId":`, `"kind":`, `"name":`, `"children":`} {
		if !strings.Contains(s, key) {
			t.Fatalf("missing JSON key %s in %s", key, s)
		}
	}
	if strings.Contains(s, `"ParentID"`) || strings.Contains(s, `"Location":`) {
		t.Fatalf("legacy Go field name leaked into JSON: %s", s)
	}
}
