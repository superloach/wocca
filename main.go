package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
			return -1, err
		}

		defer f.Close()
	}

	input, err := io.ReadAll(f)
	if err != nil {
		return -1, err
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

	rets := make([]chan retVal, 0, len(files))
	for _, f := range files {
		file := f
		ret := make(chan retVal)
		rets = append(rets, ret)

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

	retVals := make([]retVal, 0, len(files))
	total := 0
	cols := 0.0

	for _, ret := range rets {
		val := <-ret
		retVals = append(retVals, val)

		if val.Words > 0 {
			cols = math.Max(cols, math.Log10(float64(val.Words)))
		}

		total += val.Words
	}

	if len(files) > 1 {
		cols = math.Max(cols, math.Log10(float64(total)))

		retVals = append(retVals, retVal{
			File: "total",
			Words: total,
			Error: nil,
		})
	}

	colstr := strconv.Itoa(int(math.Ceil(cols)))

	for _, val := range retVals {
		if val.Error != nil {
			fstr := "%" + colstr + "s %s\n"
			fmt.Fprintf(os.Stderr, fstr, "#", val.Error)
			continue
		}

		fmt.Printf("%" + colstr + "d %s\n", val.Words, val.File)
	}
}
