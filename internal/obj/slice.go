package obj

import (
	"fmt"
	"math"
	"strings"
)

type SliceOptions struct {
	Axis         int
	KeepPositive bool
	Boundary     float64
	Epsilon      float64
}

type SliceResult struct {
	InputFaces     int
	OutputFaces    int
	DiscardedFaces int
	SplitFaces     int
	NewVertices    int
	NewTexCoords   int
	NewNormals     int
}

type indexedFace struct {
	recordIndex   int
	record        Record
	vertexCount   int
	texCoordCount int
	normalCount   int
}

type logicalFace struct {
	refs          []FaceRef
	inlineComment string
}

type clipVertex struct {
	ref         FaceRef
	position    Vec3
	texCoord    []float64
	normal      Vec3
	hasTexCoord bool
	hasNormal   bool
}

type sliceBuilder struct {
	originalVertexCount   int
	originalTexCoordCount int
	originalNormalCount   int
	newVertices           []VertexRecord
	newTexCoords          []TexCoordRecord
	newNormals            []NormalRecord
	intersections         map[edgeKey]*edgeIntersection
}

type edgeKey struct {
	A int
	B int
}

type edgeAttributeKey struct {
	AVertex int
	BVertex int
	AIndex  int
	BIndex  int
}

type edgeIntersection struct {
	vertexIndex int
	texCoords   map[edgeAttributeKey]int
	normals     map[edgeAttributeKey]int
}

func ParseSliceSide(value string) (SliceOptions, error) {
	if len(value) != 2 {
		return SliceOptions{}, fmt.Errorf("slice side must be one of x-, x+, y-, y+, z-, z+")
	}

	options := SliceOptions{}
	switch strings.ToLower(value[:1]) {
	case "x":
		options.Axis = 0
	case "y":
		options.Axis = 1
	case "z":
		options.Axis = 2
	default:
		return SliceOptions{}, fmt.Errorf("slice side must be one of x-, x+, y-, y+, z-, z+")
	}

	switch value[1:] {
	case "+":
		options.KeepPositive = true
	case "-":
		options.KeepPositive = false
	default:
		return SliceOptions{}, fmt.Errorf("slice side must be one of x-, x+, y-, y+, z-, z+")
	}
	return options, nil
}

func (doc *Document) Slice(options SliceOptions) (SliceResult, error) {
	if options.Axis < 0 || options.Axis > 2 {
		return SliceResult{}, fmt.Errorf("slice axis must be 0, 1, or 2")
	}
	if options.Epsilon <= 0 {
		options.Epsilon = 1e-9
	}

	vertices, texCoords, normals, faces := doc.sliceInputs()
	builder := &sliceBuilder{
		originalVertexCount:   len(vertices),
		originalTexCoordCount: len(texCoords),
		originalNormalCount:   len(normals),
		intersections:         map[edgeKey]*edgeIntersection{},
	}

	result := SliceResult{InputFaces: len(faces)}
	facesByRecord := map[int][]logicalFace{}

	for _, face := range faces {
		clipped, split, err := doc.clipFace(face, vertices, texCoords, normals, builder, options)
		if err != nil {
			return SliceResult{}, err
		}
		if len(clipped) == 0 {
			result.DiscardedFaces++
			continue
		}
		if split {
			result.SplitFaces++
		}
		result.OutputFaces += len(clipped)
		facesByRecord[face.recordIndex] = clipped
	}

	result.NewVertices = len(builder.newVertices)
	result.NewTexCoords = len(builder.newTexCoords)
	result.NewNormals = len(builder.newNormals)

	insertionIndex := firstFaceIndex(doc.Records)
	newRecords := builder.records()
	vertexMap, texCoordMap, normalMap := buildSliceIndexMaps(doc.Records, insertionIndex, builder)

	records := make([]Record, 0, len(doc.Records)+len(newRecords)+result.OutputFaces-result.InputFaces)
	inserted := false
	for index, record := range doc.Records {
		if index == insertionIndex {
			records = append(records, newRecords...)
			inserted = true
		}
		if record.Face != nil {
			for _, face := range facesByRecord[index] {
				mapped, err := mapLogicalFace(face, vertexMap, texCoordMap, normalMap)
				if err != nil {
					return SliceResult{}, err
				}
				records = append(records, mapped)
			}
			continue
		}
		records = append(records, record)
	}
	if !inserted {
		records = append(records, newRecords...)
	}
	doc.Records = records
	if err := doc.pruneUnusedGeometry(); err != nil {
		return SliceResult{}, err
	}

	return result, nil
}

func (doc *Document) sliceInputs() ([]VertexRecord, []TexCoordRecord, []NormalRecord, []indexedFace) {
	var vertices []VertexRecord
	var texCoords []TexCoordRecord
	var normals []NormalRecord
	var faces []indexedFace

	for index, record := range doc.Records {
		if record.Vertex != nil {
			vertices = append(vertices, *record.Vertex)
		}
		if record.TexCoord != nil {
			texCoords = append(texCoords, *record.TexCoord)
		}
		if record.Normal != nil {
			normals = append(normals, *record.Normal)
		}
		if record.Face != nil {
			faces = append(faces, indexedFace{
				recordIndex:   index,
				record:        record,
				vertexCount:   len(vertices),
				texCoordCount: len(texCoords),
				normalCount:   len(normals),
			})
		}
	}

	return vertices, texCoords, normals, faces
}

func (doc *Document) clipFace(face indexedFace, vertices []VertexRecord, texCoords []TexCoordRecord, normals []NormalRecord, builder *sliceBuilder, options SliceOptions) ([]logicalFace, bool, error) {
	polygon := make([]clipVertex, 0, len(face.record.Face.Refs))
	hasInside := false
	hasOutside := false

	for _, rawRef := range face.record.Face.Refs {
		ref, err := ParseFaceRef(rawRef)
		if err != nil {
			return nil, false, err
		}
		resolved, err := ref.Resolve(face.vertexCount, face.texCoordCount, face.normalCount)
		if err != nil {
			return nil, false, err
		}
		vertex, err := clipVertexFromRef(resolved, vertices, texCoords, normals)
		if err != nil {
			return nil, false, err
		}
		distance := options.signedDistance(vertex.position)
		if distance >= -options.Epsilon {
			hasInside = true
		} else {
			hasOutside = true
		}
		polygon = append(polygon, vertex)
	}

	if !hasInside {
		return nil, false, nil
	}
	split := hasInside && hasOutside
	if split {
		polygon = clipPolygon(polygon, builder, options)
	}
	polygon = dedupeClipVertices(polygon, options.Epsilon)
	if len(polygon) < 3 {
		return nil, split, nil
	}

	refs := make([]FaceRef, len(polygon))
	for index, vertex := range polygon {
		refs[index] = vertex.ref
	}
	return []logicalFace{{
		refs:          refs,
		inlineComment: face.record.InlineComment,
	}}, split, nil
}

func clipVertexFromRef(ref FaceRef, vertices []VertexRecord, texCoords []TexCoordRecord, normals []NormalRecord) (clipVertex, error) {
	if ref.V < 1 || ref.V > len(vertices) {
		return clipVertex{}, fmt.Errorf("vertex index %d is out of range 1..%d", ref.V, len(vertices))
	}

	vertex := clipVertex{
		ref:      ref,
		position: vertices[ref.V-1].Position,
	}
	if ref.HasVT {
		if ref.VT < 1 || ref.VT > len(texCoords) {
			return clipVertex{}, fmt.Errorf("texture coordinate index %d is out of range 1..%d", ref.VT, len(texCoords))
		}
		vertex.texCoord = cloneFloat64s(texCoords[ref.VT-1].Values)
		vertex.hasTexCoord = true
	}
	if ref.HasVN {
		if ref.VN < 1 || ref.VN > len(normals) {
			return clipVertex{}, fmt.Errorf("normal index %d is out of range 1..%d", ref.VN, len(normals))
		}
		vertex.normal = normals[ref.VN-1].Direction
		vertex.hasNormal = true
	}
	return vertex, nil
}

func clipPolygon(vertices []clipVertex, builder *sliceBuilder, options SliceOptions) []clipVertex {
	if len(vertices) == 0 {
		return nil
	}

	output := make([]clipVertex, 0, len(vertices)+1)
	previous := vertices[len(vertices)-1]
	previousDistance := options.signedDistance(previous.position)
	previousInside := previousDistance >= -options.Epsilon

	for _, current := range vertices {
		currentDistance := options.signedDistance(current.position)
		currentInside := currentDistance >= -options.Epsilon

		switch {
		case previousInside && currentInside:
			output = append(output, current)
		case previousInside && !currentInside:
			output = append(output, intersectClipVertices(previous, current, previousDistance, currentDistance, builder, options.Epsilon))
		case !previousInside && currentInside:
			output = append(output, intersectClipVertices(previous, current, previousDistance, currentDistance, builder, options.Epsilon), current)
		}

		previous = current
		previousDistance = currentDistance
		previousInside = currentInside
	}

	return output
}

func intersectClipVertices(a clipVertex, b clipVertex, distanceA float64, distanceB float64, builder *sliceBuilder, epsilon float64) clipVertex {
	denominator := distanceA - distanceB
	if math.Abs(denominator) <= epsilon {
		return a
	}
	t := distanceA / denominator
	if t <= epsilon {
		return a
	}
	if t >= 1-epsilon {
		return b
	}

	position := lerpVec3(a.position, b.position, t)
	ref := FaceRef{V: builder.addIntersectionVertex(a.ref.V, b.ref.V, position)}
	vertex := clipVertex{ref: ref, position: position}

	if a.hasTexCoord && b.hasTexCoord && len(a.texCoord) == len(b.texCoord) {
		vertex.texCoord = lerpFloat64s(a.texCoord, b.texCoord, t)
		vertex.hasTexCoord = true
		vertex.ref.VT = builder.addIntersectionTexCoord(a, b, vertex.texCoord)
		vertex.ref.HasVT = true
	}
	if a.hasNormal && b.hasNormal {
		vertex.normal = lerpVec3(a.normal, b.normal, t).Normalize()
		vertex.hasNormal = true
		vertex.ref.VN = builder.addIntersectionNormal(a, b, vertex.normal)
		vertex.ref.HasVN = true
	}

	return vertex
}

func dedupeClipVertices(vertices []clipVertex, epsilon float64) []clipVertex {
	if len(vertices) < 2 {
		return vertices
	}

	deduped := make([]clipVertex, 0, len(vertices))
	for _, vertex := range vertices {
		if len(deduped) == 0 || !samePosition(deduped[len(deduped)-1].position, vertex.position, epsilon) {
			deduped = append(deduped, vertex)
		}
	}
	if len(deduped) > 1 && samePosition(deduped[0].position, deduped[len(deduped)-1].position, epsilon) {
		deduped = deduped[:len(deduped)-1]
	}
	return deduped
}

func (options SliceOptions) signedDistance(point Vec3) float64 {
	value := coordinate(point, options.Axis) - options.Boundary
	if options.KeepPositive {
		return value
	}
	return -value
}

func coordinate(point Vec3, axis int) float64 {
	switch axis {
	case 0:
		return point.X
	case 1:
		return point.Y
	default:
		return point.Z
	}
}

func (builder *sliceBuilder) addVertex(position Vec3) int {
	builder.newVertices = append(builder.newVertices, VertexRecord{Position: position})
	return builder.originalVertexCount + len(builder.newVertices)
}

func (builder *sliceBuilder) addIntersectionVertex(a int, b int, position Vec3) int {
	key := newEdgeKey(a, b)
	intersection, ok := builder.intersections[key]
	if ok {
		return intersection.vertexIndex
	}

	intersection = &edgeIntersection{
		vertexIndex: builder.addVertex(position),
		texCoords:   map[edgeAttributeKey]int{},
		normals:     map[edgeAttributeKey]int{},
	}
	builder.intersections[key] = intersection
	return intersection.vertexIndex
}

func (builder *sliceBuilder) addTexCoord(values []float64) int {
	builder.newTexCoords = append(builder.newTexCoords, TexCoordRecord{Values: cloneFloat64s(values)})
	return builder.originalTexCoordCount + len(builder.newTexCoords)
}

func (builder *sliceBuilder) addIntersectionTexCoord(a clipVertex, b clipVertex, values []float64) int {
	key := newEdgeKey(a.ref.V, b.ref.V)
	intersection := builder.intersections[key]
	attrKey := newEdgeAttributeKey(a.ref.V, a.ref.VT, b.ref.V, b.ref.VT)
	if index, ok := intersection.texCoords[attrKey]; ok {
		return index
	}

	index := builder.addTexCoord(values)
	intersection.texCoords[attrKey] = index
	return index
}

func (builder *sliceBuilder) addNormal(direction Vec3) int {
	builder.newNormals = append(builder.newNormals, NormalRecord{Direction: direction})
	return builder.originalNormalCount + len(builder.newNormals)
}

func (builder *sliceBuilder) addIntersectionNormal(a clipVertex, b clipVertex, direction Vec3) int {
	key := newEdgeKey(a.ref.V, b.ref.V)
	intersection := builder.intersections[key]
	attrKey := newEdgeAttributeKey(a.ref.V, a.ref.VN, b.ref.V, b.ref.VN)
	if index, ok := intersection.normals[attrKey]; ok {
		return index
	}

	index := builder.addNormal(direction)
	intersection.normals[attrKey] = index
	return index
}

func newEdgeKey(a int, b int) edgeKey {
	if a < b {
		return edgeKey{A: a, B: b}
	}
	return edgeKey{A: b, B: a}
}

func newEdgeAttributeKey(aVertex int, aIndex int, bVertex int, bIndex int) edgeAttributeKey {
	if aVertex < bVertex {
		return edgeAttributeKey{AVertex: aVertex, BVertex: bVertex, AIndex: aIndex, BIndex: bIndex}
	}
	return edgeAttributeKey{AVertex: bVertex, BVertex: aVertex, AIndex: bIndex, BIndex: aIndex}
}

func (builder *sliceBuilder) records() []Record {
	records := make([]Record, 0, len(builder.newVertices)+len(builder.newTexCoords)+len(builder.newNormals))
	for index := range builder.newVertices {
		vertex := builder.newVertices[index]
		records = append(records, Record{Kind: "v", Vertex: &vertex})
	}
	for index := range builder.newTexCoords {
		texCoord := builder.newTexCoords[index]
		records = append(records, Record{Kind: "vt", TexCoord: &texCoord})
	}
	for index := range builder.newNormals {
		normal := builder.newNormals[index]
		records = append(records, Record{Kind: "vn", Normal: &normal})
	}
	return records
}

func firstFaceIndex(records []Record) int {
	for index, record := range records {
		if record.Face != nil {
			return index
		}
	}
	return len(records)
}

func buildSliceIndexMaps(records []Record, insertionIndex int, builder *sliceBuilder) ([]int, []int, []int) {
	vertexMap := make([]int, builder.originalVertexCount+len(builder.newVertices)+1)
	texCoordMap := make([]int, builder.originalTexCoordCount+len(builder.newTexCoords)+1)
	normalMap := make([]int, builder.originalNormalCount+len(builder.newNormals)+1)

	originalVertex := 0
	originalTexCoord := 0
	originalNormal := 0
	outputVertex := 0
	outputTexCoord := 0
	outputNormal := 0
	inserted := false

	insertNew := func() {
		if inserted {
			return
		}
		for index := range builder.newVertices {
			outputVertex++
			vertexMap[builder.originalVertexCount+index+1] = outputVertex
		}
		for index := range builder.newTexCoords {
			outputTexCoord++
			texCoordMap[builder.originalTexCoordCount+index+1] = outputTexCoord
		}
		for index := range builder.newNormals {
			outputNormal++
			normalMap[builder.originalNormalCount+index+1] = outputNormal
		}
		inserted = true
	}

	for index, record := range records {
		if index == insertionIndex {
			insertNew()
		}
		if record.Vertex != nil {
			originalVertex++
			outputVertex++
			vertexMap[originalVertex] = outputVertex
		}
		if record.TexCoord != nil {
			originalTexCoord++
			outputTexCoord++
			texCoordMap[originalTexCoord] = outputTexCoord
		}
		if record.Normal != nil {
			originalNormal++
			outputNormal++
			normalMap[originalNormal] = outputNormal
		}
	}
	insertNew()

	return vertexMap, texCoordMap, normalMap
}

func mapLogicalFace(face logicalFace, vertexMap []int, texCoordMap []int, normalMap []int) (Record, error) {
	refs := make([]string, len(face.refs))
	for index, ref := range face.refs {
		mapped := FaceRef{HasVT: ref.HasVT, HasVN: ref.HasVN}
		if ref.V < 1 || ref.V >= len(vertexMap) || vertexMap[ref.V] == 0 {
			return Record{}, fmt.Errorf("vertex index %d has no output mapping", ref.V)
		}
		mapped.V = vertexMap[ref.V]
		if ref.HasVT {
			if ref.VT < 1 || ref.VT >= len(texCoordMap) || texCoordMap[ref.VT] == 0 {
				return Record{}, fmt.Errorf("texture coordinate index %d has no output mapping", ref.VT)
			}
			mapped.VT = texCoordMap[ref.VT]
		}
		if ref.HasVN {
			if ref.VN < 1 || ref.VN >= len(normalMap) || normalMap[ref.VN] == 0 {
				return Record{}, fmt.Errorf("normal index %d has no output mapping", ref.VN)
			}
			mapped.VN = normalMap[ref.VN]
		}
		refs[index] = mapped.String()
	}
	return Record{
		Kind:          "f",
		InlineComment: face.inlineComment,
		Face:          &FaceRecord{Refs: refs},
	}, nil
}

func lerpVec3(a Vec3, b Vec3, t float64) Vec3 {
	return Vec3{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
	}
}

func lerpFloat64s(a []float64, b []float64, t float64) []float64 {
	values := make([]float64, len(a))
	for index := range a {
		values[index] = a[index] + (b[index]-a[index])*t
	}
	return values
}

func samePosition(a Vec3, b Vec3, epsilon float64) bool {
	return math.Abs(a.X-b.X) <= epsilon && math.Abs(a.Y-b.Y) <= epsilon && math.Abs(a.Z-b.Z) <= epsilon
}

func cloneFloat64s(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}
	clone := make([]float64, len(values))
	copy(clone, values)
	return clone
}
