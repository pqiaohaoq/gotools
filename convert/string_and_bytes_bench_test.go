package convert

import (
	"bytes"
	"strings"
	"testing"
)

var (
	L   = 1024 * 1024
	str = strings.Repeat("a", L)
	s   = bytes.Repeat([]byte{'a'}, L)
)

func BenchmarkString2Slice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bt := []byte(str)
		if len(bt) != L {
			b.Fatal()
		}
	}
}

func BenchmarkString2SliceReflect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bt := StringToSliceByte(str)
		if len(bt) != L {
			b.Fatal()
		}
	}
}

func BenchmarkSlice2String(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ss := string(s)
		if len(ss) != L {
			b.Fatal()
		}
	}
}

func BenchmarkSliceByteToString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ss := SliceByteToString(s)
		if len(ss) != L {
			b.Fatal()
		}
	}
}
