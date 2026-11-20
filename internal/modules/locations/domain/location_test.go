package domain_test

import (
	"testing"

	"fanavaryar/internal/modules/locations/domain"
)

func TestCanNest(t *testing.T) {
	cases := []struct {
		parent domain.Kind
		child  domain.Kind
		want   bool
	}{
		{domain.KindBuilding, domain.KindFloor, true},
		{domain.KindFloor, domain.KindRoom, true},
		{domain.KindRoom, domain.KindRack, true},
		// قواعد نامعتبر که باید رد شوند:
		{domain.KindBuilding, domain.KindRoom, false},
		{domain.KindBuilding, domain.KindRack, false},
		{domain.KindFloor, domain.KindFloor, false},
		{domain.KindFloor, domain.KindRack, false},
		{domain.KindRoom, domain.KindBuilding, false},
		{domain.KindRack, domain.KindRoom, false},
	}

	for _, tc := range cases {
		got := domain.CanNest(tc.parent, tc.child)
		if got != tc.want {
			t.Errorf("CanNest(%s, %s) = %v; want %v", tc.parent, tc.child, got, tc.want)
		}
	}
}

func TestValidate_RootMustBeBuilding(t *testing.T) {
	loc := domain.Location{Kind: domain.KindFloor, Name: "طبقه ۲"}
	if err := loc.Validate(); err == nil {
		t.Fatal("expected error for root non-building location")
	}
}

func TestValidate_EmptyName(t *testing.T) {
	loc := domain.Location{Kind: domain.KindBuilding, Name: "   "}
	if err := loc.Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidate_NameIsTrimmed(t *testing.T) {
	loc := domain.Location{Kind: domain.KindBuilding, Name: "  ساختمان مرکزی  "}
	if err := loc.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.Name != "ساختمان مرکزی" {
		t.Fatalf("name not trimmed: %q", loc.Name)
	}
}

func TestValidate_LongNameRejected(t *testing.T) {
	loc := domain.Location{Kind: domain.KindBuilding, Name: string(make([]byte, 200))}
	if err := loc.Validate(); err == nil {
		t.Fatal("expected error for over-long name")
	}
}

func TestBuildTree_NestsChildrenAndKeepsOrder(t *testing.T) {
	floorOf := func(parent int64) *int64 { return &parent }
	all := []domain.Location{
		{ID: 1, Kind: domain.KindBuilding, Name: "ساختمان اصلی"},
		{ID: 2, ParentID: floorOf(1), Kind: domain.KindFloor, Name: "طبقه اول"},
		{ID: 3, ParentID: floorOf(2), Kind: domain.KindRoom, Name: "اتاق سرور"},
		{ID: 4, Kind: domain.KindBuilding, Name: "ساختمان دوم"},
	}

	roots := domain.BuildTree(all)
	if len(roots) != 2 {
		t.Fatalf("want 2 roots, got %d", len(roots))
	}
	if roots[0].Name != "ساختمان اصلی" || len(roots[0].Children) != 1 {
		t.Fatalf("unexpected tree root: %+v", roots[0])
	}
	room := roots[0].Children[0].Children[0]
	if room.Name != "اتاق سرور" {
		t.Fatalf("deep nesting broken: %+v", room)
	}
}
