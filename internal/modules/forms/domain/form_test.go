// تست اعتبارسنجی فیلدهای پویا و داده فرم — قرارداد ماژول فرم‌ها.
package domain

import (
	"strings"
	"testing"
)

func validFields() []FieldDef {
	min := 1.0
	max := 100.0
	return []FieldDef{
		{Key: "summary", Label: "خلاصه", Type: FieldText, Required: true},
		{Key: "detail", Label: "شرح", Type: FieldTextarea},
		{Key: "hours", Label: "ساعت کار", Type: FieldNumber, Min: &min, Max: &max},
		{Key: "done_date", Label: "تاریخ", Type: FieldDate},
		{Key: "kind", Label: "نوع", Type: FieldSelect, Options: []string{"نرم‌افزاری", "سخت‌افزاری"}},
		{Key: "urgent", Label: "فوری", Type: FieldCheckbox},
	}
}

func TestValidateFields(t *testing.T) {
	if err := ValidateFields(validFields()); err != nil {
		t.Fatalf("valid fields rejected: %v", err)
	}
	if err := ValidateFields(nil); err == nil {
		t.Fatal("empty fields must fail")
	}

	bad := validFields()
	bad[0].Key = "Summary"
	if err := ValidateFields(bad); err == nil || !strings.Contains(err.Error(), "کلید") {
		t.Fatalf("uppercase key must fail: %v", err)
	}

	dup := validFields()
	dup[1].Key = "summary"
	if err := ValidateFields(dup); err == nil {
		t.Fatal("duplicate key must fail")
	}

	noOpts := validFields()
	noOpts[4].Options = nil
	if err := ValidateFields(noOpts); err == nil {
		t.Fatal("select without options must fail")
	}
}

func TestValidateFormData(t *testing.T) {
	fields := validFields()

	// همه مقادیر مجاز
	data := map[string]any{
		"summary": "تعویض منبع تغذیه", "detail": "شرح کامل",
		"hours": float64(4), "done_date": "2025-11-21",
		"kind": "سخت‌افزاری", "urgent": true,
	}
	if err := ValidateFormData(fields, data); err != nil {
		t.Fatalf("valid data rejected: %v", err)
	}

	// فیلد الزامی غایب
	if err := ValidateFormData(fields, map[string]any{}); err == nil {
		t.Fatal("missing required field must fail")
	}

	// عدد خارج از بازه
	data2 := map[string]any{"summary": "x", "hours": float64(500)}
	if err := ValidateFormData(fields, data2); err == nil {
		t.Fatal("out-of-range number must fail")
	}

	// مقدار خارج از گزینه‌ها
	data3 := map[string]any{"summary": "x", "kind": "برقی"}
	if err := ValidateFormData(fields, data3); err == nil {
		t.Fatal("invalid select option must fail")
	}

	// فیلد اختیاری غایب: مجاز
	data4 := map[string]any{"summary": "x"}
	if err := ValidateFormData(fields, data4); err != nil {
		t.Fatalf("optional missing must pass: %v", err)
	}
}
