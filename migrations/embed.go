// Package migrations — فایل‌های SQL را برای اجرای درون برنامه embed می‌کند.
// با این روش، فایل اجرایی خودش نسخه‌های Migration را حمل می‌کند و
// استقرار به هیچ ابزار بیرونی وابسته نیست.
package migrations

import "embed"

//go:embed *.up.sql
var FS embed.FS
