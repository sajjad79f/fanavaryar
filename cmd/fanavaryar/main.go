// Package main — نقطه ورود سامانه فناوریار.
// کارهای واقعی راه‌اندازی در پکیج bootstrap انجام می‌شود.
package main

import (
	"context"
	"os"

	"fanavaryar/internal/bootstrap"
)

func main() {
	ctx := context.Background()

	if err := bootstrap.Run(ctx, os.Getenv); err != nil {
		os.Exit(1)
	}
}
