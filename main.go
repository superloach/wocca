package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
)

func isWhite(b byte) bool {
	switch b {
	case ' ', '\t', '\r', '\n':
		return true
	}

	return false
}

func makeChunks(input []byte, n int) [][]byte {
	type chunk struct {
		start int
		end int
	}

	chunks := make([]chunk, 0, n)

	size := len(input) / n
	for i := 0; i < n; i++ {
		last := 0
		if len(chunks) > 0 {
			last = chunks[len(chunks)-1].end
		}

		c := chunk{
			start: last,
			end: last + size,
		}

		if c.end > len(input) {
			c.end = len(input)
		}

		for c.end < len(input) && !isWhite(input[c.end-1]) {
			c.end++
		}

		if c.start != c.end {
			chunks = append(chunks, c)
		}
	}

	bss := make([][]byte, 0, len(chunks))
	for _, c := range chunks {
		bss = append(bss, input[c.start:c.end])
	}

	return bss
}

func wordCount(input []byte) int {
	start := 0
	for isWhite(input[start]) {
		start++
	}

	end := len(input) - 1
	for isWhite(input[end]) {
		end--
	}

	wc := 0
	last := true
	for i := start; i <= end; i++ {
		white := isWhite(input[i])
		if !white && last {
			wc++
		}
		last = white
	}

	return wc
}

func wordCountParallel(input []byte, jobs int) int {
	chunks := makeChunks(input, jobs)

	ret := make(chan int)
	for _, c := range chunks {
		chunk := c
		go func() {
			ret <- wordCount(chunk)
		}()
	}

	wc := 0
	for i := 0; i < len(chunks); i++ {
		wc += <-ret
	}

	return wc
}

var (
	flagJobs = flag.Int("jobs", runtime.NumCPU(), "number of parallel jobs")
	flagInput = flag.String("input", "", "input file")
)

func main() {
	flag.Parse()

	f, err := os.Open(*flagInput)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	input, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}

	fmt.Println(wordCountParallel(input, *flagJobs))
}
