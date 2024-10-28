package types

import (
	"errors"
)

type BN128 struct {
	Fr Field
	Tm *TM
}

type Field struct{}

func (f Field) E(val interface{}) []byte {
	// Implement Field encoding logic here
	return []byte{}
}

func (f Field) Zero() []byte {
	return make([]byte, 32) // Assuming 32 bytes for zero value
}

type TM struct {
	Instance Instance
}

func (tm *TM) Alloc(size int) []byte {
	return make([]byte, size)
}

func (tm *TM) SetBuff(p []byte, buff []byte) {
	copy(p, buff)
}

func (tm *TM) GetBuff(p []byte, size int) []byte {
	return p[:size]
}

type Instance struct {
	Exports Exports
}

type Exports struct{}

func (e *Exports) Poseidon(pState []byte, pIn []byte, n int, pOut []byte, nOut int) {
	// Implement Poseidon cryptographic function here
}

func getCurveFromName(name string, flag bool, wasmBuilder func()) (*BN128, error) {
	if name != "bn128" {
		return nil, errors.New("curve not found")
	}
	return &BN128{
		Fr: Field{},
		Tm: &TM{
			Instance: Instance{
				Exports: Exports{},
			},
		},
	}, nil
}

func BuildPoseidon() (func(arr interface{}, state interface{}, nOut int) (interface{}, error), error) {
	bn128, err := getCurveFromName("bn128", true, func() {
		// Placeholder for buildPoseidonWasm function
	})
	if err != nil {
		return nil, err
	}

	F := bn128.Fr
	pState := bn128.Tm.Alloc(32)
	pIn := bn128.Tm.Alloc(32 * 16)
	pOut := bn128.Tm.Alloc(32 * 17)

	poseidon := func(arr interface{}, state interface{}, nOut int) (interface{}, error) {
		var buff []byte
		var n int

		switch v := arr.(type) {
		case []interface{}:
			n = len(v)
			buff = make([]byte, n*32)
			for i := 0; i < n; i++ {
				copy(buff[i*32:], F.E(v[i]))
			}
		case []byte:
			buff = v
			n = len(buff) / 32
			if n*32 != len(buff) {
				return nil, errors.New("invalid input buffer size")
			}
		default:
			return nil, errors.New("invalid input type")
		}

		bn128.Tm.SetBuff(pIn, buff)

		if n < 1 || n > 16 {
			return nil, errors.New("invalid poseidon size")
		}

		var stateVal []byte
		if state == nil {
			stateVal = F.Zero()
		} else {
			stateVal = F.E(state)
		}

		bn128.Tm.SetBuff(pState, stateVal)
		if nOut == 0 {
			nOut = 1
		}

		bn128.Tm.Instance.Exports.Poseidon(pState, pIn, n, pOut, nOut)

		if nOut == 1 {
			return bn128.Tm.GetBuff(pOut, 32), nil
		}

		out := make([][]byte, nOut)
		for i := 0; i < nOut; i++ {
			out[i] = bn128.Tm.GetBuff(pOut[i*32:], 32)
		}
		return out, nil
	}

	return poseidon, nil
}
