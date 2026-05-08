package main

import (
	"context"
	"fmt"
	"os"

	"bili-download/internal/xhs"
)

func main() {
	if err := xhs.CreateLivePhoto(context.Background(),
		os.Args[1], os.Args[2], os.Args[3]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fi, _ := os.Stat(os.Args[3])
	fmt.Printf("OK: %s (%d B)\n", os.Args[3], fi.Size())
}
