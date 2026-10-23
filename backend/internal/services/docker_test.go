package services

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type failingDockerBuildReader struct {
	delivered bool
}

func (r *failingDockerBuildReader) Read(p []byte) (int, error) {
	if !r.delivered {
		r.delivered = true
		return copy(p, `{"stream":"Step 1/2\n"}`), nil
	}
	return 0, io.ErrUnexpectedEOF
}

func TestDecodeDockerBuildResponseReturnsStreamErrors(t *testing.T) {
	err := decodeDockerBuildResponse(&failingDockerBuildReader{}, nil)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected unexpected EOF, got %v", err)
	}
}

func TestDecodeDockerBuildResponseReturnsDockerErrors(t *testing.T) {
	err := decodeDockerBuildResponse(strings.NewReader(`{"error":"build failed"}`), nil)
	if err == nil || err.Error() != "build failed" {
		t.Fatalf("expected Docker build error, got %v", err)
	}
}

func TestEmitDockerLogLinesStreamsNonEmptyLines(t *testing.T) {
	var got []string
	emitDockerLogLines("Step 1/2\r\n\r\nInstalling package\nDone\n", func(line string) {
		got = append(got, line)
	})

	want := []string{"Step 1/2", "Installing package", "Done"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
