package services

import (
	"reflect"
	"testing"
)

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
