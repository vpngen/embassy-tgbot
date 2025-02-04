package main

import (
	"fmt"
	"os"
	"testing"
)

func TestGetCaptchaText(t *testing.T) {
	const (
		count = 100
		maxl  = 8
	)

	for i := 0; i < count; i++ {
		like, text := GetCaptchaText()

		if text == "" {
			t.Errorf("GetCaptchaText() returned empty string")
		}

		if like == "" {
			t.Errorf("GetCaptchaText() returned invalid index %d", i)
		}

		fmt.Fprintf(os.Stderr, "%s\nlike=%s\n\n", text, like)
		fmt.Fprintln(os.Stderr, "========================================")
	}
}
