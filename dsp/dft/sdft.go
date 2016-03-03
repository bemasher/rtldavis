package dft

// KP-prefixed constants are defined in dft.go
const (
	C1 = +KP900968867 + KP433883739*1i
	C2 = +KP623489801 + KP781831482*1i
	C3 = +KP222520933 + KP974927912*1i
	C4 = -KP222520933 + KP974927912*1i
	C5 = -KP623489801 + KP781831482*1i
	C6 = -KP900968867 + KP433883739*1i
)

type SDFT struct {
	delta    complex128
	h10, h11 complex128
}

func (sdft *SDFT) Demod(in []complex128, out []float64) {
	h10 := -(sdft.h10 + sdft.delta) * C3
	h11 := -(sdft.h11 + sdft.delta) * C4
	// out[0] = magDiff(h10, h11)
	out[0] = magDiff(h10, h11) / (real(in[0])*real(in[0]) + imag(in[0])*imag(in[0]))

	for idx := 1; idx < len(in)-N; idx++ {
		delta := in[idx+N-1] - in[idx-1]

		h10 = -(h10 + delta) * C3
		h11 = -(h11 + delta) * C4

		// out[idx] = magDiff(h10, h11)
		out[idx] = magDiff(h10, h11) / (real(in[idx])*real(in[idx]) + imag(in[idx])*imag(in[idx]))
	}

	sdft.delta = in[len(in)-1] - in[len(in)-N-1]
	sdft.h10 = h10
	sdft.h11 = h11
}

func magDiff(i, j complex128) float64 {
	return (real(i)*real(i) + imag(i)*imag(i)) - (real(j)*real(j) + imag(j)*imag(j))
}
