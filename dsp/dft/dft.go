package dft

import (
	"math"
	"math/cmplx"
)

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

// Implements a 14-point complex-to-complex DFT.
func dft14(xi, xo []complex128) {
	T1 := xi[0]
	T2 := xi[7]
	T3 := T1 - T2
	T34 := T1 + T2
	T18 := xi[6]
	T19 := xi[13]
	T20 := T18 - T19
	T38 := T18 + T19
	T21 := xi[8]
	T22 := xi[1]
	T23 := T21 - T22
	T39 := T21 + T22
	T24 := T20 + T23
	T46 := T38 - T39
	T27 := T23 - T20
	T40 := T38 + T39
	T4 := xi[2]
	T5 := xi[9]
	T6 := T4 - T5
	T35 := T4 + T5
	T7 := xi[12]
	T8 := xi[5]
	T9 := T7 - T8
	T36 := T7 + T8
	T10 := T6 + T9
	T45 := T36 - T35
	T26 := T9 - T6
	T37 := T35 + T36
	T11 := xi[4]
	T12 := xi[11]
	T13 := T11 - T12
	T41 := T11 + T12
	T14 := xi[10]
	T15 := xi[3]
	T16 := T14 - T15
	T42 := T14 + T15
	T17 := T13 + T16
	T47 := T41 - T42
	T28 := T16 - T13
	T43 := T41 + T42
	xo[7] = T3 + T10 + T17 + T24
	xo[0] = T34 + T37 + T43 + T40
	T29 := I * ((KP974927912 * T26) - (KP781831482 * T27) - (KP433883739 * T28))
	T25 := T3 + (KP623489801 * T24) - (KP900968867 * T17) - (KP222520933 * T10)
	xo[5] = T25 - T29
	xo[9] = T25 + T29
	T51 := I * ((KP974927912 * T45) + (KP433883739 * T47) + (KP781831482 * T46))
	T52 := T34 + (KP623489801 * T40) - (KP900968867 * T43) - (KP222520933 * T37)
	xo[2] = T51 + T52
	xo[12] = T52 - T51
	T31 := I * ((KP781831482 * T26) + (KP974927912 * T28) + (KP433883739 * T27))
	T30 := T3 + (KP623489801 * T10) - (KP900968867 * T24) - (KP222520933 * T17)
	xo[13] = T30 - T31
	xo[1] = T30 + T31
	T48 := I * ((KP781831482 * T45) - (KP433883739 * T46) - (KP974927912 * T47))
	T44 := T34 + (KP623489801 * T37) - (KP900968867 * T40) - (KP222520933 * T43)
	xo[6] = T44 - T48
	xo[8] = T48 + T44
	T50 := I * ((KP433883739 * T45) + (KP781831482 * T47) - (KP974927912 * T46))
	T49 := T34 + (KP623489801 * T43) - (KP222520933 * T40) - (KP900968867 * T37)
	xo[4] = T49 - T50
	xo[10] = T50 + T49
	T33 := I * ((KP433883739 * T26) + (KP974927912 * T27) - (KP781831482 * T28))
	T32 := T3 + (KP623489801 * T17) - (KP222520933 * T24) - (KP900968867 * T10)
	xo[11] = T32 - T33
	xo[3] = T32 + T33
}

func naiveSDFT14(in []complex128, out [][]complex128) {
	for idx := range in[:len(in)-N] {
		dft14(in[idx:], out[idx])
	}
}

// A generic Sliding-DFT for testing purposes.
type slidingDft struct {
	n      int
	coeffs []complex128
	hx     []complex128

	first bool

	delta complex128

	h0, h1, h2, h3, h4, h5, h6, h7, h8, h9, h10, h11, h12, h13 complex128
}

<<<<<<< HEAD
func newSlidingDft(n int) (sdft slidingDft) {
	sdft.n = n
	sdft.coeffs = make([]complex128, n)
	sdft.first = false

	sdft.hx = make([]complex128, n)

	for idx := range sdft.coeffs {
		sdft.coeffs[idx] = cmplx.Exp(complex(0, 2.0*math.Pi*float64(idx)/float64(n)))
	}

	return
}

func (sdft *slidingDft) Execute(in []complex128, out [][]complex128) {
	for j := range out[0] {
		out[0][j] = (sdft.hx[j] + sdft.delta) * sdft.coeffs[j]
	}

	for i := 1; i < len(in)-sdft.n+1; i++ {
		delta := in[i+sdft.n-1] - in[i-1]

		for j := range out[i] {
			out[i][j] = (out[i-1][j] + delta) * sdft.coeffs[j]
		}
	}
	sdft.delta = in[len(in)-1] - in[len(in)-sdft.n-1]
	copy(sdft.hx, out[len(out)-1])
}

func (sdft *slidingDft) ExecuteUnroll(in []complex128, out [][]complex128) {
	h0 := (sdft.h0 + sdft.delta)
	h1 := (sdft.h1 + sdft.delta) * C1
	h2 := (sdft.h2 + sdft.delta) * C2
	h3 := (sdft.h3 + sdft.delta) * C3
	h4 := (sdft.h4 + sdft.delta) * C4
	h5 := (sdft.h5 + sdft.delta) * C5
	h6 := (sdft.h6 + sdft.delta) * C6
	h7 := -(sdft.h7 + sdft.delta)
	h8 := -(sdft.h8 + sdft.delta) * C1
	h9 := -(sdft.h9 + sdft.delta) * C2
	h10 := -(sdft.h10 + sdft.delta) * C3
	h11 := -(sdft.h11 + sdft.delta) * C4
	h12 := -(sdft.h12 + sdft.delta) * C5
	h13 := -(sdft.h13 + sdft.delta) * C6

	outWin := out[0]
	outWin[0] = h0
	outWin[1] = h1
	outWin[2] = h2
	outWin[3] = h3
	outWin[4] = h4
	outWin[5] = h5
	outWin[6] = h6
	outWin[7] = h7
	outWin[8] = h8
	outWin[9] = h9
	outWin[10] = h10
	outWin[11] = h11
	outWin[12] = h12
	outWin[13] = h13

	for idx := 1; idx < len(in)-N+1; idx++ {
		delta := in[idx+N-1] - in[idx-1]

<<<<<<< HEAD
		h0 = h0 + delta
		h1 = (h1 + delta) * C1
		h2 = (h2 + delta) * C2
		h3 = (h3 + delta) * C3
		h4 = (h4 + delta) * C4
		h5 = (h5 + delta) * C5
		h6 = (h6 + delta) * C6
		h7 = -(h7 + delta)
		h8 = -(h8 + delta) * C1
		h9 = -(h9 + delta) * C2
		h10 = -(h10 + delta) * C3
		h11 = -(h11 + delta) * C4
		h12 = -(h12 + delta) * C5
		h13 = -(h13 + delta) * C6

		outWin := out[idx]
		outWin[0] = h0
		outWin[1] = h1
		outWin[2] = h2
		outWin[3] = h3
		outWin[4] = h4
		outWin[5] = h5
		outWin[6] = h6
		outWin[7] = h7
		outWin[8] = h8
		outWin[9] = h9
		outWin[10] = h10
		outWin[11] = h11
		outWin[12] = h12
		outWin[13] = h13
	}

	sdft.h0 = h0
	sdft.h1 = h1
	sdft.h2 = h2
	sdft.h3 = h3
	sdft.h4 = h4
	sdft.h5 = h5
	sdft.h6 = h6
	sdft.h7 = h7
	sdft.h8 = h8
	sdft.h9 = h9
	sdft.h10 = h10
	sdft.h11 = h11
	sdft.h12 = h12
	sdft.h13 = h13
}
