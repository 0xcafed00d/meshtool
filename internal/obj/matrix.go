package obj

import (
	"errors"
	"math"
)

type Mat3 [3][3]float64

type Mat4 [4][4]float64

func IdentityMat3() Mat3 {
	return Mat3{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
	}
}

func IdentityMat4() Mat4 {
	return Mat4{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

func ScaleMat3(scale Vec3) Mat3 {
	return Mat3{
		{scale.X, 0, 0},
		{0, scale.Y, 0},
		{0, 0, scale.Z},
	}
}

func RotateXMat3(degrees float64) Mat3 {
	radians := degrees * math.Pi / 180
	cosine := math.Cos(radians)
	sine := math.Sin(radians)
	return Mat3{
		{1, 0, 0},
		{0, cosine, -sine},
		{0, sine, cosine},
	}
}

func RotateYMat3(degrees float64) Mat3 {
	radians := degrees * math.Pi / 180
	cosine := math.Cos(radians)
	sine := math.Sin(radians)
	return Mat3{
		{cosine, 0, sine},
		{0, 1, 0},
		{-sine, 0, cosine},
	}
}

func RotateZMat3(degrees float64) Mat3 {
	radians := degrees * math.Pi / 180
	cosine := math.Cos(radians)
	sine := math.Sin(radians)
	return Mat3{
		{cosine, -sine, 0},
		{sine, cosine, 0},
		{0, 0, 1},
	}
}

func (m Mat3) Mul(other Mat3) Mat3 {
	var result Mat3
	for row := range 3 {
		for col := range 3 {
			result[row][col] = m[row][0]*other[0][col] + m[row][1]*other[1][col] + m[row][2]*other[2][col]
		}
	}
	return result
}

func (m Mat3) MulVec(v Vec3) Vec3 {
	return Vec3{
		X: m[0][0]*v.X + m[0][1]*v.Y + m[0][2]*v.Z,
		Y: m[1][0]*v.X + m[1][1]*v.Y + m[1][2]*v.Z,
		Z: m[2][0]*v.X + m[2][1]*v.Y + m[2][2]*v.Z,
	}
}

func (m Mat3) Determinant() float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}

func (m Mat3) Transpose() Mat3 {
	return Mat3{
		{m[0][0], m[1][0], m[2][0]},
		{m[0][1], m[1][1], m[2][1]},
		{m[0][2], m[1][2], m[2][2]},
	}
}

func (m Mat3) Inverse() (Mat3, error) {
	determinant := m.Determinant()
	if math.Abs(determinant) < 1e-12 {
		return Mat3{}, errors.New("transform matrix is singular")
	}

	invDet := 1 / determinant
	return Mat3{
		{
			(m[1][1]*m[2][2] - m[1][2]*m[2][1]) * invDet,
			(m[0][2]*m[2][1] - m[0][1]*m[2][2]) * invDet,
			(m[0][1]*m[1][2] - m[0][2]*m[1][1]) * invDet,
		},
		{
			(m[1][2]*m[2][0] - m[1][0]*m[2][2]) * invDet,
			(m[0][0]*m[2][2] - m[0][2]*m[2][0]) * invDet,
			(m[0][2]*m[1][0] - m[0][0]*m[1][2]) * invDet,
		},
		{
			(m[1][0]*m[2][1] - m[1][1]*m[2][0]) * invDet,
			(m[0][1]*m[2][0] - m[0][0]*m[2][1]) * invDet,
			(m[0][0]*m[1][1] - m[0][1]*m[1][0]) * invDet,
		},
	}, nil
}

func (m Mat4) AffineParts() (Mat3, Vec3, error) {
	if math.Abs(m[3][0]) > 1e-12 || math.Abs(m[3][1]) > 1e-12 || math.Abs(m[3][2]) > 1e-12 || math.Abs(m[3][3]-1) > 1e-12 {
		return Mat3{}, Vec3{}, errors.New("4x4 transform matrix must be affine; bottom row must be 0 0 0 1")
	}

	linear := Mat3{
		{m[0][0], m[0][1], m[0][2]},
		{m[1][0], m[1][1], m[1][2]},
		{m[2][0], m[2][1], m[2][2]},
	}
	translation := Vec3{X: m[0][3], Y: m[1][3], Z: m[2][3]}
	return linear, translation, nil
}
