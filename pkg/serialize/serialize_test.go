package serialize

import (
	"errors"
	"io"
	"testing"

	"github.com/spf13/afero"
)

func fx(x uint64) uint64 {
	return x + 100
}

func TestSerialize(t *testing.T) {
	file, err := afero.NewMemMapFs().Create("TestSerialize")
	if err != nil {
		t.Fatal(err)
	}

	k := 30

	var wrote, entryLen int
	for x := uint64(0); x < 100; x++ {
		w, err := Write(file, int64(wrote), fx(x), &x, nil, nil, nil, k)
		if err != nil {
			t.Fatalf("cannot write x=%d: %v", x, err)
		}
		wrote += w
		entryLen = w
	}

	var read int
	for {
		e, r, err := Read(file, int64(read), entryLen, k)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("cannot read: %v", err)
		}
		read += r
		if e.Fx != *e.X+100 {
			t.Fatalf("unexpected result f(x)=%d, expected f(x)=%d", e.Fx, *e.X+100)
		}
	}
}
