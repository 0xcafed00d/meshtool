package obj

type TriangulateResult struct {
	Polygons     int
	NewTriangles int
}

func (doc *Document) Triangulate() TriangulateResult {
	result := TriangulateResult{}
	records := make([]Record, 0, len(doc.Records))

	for _, record := range doc.Records {
		if record.Face == nil || len(record.Face.Refs) <= 3 {
			records = append(records, record)
			continue
		}

		result.Polygons++
		refs := record.Face.Refs
		for index := 1; index < len(refs)-1; index++ {
			triangle := record
			triangle.Raw = ""
			triangle.Face = &FaceRecord{Refs: []string{refs[0], refs[index], refs[index+1]}}
			if index > 1 {
				triangle.InlineComment = ""
			}
			records = append(records, triangle)
			result.NewTriangles++
		}
	}

	doc.Records = records
	return result
}
