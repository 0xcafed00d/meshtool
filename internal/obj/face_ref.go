package obj

import (
	"fmt"
	"strconv"
	"strings"
)

type FaceRef struct {
	V     int
	VT    int
	VN    int
	HasVT bool
	HasVN bool
}

func ParseFaceRef(raw string) (FaceRef, error) {
	parts := strings.Split(raw, "/")
	if len(parts) == 0 || len(parts) > 3 || parts[0] == "" {
		return FaceRef{}, fmt.Errorf("invalid face reference %q", raw)
	}

	vertex, err := strconv.Atoi(parts[0])
	if err != nil {
		return FaceRef{}, fmt.Errorf("invalid vertex index %q", parts[0])
	}
	ref := FaceRef{V: vertex}

	if len(parts) >= 2 && parts[1] != "" {
		texCoord, err := strconv.Atoi(parts[1])
		if err != nil {
			return FaceRef{}, fmt.Errorf("invalid texture coordinate index %q", parts[1])
		}
		ref.VT = texCoord
		ref.HasVT = true
	}
	if len(parts) == 3 && parts[2] != "" {
		normal, err := strconv.Atoi(parts[2])
		if err != nil {
			return FaceRef{}, fmt.Errorf("invalid normal index %q", parts[2])
		}
		ref.VN = normal
		ref.HasVN = true
	}

	return ref, nil
}

func (ref FaceRef) String() string {
	switch {
	case ref.HasVT && ref.HasVN:
		return fmt.Sprintf("%d/%d/%d", ref.V, ref.VT, ref.VN)
	case ref.HasVT:
		return fmt.Sprintf("%d/%d", ref.V, ref.VT)
	case ref.HasVN:
		return fmt.Sprintf("%d//%d", ref.V, ref.VN)
	default:
		return strconv.Itoa(ref.V)
	}
}

func (ref FaceRef) Resolve(vertexCount int, texCoordCount int, normalCount int) (FaceRef, error) {
	resolvedVertex, err := resolveOBJIndex(ref.V, vertexCount, "vertex")
	if err != nil {
		return FaceRef{}, err
	}
	resolved := FaceRef{V: resolvedVertex, HasVT: ref.HasVT, HasVN: ref.HasVN}

	if ref.HasVT {
		resolvedTexCoord, err := resolveOBJIndex(ref.VT, texCoordCount, "texture coordinate")
		if err != nil {
			return FaceRef{}, err
		}
		resolved.VT = resolvedTexCoord
	}
	if ref.HasVN {
		resolvedNormal, err := resolveOBJIndex(ref.VN, normalCount, "normal")
		if err != nil {
			return FaceRef{}, err
		}
		resolved.VN = resolvedNormal
	}

	return resolved, nil
}

func resolveOBJIndex(index int, count int, label string) (int, error) {
	if index > 0 {
		if index > count {
			return 0, fmt.Errorf("%s index %d is out of range 1..%d", label, index, count)
		}
		return index, nil
	}
	if index < 0 {
		resolved := count + index + 1
		if resolved < 1 || resolved > count {
			return 0, fmt.Errorf("%s index %d resolves outside 1..%d", label, index, count)
		}
		return resolved, nil
	}
	return 0, fmt.Errorf("%s index must not be 0", label)
}
