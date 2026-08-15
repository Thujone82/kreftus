//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
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

var difficulties = []string{"easy", "medium", "hard", "diabolical"}

var sampleCounts = map[string]int{
	"easy":       2500,
	"medium":     2000,
	"hard":       1000,
	"diabolical": 250,
}

type puzzleEntry struct {
	ID     string `json:"id"`
	Givens string `json:"givens"`
}

type puzzleFile struct {
	Source     string        `json:"source"`
	Easy       []puzzleEntry `json:"easy"`
	Medium     []puzzleEntry `json:"medium"`
	Hard       []puzzleEntry `json:"hard"`
	Diabolical []puzzleEntry `json:"diabolical"`
}

func main() {
	rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1))
	out := puzzleFile{
		Source: "Sudoku Exchange Puzzle Bank (Grant McLean), public domain — https://github.com/grantm/sudoku-exchange-puzzle-bank",
	}
	for _, d := range difficulties {
		n := sampleCounts[d]
		fmt.Fprintf(os.Stderr, "sampling %s (%d)...\n", d, n)
		picked, err := sampleBank(bankBase+d+".txt", n, rng)
		if err != nil {
			fmt.Fprintf(os.Stderr, "download %s: %v\n", d, err)
			os.Exit(1)
		}
		entries := make([]puzzleEntry, 0, n)
		for _, p := range picked {
			if sol := solveSudoku(p.givens); len(sol) != 81 {
				fmt.Fprintf(os.Stderr, "skip %s %s: unsolvable\n", d, p.id)
				continue
			}
			entries = append(entries, puzzleEntry{ID: p.id, Givens: p.givens})
		}
		if len(entries) < n {
			fmt.Fprintf(os.Stderr, "%s: only %d solvable of %d sampled\n", d, len(entries), n)
			os.Exit(1)
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
		switch d {
		case "easy":
			out.Easy = entries
		case "medium":
			out.Medium = entries
		case "hard":
			out.Hard = entries
		case "diabolical":
			out.Diabolical = entries
		}
		fmt.Fprintf(os.Stderr, "  %s: %d puzzles\n", d, len(entries))
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	bom := []byte{0xEF, 0xBB, 0xBF}
	payload := append(append([]byte{}, data...), '\n')
	if err := os.WriteFile("puzzles.json", append(bom, payload...), 0644); err != nil {
		panic(err)
	}
	if err := writeGzip("puzzles.json.gz", payload); err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, "wrote puzzles.json and puzzles.json.gz\n")
}

func writeGzip(path string, data []byte) error {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := zw.Write(data); err != nil {
		_ = zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

type rawPuzzle struct {
	id     string
	givens string
}

func sampleBank(url string, n int, rng *rand.Rand) ([]rawPuzzle, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "sudokutui-import/1.0")
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %s: %s", resp.Status, body)
	}
	// Reservoir sample n records (100 bytes each: hash puzzle rating).
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 256), 1024)
	picked := make([]rawPuzzle, 0, n)
	seen := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, ok := parseBankLine(line)
		if !ok {
			continue
		}
		seen++
		if len(picked) < n {
			picked = append(picked, p)
			continue
		}
		j := rng.IntN(seen)
		if j < n {
			picked[j] = p
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(picked) < n {
		return nil, fmt.Errorf("only %d records in bank (need %d)", len(picked), n)
	}
	return picked, nil
}

func parseBankLine(line string) (rawPuzzle, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return rawPuzzle{}, false
	}
	id := fields[0]
	givens := fields[1]
	if len(id) < 8 || len(givens) != 81 {
		return rawPuzzle{}, false
	}
	for i := 0; i < 81; i++ {
		c := givens[i]
		if (c < '0' || c > '9') && c != '.' {
			return rawPuzzle{}, false
		}
	}
	// Normalize '.' to '0'.
	b := []byte(givens)
	for i, c := range b {
		if c == '.' {
			b[i] = '0'
		}
	}
	return rawPuzzle{id: id, givens: string(b)}, true
}
