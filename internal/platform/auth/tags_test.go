// تست قرارداد JSON — کلیدهای پاسخ باید دقیقاً قرارداد v1 فرانت‌اند باشند
// (snake_case: full_name، is_superadmin، permissions به‌صورت آرایه).
package auth

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthUserJSONKeys(t *testing.T) {
	u := AuthUser{
		ID:           1,
		Username:     "admin",
		FullName:     "مدیر سامانه",
		JobTitle:     "مدیر فناوری اطلاعات",
		Role:         "admin",
		IsSuperadmin: true,
		Perms:        []string{"assets.create", "audit.view"},
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, key := range []string{
		`"id":`, `"username":`, `"full_name":`, `"job_title":`,
		`"role":`, `"is_superadmin":`, `"permissions":`,
	} {
		if !strings.Contains(s, key) {
			t.Fatalf("missing JSON key %s in %s", key, s)
		}
	}
	for _, legacy := range []string{`"ID"`, `"Username"`, `"FullName"`, `"Perms"`} {
		if strings.Contains(s, legacy) {
			t.Fatalf("legacy Go field name leaked into JSON: %s", s)
		}
	}
}

func TestAdminBypass(t *testing.T) {
	admin := &AuthUser{Role: "admin", Perms: []string{}}
	super := &AuthUser{IsSuperadmin: true, Role: "user", Perms: []string{}}
	plain := &AuthUser{Role: "user", Perms: []string{"assets.create"}}

	if !admin.Has("reports.approve") {
		t.Fatal("admin must bypass permission checks")
	}
	if !super.Has("anything.else") {
		t.Fatal("superadmin must bypass permission checks")
	}
	if !plain.Has("assets.create") {
		t.Fatal("granted code must match")
	}
	if plain.Has("assets.delete") {
		t.Fatal("missing code must not match")
	}
	var nilUser *AuthUser
	if nilUser.Has("assets.view") {
		t.Fatal("nil user must not have permissions")
	}
}

func TestParsePermissionCode(t *testing.T) {
	m, a, ok := ParsePermissionCode("assets.create")
	if !ok || m != "assets" || a != "create" {
		t.Fatalf("got %q %q %v", m, a, ok)
	}
	if _, _, ok := ParsePermissionCode("nodot"); ok {
		t.Fatal("code without dot must fail")
	}
}
