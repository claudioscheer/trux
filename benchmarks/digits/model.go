package main

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Notebook SVM shape: load_digits, 64 features, SVC(gamma=0.001).
// The notebook's trained model has 519 support vectors, dual coefficients
// shaped 9x519, and 45 intercepts. The synthetic file uses that shape.
// Class 0 owns the first 52 vectors, which are zeros, so an all-zero image
// votes for 0 by a wide margin. The other 467 vectors are 16s and their
// dual coefficients are 0, so the host vote does not depend on exp of a
// large distance.
const (
	digitsFeatures = 64
	digitsClasses  = 10
	digitsPairs    = 45
	digitsSV       = 519
	digitsGamma    = 0.001
	txmDTypeInt    = 1
	txmDTypeFloat  = 2
)

type digitsModel struct {
	gamma     float64
	support   []float64
	dual      []float64
	intercept []float64
	nSupport  []int64
	nSV       int
}

func zeroImage() []float64 {
	return make([]float64, digitsFeatures)
}

func syntheticModel() digitsModel {
	nSupport := make([]int64, digitsClasses)
	for i := 0; i < 9; i++ {
		nSupport[i] = 52
	}
	nSupport[9] = 51

	support := make([]float64, digitsSV*digitsFeatures)
	for sv := 52; sv < digitsSV; sv++ {
		for f := 0; f < digitsFeatures; f++ {
			support[sv*digitsFeatures+f] = 16
		}
	}

	dual := make([]float64, 9*digitsSV)
	for row := 0; row < 9; row++ {
		for col := 0; col < 52; col++ {
			dual[row*digitsSV+col] = 1
		}
	}

	return digitsModel{
		gamma:     digitsGamma,
		support:   support,
		dual:      dual,
		intercept: make([]float64, digitsPairs),
		nSupport:  nSupport,
		nSV:       digitsSV,
	}
}

func (m digitsModel) scores(input []float64) []float64 {
	out := make([]float64, m.nSV)
	for sv := 0; sv < m.nSV; sv++ {
		dist := 0.0
		base := sv * digitsFeatures
		for i := 0; i < digitsFeatures; i++ {
			d := m.support[base+i] - input[i]
			dist += d * d
		}
		out[sv] = math.Exp(-m.gamma * dist)
	}
	return out
}

// voteDigit is the host one-vs-one loop in benchmarks/digits/server.tx.
func (m digitsModel) voteDigit(scores []float64) int {
	start := make([]int, digitsClasses)
	acc := 0
	for c := 0; c < digitsClasses; c++ {
		start[c] = acc
		acc += int(m.nSupport[c])
	}
	votes := make([]int, digitsClasses)
	p := 0
	for i := 0; i < digitsClasses; i++ {
		for j := i + 1; j < digitsClasses; j++ {
			sum := m.intercept[p]
			row1 := j - 1
			row2 := i
			for k := 0; k < int(m.nSupport[i]); k++ {
				idx := start[i] + k
				sum += m.dual[row1*m.nSV+idx] * scores[idx]
			}
			for k := 0; k < int(m.nSupport[j]); k++ {
				idx := start[j] + k
				sum += m.dual[row2*m.nSV+idx] * scores[idx]
			}
			if sum > 0 {
				votes[i]++
			} else {
				votes[j]++
			}
			p++
		}
	}
	best := votes[0]
	digit := 0
	for c := 1; c < digitsClasses; c++ {
		if votes[c] > best {
			best = votes[c]
			digit = c
		}
	}
	return digit
}

func (m digitsModel) encodeTXM() []byte {
	out := []byte("TXM1")
	out = appendU32(out, 1)
	out = appendU32(out, 5)
	out = addFloatTensor(out, "gamma", []uint64{1}, []float64{m.gamma})
	out = addFloatTensor(out, "support", []uint64{uint64(m.nSV), digitsFeatures}, m.support)
	out = addFloatTensor(out, "dual", []uint64{9, uint64(m.nSV)}, m.dual)
	out = addFloatTensor(out, "intercept", []uint64{digitsPairs}, m.intercept)
	out = addIntTensor(out, "n_support", []uint64{digitsClasses}, m.nSupport)
	return out
}

func syntheticTXM() []byte {
	return syntheticModel().encodeTXM()
}

func parseModel(data []byte) (digitsModel, error) {
	if len(data) < 12 || string(data[:4]) != "TXM1" {
		return digitsModel{}, fmt.Errorf("model file has a bad magic")
	}
	version := binary.LittleEndian.Uint32(data[4:8])
	if version != 1 {
		return digitsModel{}, fmt.Errorf("model file has a bad version")
	}
	count := binary.LittleEndian.Uint32(data[8:12])
	rest := data[12:]
	tensors := map[string]parsedTensor{}
	for n := uint32(0); n < count; n++ {
		tensor, next, err := readTensor(rest)
		if err != nil {
			return digitsModel{}, err
		}
		if _, ok := tensors[tensor.name]; ok {
			return digitsModel{}, fmt.Errorf("model file has a duplicate tensor name")
		}
		tensors[tensor.name] = tensor
		rest = next
	}
	if len(rest) != 0 {
		return digitsModel{}, fmt.Errorf("model file has trailing bytes")
	}

	gamma, err := floatTensor(tensors, "gamma", []uint64{1})
	if err != nil {
		return digitsModel{}, err
	}
	nSupport, err := intTensor(tensors, "n_support", []uint64{digitsClasses})
	if err != nil {
		return digitsModel{}, err
	}
	var nSV int64
	for _, n := range nSupport {
		if n < 0 {
			return digitsModel{}, fmt.Errorf("n_support has a negative count")
		}
		nSV += n
	}
	if nSV < 1 {
		return digitsModel{}, fmt.Errorf("n_support is empty")
	}
	support, err := floatTensor(tensors, "support", []uint64{uint64(nSV), digitsFeatures})
	if err != nil {
		return digitsModel{}, err
	}
	dual, err := floatTensor(tensors, "dual", []uint64{9, uint64(nSV)})
	if err != nil {
		return digitsModel{}, err
	}
	intercept, err := floatTensor(tensors, "intercept", []uint64{digitsPairs})
	if err != nil {
		return digitsModel{}, err
	}
	return digitsModel{
		gamma:     gamma[0],
		support:   support,
		dual:      dual,
		intercept: intercept,
		nSupport:  nSupport,
		nSV:       int(nSV),
	}, nil
}

type parsedTensor struct {
	name  string
	dtype uint32
	dims  []uint64
	raw   []byte
}

func readTensor(data []byte) (parsedTensor, []byte, error) {
	if len(data) < 4 {
		return parsedTensor{}, nil, fmt.Errorf("model file is truncated")
	}
	nameLen := binary.LittleEndian.Uint32(data[:4])
	data = data[4:]
	if nameLen < 1 || nameLen > 64 || uint32(len(data)) < nameLen {
		return parsedTensor{}, nil, fmt.Errorf("model tensor name has a bad length")
	}
	name := string(data[:nameLen])
	data = data[nameLen:]
	if len(data) < 8 {
		return parsedTensor{}, nil, fmt.Errorf("model file is truncated")
	}
	dtype := binary.LittleEndian.Uint32(data[:4])
	rank := binary.LittleEndian.Uint32(data[4:8])
	data = data[8:]
	if dtype != txmDTypeInt && dtype != txmDTypeFloat {
		return parsedTensor{}, nil, fmt.Errorf("model tensor has a bad dtype")
	}
	if rank < 1 || rank > 4 {
		return parsedTensor{}, nil, fmt.Errorf("model tensor rank must be from 1 to 4")
	}
	if uint32(len(data)) < rank*8 {
		return parsedTensor{}, nil, fmt.Errorf("model file is truncated")
	}
	dims := make([]uint64, rank)
	product := uint64(1)
	for i := uint32(0); i < rank; i++ {
		dims[i] = binary.LittleEndian.Uint64(data[:8])
		data = data[8:]
		if dims[i] < 1 {
			return parsedTensor{}, nil, fmt.Errorf("model tensor dimension must be at least 1")
		}
		if product > (1<<63)/dims[i] {
			return parsedTensor{}, nil, fmt.Errorf("model tensor dimension product overflows")
		}
		product *= dims[i]
	}
	if product > (1<<63)/8 {
		return parsedTensor{}, nil, fmt.Errorf("model tensor dimension product overflows")
	}
	nbytes := int(product * 8)
	if len(data) < nbytes {
		return parsedTensor{}, nil, fmt.Errorf("model file is truncated")
	}
	raw := append([]byte(nil), data[:nbytes]...)
	return parsedTensor{name: name, dtype: dtype, dims: dims, raw: raw}, data[nbytes:], nil
}

func floatTensor(tensors map[string]parsedTensor, name string, dims []uint64) ([]float64, error) {
	tensor, ok := tensors[name]
	if !ok {
		return nil, fmt.Errorf("model file is missing %s", name)
	}
	if tensor.dtype != txmDTypeFloat || !sameDims(tensor.dims, dims) {
		return nil, fmt.Errorf("model tensor %s has the wrong shape", name)
	}
	out := make([]float64, len(tensor.raw)/8)
	for i := range out {
		out[i] = math.Float64frombits(binary.LittleEndian.Uint64(tensor.raw[i*8:]))
	}
	return out, nil
}

func intTensor(tensors map[string]parsedTensor, name string, dims []uint64) ([]int64, error) {
	tensor, ok := tensors[name]
	if !ok {
		return nil, fmt.Errorf("model file is missing %s", name)
	}
	if tensor.dtype != txmDTypeInt || !sameDims(tensor.dims, dims) {
		return nil, fmt.Errorf("model tensor %s has the wrong shape", name)
	}
	out := make([]int64, len(tensor.raw)/8)
	for i := range out {
		out[i] = int64(binary.LittleEndian.Uint64(tensor.raw[i*8:]))
	}
	return out, nil
}

func sameDims(a []uint64, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func addFloatTensor(out []byte, name string, dims []uint64, values []float64) []byte {
	raw := make([]byte, 0, len(values)*8)
	for _, value := range values {
		raw = appendF64(raw, value)
	}
	return addTensor(out, name, txmDTypeFloat, dims, raw)
}

func addIntTensor(out []byte, name string, dims []uint64, values []int64) []byte {
	raw := make([]byte, 0, len(values)*8)
	for _, value := range values {
		raw = appendU64(raw, uint64(value))
	}
	return addTensor(out, name, txmDTypeInt, dims, raw)
}

func addTensor(out []byte, name string, dtype uint32, dims []uint64, payload []byte) []byte {
	out = appendU32(out, uint32(len(name)))
	out = append(out, name...)
	out = appendU32(out, dtype)
	out = appendU32(out, uint32(len(dims)))
	for _, dim := range dims {
		out = appendU64(out, dim)
	}
	return append(out, payload...)
}

func appendU32(out []byte, value uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], value)
	return append(out, buf[:]...)
}

func appendU64(out []byte, value uint64) []byte {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], value)
	return append(out, buf[:]...)
}

func appendF64(out []byte, value float64) []byte {
	return appendU64(out, math.Float64bits(value))
}
