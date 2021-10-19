package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

const flagInputStdin = "-"

var flagJobs = flag.Int("jobs", runtime.NumCPU(), "number of parallel jobs")

func countFile(name string, jobs int) (int, error) {
	f, err := os.Stdin, error(nil)

	if name != flagInputStdin {
		f, err = os.Open(name)
		if err != nil {
			return -1, fmt.Errorf("open %q: %w", name, err)
		}

		defer f.Close()
	}

	input, err := io.ReadAll(f)
	if err != nil {
		return -1, fmt.Errorf("read %q: %w", f.Name(), err)
	}

	return wordCountParallel(input, jobs), nil
}

func usage() {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] [files]\n", exe)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	files := flag.Args()

	if len(files) == 0 {
		files = append(files, flagInputStdin)
	}

	type retVal struct {
		File string
		Words int
		Error error
	}

	ret := make(chan retVal)
	for _, f := range files {
		file := f

		go func() {
			n, err := countFile(file, *flagJobs)
			if err != nil {
				err = fmt.Errorf("count %q: %w", file, err)
			}

			ret <- retVal{
				File: file,
				Words: n,
				Error: err,
			}
		}()
	}

	for i := 0; i < len(files); i++ {
		r := <-ret

		if r.Error != nil {
			fmt.Println(r.File, r.Error)
			continue
		}

		fmt.Println(r.File, r.Words)
	}
}
