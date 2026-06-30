package obj

import (
	"fmt"
	"math"
	"strconv"
)

type TriangleMesh struct {
	Vertices []Vec3
	Faces    [][3]int
}

type EdgeLengthStats struct {
	Edges  int
	Min    float64
	Max    float64
	Mean   float64
	Median float64
	P90    float64
	P95    float64
}

type RemeshOptions struct {
	TargetLength float64
	MaxFactor    float64
	Iterations   int
}

type RemeshResult struct {
	InputVertices  int
	InputFaces     int
	OutputVertices int
	OutputFaces    int
	Splits         int
	Iterations     int
}

func (doc *Document) ToTriangleMesh() (*TriangleMesh, error) {
	var vertices []Vec3
	var faces [][3]int
	var vertexCount int
	var texCoordCount int
	var normalCount int

	for _, record := range doc.Records {
		switch {
		case record.Vertex != nil:
			vertices = append(vertices, record.Vertex.Position)
			vertexCount++
		case record.TexCoord != nil:
			texCoordCount++
		case record.Normal != nil:
			normalCount++
		case record.Face != nil:
			if len(record.Face.Refs) < 3 {
				continue
			}
			refs := make([]int, len(record.Face.Refs))
			for index, rawRef := range record.Face.Refs {
				ref, err := ParseFaceRef(rawRef)
				if err != nil {
					return nil, err
				}
				resolved, err := ref.Resolve(vertexCount, texCoordCount, normalCount)
				if err != nil {
					return nil, err
				}
				refs[index] = resolved.V - 1
			}
			for index := 1; index < len(refs)-1; index++ {
				faces = append(faces, [3]int{refs[0], refs[index], refs[index+1]})
			}
		}
	}

	return &TriangleMesh{Vertices: vertices, Faces: faces}, nil
}

func (doc *Document) EdgeLengthStats() (EdgeLengthStats, error) {
	mesh, err := doc.ToTriangleMesh()
	if err != nil {
		return EdgeLengthStats{}, err
	}
	return mesh.EdgeLengthStats(), nil
}

func (mesh *TriangleMesh) EdgeLengthStats() EdgeLengthStats {
	lengths := mesh.uniqueEdgeLengths()
	if len(lengths) == 0 {
		return EdgeLengthStats{}
	}

	sortFloat64s(lengths)
	var sum float64
	for _, length := range lengths {
		sum += length
	}

	return EdgeLengthStats{
		Edges:  len(lengths),
		Min:    lengths[0],
		Max:    lengths[len(lengths)-1],
		Mean:   sum / float64(len(lengths)),
		Median: percentileSorted(lengths, 0.5),
		P90:    percentileSorted(lengths, 0.9),
		P95:    percentileSorted(lengths, 0.95),
	}
}

func (doc *Document) Remesh(options RemeshOptions) (RemeshResult, error) {
	mesh, err := doc.ToTriangleMesh()
	if err != nil {
		return RemeshResult{}, err
	}

	result, err := mesh.Remesh(options)
	if err != nil {
		return RemeshResult{}, err
	}
	doc.Records = mesh.records()
	return result, nil
}

func (mesh *TriangleMesh) Remesh(options RemeshOptions) (RemeshResult, error) {
	if options.TargetLength <= 0 || math.IsNaN(options.TargetLength) || math.IsInf(options.TargetLength, 0) {
		return RemeshResult{}, fmt.Errorf("target length must be greater than 0")
	}
	if options.MaxFactor <= 0 || math.IsNaN(options.MaxFactor) || math.IsInf(options.MaxFactor, 0) {
		return RemeshResult{}, fmt.Errorf("max factor must be greater than 0")
	}
	if options.Iterations < 1 {
		options.Iterations = 1
	}

	result := RemeshResult{
		InputVertices: len(mesh.Vertices),
		InputFaces:    len(mesh.Faces),
	}
	maxLength := options.TargetLength * options.MaxFactor

	for iteration := 0; iteration < options.Iterations; iteration++ {
		splits := mesh.splitLongEdges(maxLength)
		if splits == 0 {
			break
		}
		result.Splits += splits
		result.Iterations++
	}

	result.OutputVertices = len(mesh.Vertices)
	result.OutputFaces = len(mesh.Faces)
	return result, nil
}

func (mesh *TriangleMesh) splitLongEdges(maxLength float64) int {
	longEdges := mesh.longEdges(maxLength)
	if len(longEdges) == 0 {
		return 0
	}

	midpoints := map[edgeKey]int{}
	faces := make([][3]int, 0, len(mesh.Faces))

	for _, face := range mesh.Faces {
		faces = append(faces, mesh.splitFace(face, longEdges, midpoints)...)
	}

	mesh.Faces = faces
	return len(longEdges)
}

func (mesh *TriangleMesh) longEdges(maxLength float64) map[edgeKey]bool {
	edges := map[edgeKey]bool{}
	for _, face := range mesh.Faces {
		for _, edge := range [3]edgeKey{
			newEdgeKey(face[0], face[1]),
			newEdgeKey(face[1], face[2]),
			newEdgeKey(face[2], face[0]),
		} {
			length := mesh.Vertices[edge.A].Sub(mesh.Vertices[edge.B]).Length()
			if length > maxLength {
				edges[edge] = true
			}
		}
	}
	return edges
}

func (mesh *TriangleMesh) splitFace(face [3]int, longEdges map[edgeKey]bool, midpoints map[edgeKey]int) [][3]int {
	a, b, c := face[0], face[1], face[2]
	splitAB := longEdges[newEdgeKey(a, b)]
	splitBC := longEdges[newEdgeKey(b, c)]
	splitCA := longEdges[newEdgeKey(c, a)]
	splitCount := boolCount(splitAB, splitBC, splitCA)
	if splitCount == 0 {
		return [][3]int{face}
	}

	midpointAB := -1
	midpointBC := -1
	midpointCA := -1
	if splitAB {
		midpointAB = mesh.midpointVertex(a, b, midpoints)
	}
	if splitBC {
		midpointBC = mesh.midpointVertex(b, c, midpoints)
	}
	if splitCA {
		midpointCA = mesh.midpointVertex(c, a, midpoints)
	}

	switch {
	case splitAB && !splitBC && !splitCA:
		return [][3]int{{a, midpointAB, c}, {midpointAB, b, c}}
	case !splitAB && splitBC && !splitCA:
		return [][3]int{{a, b, midpointBC}, {a, midpointBC, c}}
	case !splitAB && !splitBC && splitCA:
		return [][3]int{{a, b, midpointCA}, {b, c, midpointCA}}
	case splitAB && splitBC && !splitCA:
		return [][3]int{{a, midpointAB, c}, {midpointAB, midpointBC, c}, {midpointAB, b, midpointBC}}
	case !splitAB && splitBC && splitCA:
		return [][3]int{{a, b, midpointCA}, {b, midpointBC, midpointCA}, {midpointBC, c, midpointCA}}
	case splitAB && !splitBC && splitCA:
		return [][3]int{{a, midpointAB, midpointCA}, {midpointAB, b, midpointCA}, {b, c, midpointCA}}
	default:
		return [][3]int{
			{a, midpointAB, midpointCA},
			{midpointAB, b, midpointBC},
			{midpointBC, c, midpointCA},
			{midpointAB, midpointBC, midpointCA},
		}
	}
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func (mesh *TriangleMesh) midpointVertex(a int, b int, midpoints map[edgeKey]int) int {
	key := newEdgeKey(a, b)
	if index, ok := midpoints[key]; ok {
		return index
	}

	midpoint := mesh.Vertices[a].Add(mesh.Vertices[b]).MulScalar(0.5)
	index := len(mesh.Vertices)
	mesh.Vertices = append(mesh.Vertices, midpoint)
	midpoints[key] = index
	return index
}

func (mesh *TriangleMesh) uniqueEdgeLengths() []float64 {
	edges := map[edgeKey]bool{}
	for _, face := range mesh.Faces {
		edges[newEdgeKey(face[0], face[1])] = true
		edges[newEdgeKey(face[1], face[2])] = true
		edges[newEdgeKey(face[2], face[0])] = true
	}

	lengths := make([]float64, 0, len(edges))
	for edge := range edges {
		lengths = append(lengths, mesh.Vertices[edge.A].Sub(mesh.Vertices[edge.B]).Length())
	}
	return lengths
}

func sortFloat64s(values []float64) {
	for index := 1; index < len(values); index++ {
		value := values[index]
		position := index - 1
		for position >= 0 && values[position] > value {
			values[position+1] = values[position]
			position--
		}
		values[position+1] = value
	}
}

func percentileSorted(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}
	position := percentile * float64(len(values)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return values[lower]
	}
	weight := position - float64(lower)
	return values[lower] + (values[upper]-values[lower])*weight
}

func (mesh *TriangleMesh) records() []Record {
	records := make([]Record, 0, len(mesh.Vertices)+len(mesh.Faces))
	for _, position := range mesh.Vertices {
		vertex := VertexRecord{Position: position}
		records = append(records, Record{Kind: "v", Vertex: &vertex})
	}
	for _, face := range mesh.Faces {
		refs := []string{
			strconv.Itoa(face[0] + 1),
			strconv.Itoa(face[1] + 1),
			strconv.Itoa(face[2] + 1),
		}
		records = append(records, Record{Kind: "f", Face: &FaceRecord{Refs: refs}})
	}
	return records
}
