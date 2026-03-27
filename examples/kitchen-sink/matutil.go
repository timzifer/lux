package main

import "math"

// ── Matrix math (column-major, shared between OpenGL and WGPU cube) ──

func perspectiveMatrix(fovY, aspect, near, far float32) [16]float32 {
	f := float32(1.0 / math.Tan(float64(fovY/2)))
	nf := near - far
	return [16]float32{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / nf, -1,
		0, 0, (2 * far * near) / nf, 0,
	}
}

func translationMatrix(x, y, z float32) [16]float32 {
	return [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		x, y, z, 1,
	}
}

func rotationX(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	return [16]float32{
		1, 0, 0, 0,
		0, c, s, 0,
		0, -s, c, 0,
		0, 0, 0, 1,
	}
}

func rotationY(a float32) [16]float32 {
	c := float32(math.Cos(float64(a)))
	s := float32(math.Sin(float64(a)))
	return [16]float32{
		c, 0, -s, 0,
		0, 1, 0, 0,
		s, 0, c, 0,
		0, 0, 0, 1,
	}
}

func matMul4(a, b [16]float32) [16]float32 {
	// Column-major multiplication: R = A * B
	// Storage: mat[col*4 + row], so R[col_i, row_j] = Σ_k A[col_k, row_j] * B[col_i, row_k]
	var r [16]float32
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				r[i*4+j] += a[k*4+j] * b[i*4+k]
			}
		}
	}
	return r
}
