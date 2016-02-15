package dft

import "math"

const N = 14

const (
	I           = 1i
	KP222520933 = +0.222520933956314404288902564496794759466355569
	KP900968867 = +0.900968867902419126236102319507445051165919162
	KP623489801 = +0.623489801858733530525004884004239810632274731
	KP433883739 = +0.433883739117558120475768332848358754609990728
	KP781831482 = +0.781831482468029808708444526674057750232334519
	KP974927912 = +0.974927912181823607018131682993931217232785801
)

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
	out[0] = MagDiff(h10, h11)

	for idx := 1; idx < len(in)-N; idx++ {
		delta := in[idx+N-1] - in[idx-1]

		h10 = -(h10 + delta) * C3
		h11 = -(h11 + delta) * C4

		out[idx] = MagDiff(h10, h11)
	}

	sdft.delta = in[len(in)-1] - in[len(in)-N-1]
	sdft.h10 = h10
	sdft.h11 = h11
}

func MagDiff(i, j complex128) float64 {
	return math.Sqrt(real(i)*real(i)+imag(i)*imag(i)) - math.Sqrt(real(j)*real(j)+imag(j)*imag(j))
}
