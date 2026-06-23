package obj

import (
	"strings"
	"testing"
)

func TestParseStatsAndWrite(t *testing.T) {
	input := strings.Join([]string{
		"# sample",
		"vn 0 1 0",
		"v 1 2 3 1.0",
		"vt 0.5 0.25",
		"f 1/1/1 1/1/1 1/1/1 # face",
		"",
	}, "\n")

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	stats := doc.Stats()
	if stats.Vertices != 1 || stats.Normals != 1 || stats.TexCoords != 1 || stats.Faces != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	var output strings.Builder
	if err := doc.Write(&output); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	written := output.String()
	if !strings.Contains(written, "v 1 2 3 1.0") {
		t.Fatalf("written OBJ missing vertex extra value:\n%s", written)
	}
	if !strings.Contains(written, "f 1/1/1 1/1/1 1/1/1 # face") {
		t.Fatalf("written OBJ missing face comment:\n%s", written)
	}
}

func TestTransform(t *testing.T) {
	doc, err := Parse(strings.NewReader(strings.Join([]string{
		"v 1 2 3",
		"vn 0 0 1",
		"f 1//1 1//1 1//1",
	}, "\n")))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	options := DefaultTransformOptions()
	options.Scale = Vec3{X: 2, Y: 2, Z: 2}
	options.Translate = Vec3{X: 1, Y: 0, Z: -1}
	if err := doc.Transform(options); err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	got := doc.Records[0].Vertex.Position
	want := Vec3{X: 3, Y: 4, Z: 5}
	if got != want {
		t.Fatalf("transformed position = %+v, want %+v", got, want)
	}
	gotNormal := doc.Records[1].Normal.Direction
	if gotNormal != (Vec3{X: 0, Y: 0, Z: 1}) {
		t.Fatalf("transformed normal = %+v", gotNormal)
	}
}

func TestTransformWithMatrix(t *testing.T) {
	doc, err := Parse(strings.NewReader(strings.Join([]string{
		"v 1 2 3",
		"vn 0 0 1",
		"f 1//1 1//1 1//1",
	}, "\n")))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	matrix := Mat4{
		{2, 0, 0, 10},
		{0, 3, 0, 20},
		{0, 0, 4, 30},
		{0, 0, 0, 1},
	}
	options := DefaultTransformOptions()
	options.Matrix = &matrix
	if err := doc.Transform(options); err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	got := doc.Records[0].Vertex.Position
	want := Vec3{X: 12, Y: 26, Z: 42}
	if got != want {
		t.Fatalf("transformed position = %+v, want %+v", got, want)
	}
	gotNormal := doc.Records[1].Normal.Direction
	if gotNormal != (Vec3{X: 0, Y: 0, Z: 1}) {
		t.Fatalf("transformed normal = %+v", gotNormal)
	}
}

func TestTransformRejectsProjectiveMatrix(t *testing.T) {
	doc, err := Parse(strings.NewReader("v 1 2 3\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	matrix := IdentityMat4()
	matrix[3][2] = 1
	options := DefaultTransformOptions()
	options.Matrix = &matrix
	if err := doc.Transform(options); err == nil {
		t.Fatal("Transform() error = nil, want projective matrix error")
	}
}

func TestTriangulateQuad(t *testing.T) {
	doc, err := Parse(strings.NewReader("f 1 2 3 4 # quad\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	result := doc.Triangulate()
	if result.Polygons != 1 || result.NewTriangles != 2 {
		t.Fatalf("unexpected triangulate result: %+v", result)
	}

	var output strings.Builder
	if err := doc.Write(&output); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	want := "f 1 2 3 # quad\nf 1 3 4\n"
	if output.String() != want {
		t.Fatalf("triangulated OBJ = %q, want %q", output.String(), want)
	}
}

func TestSliceReusesIntersectionVertexForSharedEdge(t *testing.T) {
	doc, err := Parse(strings.NewReader(strings.Join([]string{
		"v -1 0 0",
		"v 1 0 0",
		"v -1 1 0",
		"v 1 1 0",
		"f 1 2 3",
		"f 3 2 4",
	}, "\n")))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	options, err := ParseSliceSide("x+")
	if err != nil {
		t.Fatalf("ParseSliceSide() error = %v", err)
	}
	if _, err := doc.Slice(options); err != nil {
		t.Fatalf("Slice() error = %v", err)
	}

	stats := doc.Stats()
	if stats.Vertices != 5 {
		t.Fatalf("sliced vertex count = %d, want 5", stats.Vertices)
	}

	var output strings.Builder
	if err := doc.Write(&output); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	written := output.String()
	if !strings.Contains(written, "f 3 1 4") || !strings.Contains(written, "f 5 4 1 2") {
		t.Fatalf("sliced faces do not share the cut-edge vertex:\n%s", written)
	}
}
