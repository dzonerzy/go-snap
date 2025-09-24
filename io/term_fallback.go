package snapio

import "os"

func fallbackTermSizeFromEnv() (int, int) {
	var w, h int
	if c := os.Getenv("COLUMNS"); len(c) > 0 {
		if v := atoi(c); v > 0 {
			w = v
		}
	}
	if l := os.Getenv("LINES"); len(l) > 0 {
		if v := atoi(l); v > 0 {
			h = v
		}
	}
	return w, h
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
