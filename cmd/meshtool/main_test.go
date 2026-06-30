package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseMat4Flag(t *testing.T) {
	matrix, err := parseMat4Flag("1 0 0 10; 0 1 0 20; 0 0 1 30; 0 0 0 1")
	if err != nil {
		t.Fatalf("parseMat4Flag() error = %v", err)
	}

	if matrix[0][3] != 10 || matrix[1][3] != 20 || matrix[2][3] != 30 || matrix[3][3] != 1 {
		t.Fatalf("unexpected matrix: %+v", matrix)
	}
}

func TestParseMat4FlagIgnoresBracketsAndCommas(t *testing.T) {
	matrix, err := parseMat4Flag("[[1, 0, 0, 10], [0, 1, 0, 20], [0, 0, 1, 30], [0, 0, 0, 1]]")
	if err != nil {
		t.Fatalf("parseMat4Flag() error = %v", err)
	}

	if matrix[0][3] != 10 || matrix[1][3] != 20 || matrix[2][3] != 30 || matrix[3][3] != 1 {
		t.Fatalf("unexpected matrix: %+v", matrix)
	}
}

func TestParseMat4FlagRejectsWrongCount(t *testing.T) {
	if _, err := parseMat4Flag("1 0 0 1"); err == nil {
		t.Fatal("parseMat4Flag() error = nil, want wrong-count error")
	}
}

func TestRunChainAppliesOperationsInOrder(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"v 1 2 3",
		"f 1 1 1 1",
	}, "\n"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{
		"chain", "-", "-",
		"transform", "-scale", "2",
		"triangulate",
		"transform", "-tx", "1",
	}, input, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run(chain) error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "v 3 4 6") {
		t.Fatalf("chained output missing transformed vertex:\n%s", output)
	}
	if !strings.Contains(output, "f 1 1 1\nf 1 1 1\n") {
		t.Fatalf("chained output missing triangulated face:\n%s", output)
	}
	if !strings.Contains(stderr.String(), "applied 3 operation(s)") {
		t.Fatalf("stderr missing operation count: %q", stderr.String())
	}
}

func TestRunEdgesPrintsLengthStats(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"v 0 0 0",
		"v 3 0 0",
		"v 0 4 0",
		"f 1 2 3",
	}, "\n"))
	var stdout bytes.Buffer

	err := run([]string{"edges", "-"}, input, &stdout, ioDiscard{})
	if err != nil {
		t.Fatalf("run(edges) error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Edges: 3") || !strings.Contains(output, "Edge length max: 5") {
		t.Fatalf("edges output missing stats:\n%s", output)
	}
}

func TestRunRemeshSplitsLongEdges(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"v 0 0 0",
		"v 4 0 0",
		"v 0 3 0",
		"f 1 2 3",
	}, "\n"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"remesh", "-target", "4.5", "-max-factor", "1", "-iterations", "1", "-", "-"}, input, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run(remesh) error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "v 2 1.5 0") || !strings.Contains(output, "f 1 2 4") || !strings.Contains(output, "f 1 4 3") {
		t.Fatalf("remesh output missing split geometry:\n%s", output)
	}
	if !strings.Contains(stderr.String(), "split 1 edge(s)") {
		t.Fatalf("stderr missing remesh counts: %q", stderr.String())
	}
}

func TestRunSliceClipsPositiveSide(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"v -1 0 0",
		"v 1 0 0",
		"v 1 1 0",
		"f 1 2 3",
	}, "\n"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"slice", "x+", "-", "-"}, input, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run(slice) error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "v 0 0 0") || !strings.Contains(output, "v 0 0.5 0") {
		t.Fatalf("sliced output missing boundary vertices:\n%s", output)
	}
	if strings.Contains(output, "v -1 0 0") {
		t.Fatalf("sliced output kept unused negative-side vertex:\n%s", output)
	}
	if !strings.Contains(output, "f 3 4 1 2") {
		t.Fatalf("sliced output missing clipped face:\n%s", output)
	}
	if !strings.Contains(stderr.String(), "split 1, discarded 0") {
		t.Fatalf("stderr missing slice counts: %q", stderr.String())
	}
}

func TestRunChainRejectsUnexpectedTransformArgument(t *testing.T) {
	err := run([]string{
		"chain", "-", "-",
		"transform", "extra.obj",
	}, strings.NewReader("v 1 2 3\n"), ioDiscard{}, ioDiscard{})
	if err == nil {
		t.Fatal("run(chain) error = nil, want unexpected transform argument error")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}
