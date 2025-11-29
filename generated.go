package main

import (
	"bufio"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func nextIndex(dir string) int {
	d, err := os.ReadDir(dir)
	if err != nil {
		return 1
	}
	max := 0
	for _, de := range d {
		name := de.Name()
		if strings.HasSuffix(name, ".txt") {
			base := strings.TrimSuffix(name, ".txt")
			n, err := strconv.Atoi(base)
			if err == nil && n > max {
				max = n
			}
		}
	}
	if max <= 0 {
		return 1
	}
	return max + 1
}

func openFile(dir string, idx int) (*os.File, *bufio.Writer, string) {
	fp := filepath.Join(dir, strconv.Itoa(idx)+".txt")
	f, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, nil, fp
	}
	w := bufio.NewWriterSize(f, 1<<20)
	return f, w, fp
}

func maxBytes() int64 {
	return int64(300 * 1024 * 1024)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	dir := "data"
	os.MkdirAll(dir, 0755)
	idx := nextIndex(dir)
	f, w, _ := openFile(dir, idx)
	if f == nil {
		return
	}
	defer f.Close()
	defer w.Flush()
	charset := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789*.#")
	var size int64 = 0
	max := maxBytes()
	buf := make([]byte, 0, 64)
	for {
		l := 8 + rand.Intn(5)
		buf = buf[:0]
		for i := 0; i < l; i++ {
			buf = append(buf, charset[rand.Intn(len(charset))])
		}
		buf = append(buf, '\n')
		n, err := w.Write(buf)
		if err != nil {
			return
		}
		size += int64(n)
		if size >= max {
			w.Flush()
			f.Close()
			idx++
			if idx >= 100 {
				break
			}
			f, w, _ = openFile(dir, idx)
			if f == nil {
				return
			}
			size = 0
		}
	}
}
