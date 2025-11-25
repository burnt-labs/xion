package types

func ToLittleEndian(b []byte) []byte {
	le := make([]byte, len(b))
	for i, v := range b {
		le[len(b)-1-i] = v
	}
	return le
}
