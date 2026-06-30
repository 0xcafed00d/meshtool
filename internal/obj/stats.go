package obj

type Bounds struct {
	Min   Vec3
	Max   Vec3
	Empty bool
}

type Stats struct {
	Records      int
	Vertices     int
	TexCoords    int
	Normals      int
	Faces        int
	Triangles    int
	Quads        int
	NGons        int
	FaceVertsMin int
	FaceVertsMax int
	Bounds       Bounds
}

func (doc *Document) Stats() Stats {
	stats := Stats{
		Records:      len(doc.Records),
		FaceVertsMin: 0,
		FaceVertsMax: 0,
		Bounds:       Bounds{Empty: true},
	}

	for _, record := range doc.Records {
		switch {
		case record.Vertex != nil:
			stats.Vertices++
			stats.Bounds.Expand(record.Vertex.Position)
		case record.Normal != nil:
			stats.Normals++
		case record.TexCoord != nil:
			stats.TexCoords++
		case record.Face != nil:
			stats.Faces++
			vertexCount := len(record.Face.Refs)
			if stats.FaceVertsMin == 0 || vertexCount < stats.FaceVertsMin {
				stats.FaceVertsMin = vertexCount
			}
			if vertexCount > stats.FaceVertsMax {
				stats.FaceVertsMax = vertexCount
			}
			switch vertexCount {
			case 3:
				stats.Triangles++
			case 4:
				stats.Quads++
			default:
				stats.NGons++
			}
		}
	}

	return stats
}

func (bounds *Bounds) Expand(point Vec3) {
	if bounds.Empty {
		bounds.Min = point
		bounds.Max = point
		bounds.Empty = false
		return
	}
	if point.X < bounds.Min.X {
		bounds.Min.X = point.X
	}
	if point.Y < bounds.Min.Y {
		bounds.Min.Y = point.Y
	}
	if point.Z < bounds.Min.Z {
		bounds.Min.Z = point.Z
	}
	if point.X > bounds.Max.X {
		bounds.Max.X = point.X
	}
	if point.Y > bounds.Max.Y {
		bounds.Max.Y = point.Y
	}
	if point.Z > bounds.Max.Z {
		bounds.Max.Z = point.Z
	}
}

func (bounds Bounds) Size() Vec3 {
	if bounds.Empty {
		return Vec3{}
	}
	return bounds.Max.Sub(bounds.Min)
}

func (bounds Bounds) Center() Vec3 {
	if bounds.Empty {
		return Vec3{}
	}
	return bounds.Min.Add(bounds.Max).MulScalar(0.5)
}
