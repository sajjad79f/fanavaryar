// Package domain — قالب فرم و فیلدهای پویا (قرارداد v1).
// فیلدها به‌صورت JSONB ذخیره می‌شوند و هر نسخه فرم تغییرناپذیر است.
package domain

import (
        "encoding/json"
        "errors"
        "regexp"
        "strconv"
        "strings"
)

type FieldType string

const (
        FieldText     FieldType = "text"
        FieldTextarea FieldType = "textarea"
        FieldNumber   FieldType = "number"
        FieldDate     FieldType = "date"
        FieldSelect   FieldType = "select"
        FieldCheckbox FieldType = "checkbox"
)

var validTypes = map[FieldType]bool{
        FieldText: true, FieldTextarea: true, FieldNumber: true,
        FieldDate: true, FieldSelect: true, FieldCheckbox: true,
}

var keyRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// FieldDef تعریف یک فیلد فرم — هم‌سان با FieldDef فرانت‌اند.
type FieldDef struct {
        Key      string    `json:"key"`
        Label    string    `json:"label"`
        Type     FieldType `json:"type"`
        Required bool      `json:"required,omitempty"`
        Options  []string  `json:"options,omitempty"`
        Min      *float64  `json:"min,omitempty"`
        Max      *float64  `json:"max,omitempty"`
}

var ErrNotFound = errors.New("forms: not found")
var ErrCodeTaken = errors.New("forms: code already taken")

// ValidateFields اعتبارسنجی ساختار فیلدها (سروری؛ قرینه FieldsEditor کلاینت).
func ValidateFields(fields []FieldDef) error {
        if len(fields) == 0 {
                return errors.New("حداقل یک فیلد برای فرم لازم است")
        }
        seen := map[string]bool{}
        for _, f := range fields {
                if !keyRe.MatchString(f.Key) {
                        return errors.New("کلید فیلد باید با حرف انگلیسی کوچک شروع شود و فقط شامل حرف کوچک، رقم و زیرخط باشد")
                }
                if seen[f.Key] {
                        return errors.New("کلید فیلدها باید یکتا باشد: " + f.Key)
                }
                seen[f.Key] = true
                if strings.TrimSpace(f.Label) == "" {
                        return errors.New("برچسب فیلد الزامی است")
                }
                if !validTypes[f.Type] {
                        return errors.New("نوع فیلد نامعتبر است: " + f.Key)
                }
                if f.Type == FieldSelect && len(f.Options) == 0 {
                        return errors.New("برای فیلد انتخابی حداقل یک گزینه لازم است: " + f.Key)
                }
                if f.Type == FieldNumber && f.Min != nil && f.Max != nil && *f.Min > *f.Max {
                        return errors.New("کمینه نباید از بیشینه بزرگ‌تر باشد: " + f.Key)
                }
        }
        return nil
}

// ParseFields رمزگشایی JSONB فیلدها.
func ParseFields(raw []byte) ([]FieldDef, error) {
        var out []FieldDef
        if len(raw) == 0 {
                return out, nil
        }
        if err := json.Unmarshal(raw, &out); err != nil {
                return nil, err
        }
        return out, nil
}

// ValidateFormData بررسی داده پرشدهٔ فرم بر اساس تعریف فیلدها (سمت سرور).
func ValidateFormData(fields []FieldDef, data map[string]any) error {
        for _, f := range fields {
                v, present := data[f.Key]
                if !present || v == nil || v == "" {
                        if f.Required {
                                return errors.New("فیلد «" + f.Label + "» الزامی است")
                        }
                        continue
                }
                switch f.Type {
                case FieldNumber:
                        num, ok := toFloat(v)
                        if !ok {
                                return errors.New("فیلد «" + f.Label + "» باید عدد باشد")
                        }
                        if f.Min != nil && num < *f.Min {
                                return errors.New("فیلد «" + f.Label + "» نباید کمتر از حد مجاز باشد")
                        }
                        if f.Max != nil && num > *f.Max {
                                return errors.New("فیلد «" + f.Label + "» نباید بیشتر از حد مجاز باشد")
                        }
                case FieldSelect:
                        sv, ok := v.(string)
                        if !ok {
                                return errors.New("فیلد «" + f.Label + "» نامعتبر است")
                        }
                        found := false
                        for _, o := range f.Options {
                                if o == sv {
                                        found = true
                                        break
                                }
                        }
                        if !found {
                                return errors.New("مقدار فیلد «" + f.Label + "» در گزینه‌های مجاز نیست")
                        }
                case FieldCheckbox:
                        if _, ok := v.(bool); !ok {
                                return errors.New("فیلد «" + f.Label + "» باید بله/خیر باشد")
                        }
                case FieldText, FieldTextarea, FieldDate:
                        if _, ok := v.(string); !ok {
                                return errors.New("فیلد «" + f.Label + "» باید متنی باشد")
                        }
                }
        }
        return nil
}

func toFloat(v any) (float64, bool) {
        switch x := v.(type) {
        case float64:
                return x, true
        case float32:
                return float64(x), true
        case int:
                return float64(x), true
        case int64:
                return float64(x), true
        case string:
                f, err := strconv.ParseFloat(x, 64)
                return f, err == nil
        }
        return 0, false
}
