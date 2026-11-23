// Package main — نقطه ورود سامانه فناوریار.
// کارهای واقعی راه‌اندازی در پکیج bootstrap انجام می‌شود.
package main

import (
	"context"
	"fmt"
	"os"

	"fanavaryar/internal/bootstrap"
)

func main() {
	ctx := context.Background()

	if err := bootstrap.Run(ctx, os.Getenv); err != nil {
		// خطای راه‌اندازی هرگز بی‌صدا از سرویس خارج نمی‌شود؛
		// بدون این خط، systemd فقط exit-code=1 نشان می‌دهد و علت در journald دیده نمی‌شود.
		fmt.Fprintln(os.Stderr, "startup failed:", err)
		os.Exit(1)
	}
}
