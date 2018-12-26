package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// read bytes from reader and submit 16 byte sub-slices to returned channel.
func read(r io.Reader) <-chan []byte {
	rc := make(chan []byte)
	go func() {
		// we'll read a multiple of a block size worth of data.  this value must
		// also be divisible by our stride of 16 bytes per row.
		bs := 4096
		data := make([]byte, bs*4, bs*4)
		for {
			bytes, err := r.Read(data)

			if err != nil {
				break
			}

			// split data up into 16 byte sub-slices and send each slice into
			// rc channel.
			min := func(x, y int) int {
				if x < y {
					return x
				}
				return y
			}
			for s := data[:bytes]; len(s) != 0; s = s[min(len(s), 16):] {
				rc <- s[:min(len(s), 16)]
			}
		}
		close(rc)
	}()

	return rc
}

// ascii returns a printable string representation of a series of bytes.
func ascii(bytes []byte) string {
	printable := func(b byte) bool {
		r := rune(b)
		return r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsPunct(r) || unicode.IsDigit(r))
	}

	char := func(b byte) byte {
		if printable(b) {
			return b
		}
		return byte('.')
	}

	sb := strings.Builder{}
	sb.Grow(len(bytes))

	for _, b := range bytes {
		sb.WriteByte(char(b))
	}

	return sb.String()
}

// hex returns a hexadecimal string representation of a series of bytes.
func hex(bytes []byte) string {
	rc := ""
	end := func(s []byte, i int) int {
		if i == len(s)-1 {
			return 0
		}
		if (i+1)%4 == 0 {
			return 2
		}
		return 1
	}

	for i, b := range bytes {
		rc += fmt.Sprintf("%02x%*s", b, end(bytes, i), "")
	}
	return rc
}

func hd(path string, w io.Writer, seek int64, length int64) error {
	inf, err := os.Open(path)
	if err != nil {
		return err
	}

	if _, err := inf.Seek(seek, 0); err != nil {
		return err
	}

	in := func() io.Reader {
		if length != 0 {
			return io.LimitReader(inf, length)
		}
		return inf
	}()

	count := seek
	for s := range read(in) {
		fmt.Fprintf(w, "%08x %-50.50s  | %-16.16s |\n", count, hex(s), ascii(s))
		count += int64(len(s))
	}

	inf.Close()
	return nil
}

func main() {
	seek := flag.Int("s", 0, "seek n bytes into file")
	length := flag.Int("n", 0, "only read a limited number of bytes from input.")
	flag.Parse()
	for _, file := range os.Args[1:] {
		hd(file, os.Stdout, int64(*seek), int64(*length))
	}
}
