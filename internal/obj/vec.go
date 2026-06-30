package obj

import "math"

// Vec3 is a 3D vector used for positions, normals, bounds, and transforms.
type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

func (v Vec3) MulScalar(scale float64) Vec3 {
	return Vec3{X: v.X * scale, Y: v.Y * scale, Z: v.Z * scale}
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func (v Vec3) Normalize() Vec3 {
	length := v.Length()
	if length == 0 {
		return v
	}
	return v.MulScalar(1 / length)
}

func (v Vec3) MaxComponent() float64 {
	return math.Max(v.X, math.Max(v.Y, v.Z))
}

func cleanFloat(value float64) float64 {
	if math.Abs(value) < 1e-12 {
		return 0
	}
	return value
}
