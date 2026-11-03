//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const bankBase = "https://raw.githubusercontent.com/grantm/sudoku-exchange-puzzle-bank/master/"

func main() {
	rng := rand.New(rand.NewPCG(20261103, 250))
	names := []string{"easy", "medium", "hard", "diabolical"}
	var b strings.Builder
	b.WriteString("# 250 puzzles of each difficulty from the Sudoku Exchange Puzzle Bank\n")
	b.WriteString("# https://github.com/grantm/sudoku-exchange-puzzle-bank\n")
	for _, name := range names {
		fmt.Fprintf(os.Stderr, "sampling %s...\n", name)
		picked, err := sample(bankBase+name+".txt", 250, rng)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
			os.Exit(1)
		}
		sort.Strings(picked)
		fmt.Fprintf(&b, "# %s\n", name)
		for _, g := range picked {
			b.WriteString(g)
			b.WriteByte('\n')
		}
		fmt.Fprintf(os.Stderr, "  %s: %d\n", name, len(picked))
	}
	if err := os.WriteFile("puzzles.txt", []byte(b.String()), 0644); err != nil {
		panic(err)
	}
}

func sample(url string, n int, rng *rand.Rand) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "sudoku-solver-sample/1.0")
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("HTTP %s: %s", resp.Status, body)
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 256), 1024)
	picked := make([]string, 0, n)
	seen := 0
	for sc.Scan() {
		g, ok := givens(sc.Text())
		if !ok {
			continue
		}
		seen++
		if len(picked) < n {
			picked = append(picked, g)
			continue
		}
		if j := rng.IntN(seen); j < n {
			picked[j] = g
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(picked) < n {
		return nil, fmt.Errorf("only %d puzzles (need %d)", len(picked), n)
	}
	return picked, nil
}

func givens(line string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 || len(fields[1]) != 81 {
		return "", false
	}
	g := fields[1]
	for i := 0; i < 81; i++ {
		c := g[i]
		if c == '.' {
			continue
		}
		if c < '0' || c > '9' {
			return "", false
		}
	}
	buf := []byte(g)
	for i, c := range buf {
		if c == '.' {
			buf[i] = '0'
		}
	}
	return string(buf), true
}
