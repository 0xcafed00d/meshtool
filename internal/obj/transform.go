package obj

import "fmt"

type TransformOptions struct {
	Center         bool
	NormalizeTo    float64
	Scale          Vec3
	Translate      Vec3
	RotateDegrees  Vec3
	Matrix         *Mat4
	FlipX          bool
	FlipY          bool
	FlipZ          bool
	ReverseWinding bool
}

func DefaultTransformOptions() TransformOptions {
	return TransformOptions{Scale: Vec3{X: 1, Y: 1, Z: 1}}
}

func (doc *Document) Transform(options TransformOptions) error {
	linear := IdentityMat3()
	translation := Vec3{}

	apply := func(matrix Mat3, offset Vec3) {
		translation = matrix.MulVec(translation).Add(offset)
		linear = matrix.Mul(linear)
	}

	bounds := doc.Stats().Bounds
	if options.NormalizeTo != 0 {
		if options.NormalizeTo < 0 {
			return fmt.Errorf("normalize target must be positive")
		}
		if bounds.Empty {
			return fmt.Errorf("cannot normalize an OBJ with no vertices")
		}
		size := bounds.Size()
		maxDimension := size.MaxComponent()
		if maxDimension == 0 {
			return fmt.Errorf("cannot normalize an OBJ with zero-size bounds")
		}
		apply(IdentityMat3(), bounds.Center().MulScalar(-1))
		apply(ScaleMat3(Vec3{X: options.NormalizeTo / maxDimension, Y: options.NormalizeTo / maxDimension, Z: options.NormalizeTo / maxDimension}), Vec3{})
	} else if options.Center {
		if bounds.Empty {
			return fmt.Errorf("cannot center an OBJ with no vertices")
		}
		apply(IdentityMat3(), bounds.Center().MulScalar(-1))
	}

	flipScale := Vec3{X: 1, Y: 1, Z: 1}
	if options.FlipX {
		flipScale.X = -1
	}
	if options.FlipY {
		flipScale.Y = -1
	}
	if options.FlipZ {
		flipScale.Z = -1
	}
	apply(ScaleMat3(flipScale), Vec3{})
	apply(ScaleMat3(options.Scale), Vec3{})

	if options.RotateDegrees.X != 0 {
		apply(RotateXMat3(options.RotateDegrees.X), Vec3{})
	}
	if options.RotateDegrees.Y != 0 {
		apply(RotateYMat3(options.RotateDegrees.Y), Vec3{})
	}
	if options.RotateDegrees.Z != 0 {
		apply(RotateZMat3(options.RotateDegrees.Z), Vec3{})
	}
	apply(IdentityMat3(), options.Translate)

	if options.Matrix != nil {
		matrix, offset, err := options.Matrix.AffineParts()
		if err != nil {
			return err
		}
		apply(matrix, offset)
	}

	normalMatrix, err := normalMatrix(linear)
	if err != nil {
		return err
	}

	reverseWinding := linear.Determinant() < 0
	if options.ReverseWinding {
		reverseWinding = !reverseWinding
	}

	for index := range doc.Records {
		record := &doc.Records[index]
		if record.Vertex != nil {
			record.Vertex.Position = linear.MulVec(record.Vertex.Position).Add(translation)
		}
		if record.Normal != nil {
			record.Normal.Direction = normalMatrix.MulVec(record.Normal.Direction).Normalize()
		}
		if reverseWinding && record.Face != nil {
			reverseStrings(record.Face.Refs)
		}
	}

	return nil
}

func normalMatrix(linear Mat3) (Mat3, error) {
	inverse, err := linear.Inverse()
	if err != nil {
		return Mat3{}, fmt.Errorf("cannot transform normals: %w", err)
	}
	return inverse.Transpose(), nil
}

func reverseStrings(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
