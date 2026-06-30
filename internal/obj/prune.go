package obj

import "fmt"

func (doc *Document) pruneUnusedGeometry() error {
	usedVertices := map[int]bool{}
	usedTexCoords := map[int]bool{}
	usedNormals := map[int]bool{}

	for _, record := range doc.Records {
		if record.Face == nil {
			continue
		}
		for _, rawRef := range record.Face.Refs {
			ref, err := ParseFaceRef(rawRef)
			if err != nil {
				return err
			}
			if ref.V <= 0 || ref.VT < 0 || ref.VN < 0 {
				return fmt.Errorf("cannot prune face reference %q; expected positive indices", rawRef)
			}
			usedVertices[ref.V] = true
			if ref.HasVT {
				usedTexCoords[ref.VT] = true
			}
			if ref.HasVN {
				usedNormals[ref.VN] = true
			}
		}
	}

	vertexMap := map[int]int{}
	texCoordMap := map[int]int{}
	normalMap := map[int]int{}
	oldVertex := 0
	oldTexCoord := 0
	oldNormal := 0
	newVertex := 0
	newTexCoord := 0
	newNormal := 0

	for _, record := range doc.Records {
		if record.Vertex != nil {
			oldVertex++
			if usedVertices[oldVertex] {
				newVertex++
				vertexMap[oldVertex] = newVertex
			}
		}
		if record.TexCoord != nil {
			oldTexCoord++
			if usedTexCoords[oldTexCoord] {
				newTexCoord++
				texCoordMap[oldTexCoord] = newTexCoord
			}
		}
		if record.Normal != nil {
			oldNormal++
			if usedNormals[oldNormal] {
				newNormal++
				normalMap[oldNormal] = newNormal
			}
		}
	}

	oldVertex = 0
	oldTexCoord = 0
	oldNormal = 0
	records := make([]Record, 0, len(doc.Records))
	for _, record := range doc.Records {
		switch {
		case record.Vertex != nil:
			oldVertex++
			if usedVertices[oldVertex] {
				records = append(records, record)
			}
		case record.TexCoord != nil:
			oldTexCoord++
			if usedTexCoords[oldTexCoord] {
				records = append(records, record)
			}
		case record.Normal != nil:
			oldNormal++
			if usedNormals[oldNormal] {
				records = append(records, record)
			}
		case record.Face != nil:
			mappedRefs := make([]string, len(record.Face.Refs))
			for index, rawRef := range record.Face.Refs {
				ref, err := ParseFaceRef(rawRef)
				if err != nil {
					return err
				}
				mapped := FaceRef{HasVT: ref.HasVT, HasVN: ref.HasVN}
				var ok bool
				mapped.V, ok = vertexMap[ref.V]
				if !ok {
					return fmt.Errorf("face references pruned vertex index %d", ref.V)
				}
				if ref.HasVT {
					mapped.VT, ok = texCoordMap[ref.VT]
					if !ok {
						return fmt.Errorf("face references pruned texture coordinate index %d", ref.VT)
					}
				}
				if ref.HasVN {
					mapped.VN, ok = normalMap[ref.VN]
					if !ok {
						return fmt.Errorf("face references pruned normal index %d", ref.VN)
					}
				}
				mappedRefs[index] = mapped.String()
			}
			record.Face = &FaceRecord{Refs: mappedRefs}
			records = append(records, record)
		default:
			records = append(records, record)
		}
	}

	doc.Records = records
	return nil
}
