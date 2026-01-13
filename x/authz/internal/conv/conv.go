package conv

import "unsafe"

// UnsafeStrToBytes converts string to byte slice without allocation.
// WARNING: The returned byte slice must not be modified.
func UnsafeStrToBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// UnsafeBytesToStr converts byte slice to string without allocation.
// WARNING: The input byte slice must not be modified after this call.
func UnsafeBytesToStr(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
