package barretenberg

import "errors"

// Error codes corresponding to bb_error_t in the C wrapper
const (
	errCodeSuccess               = 0
	errCodeInvalidVKey           = 1
	errCodeInvalidProof          = 2
	errCodeInvalidPublicInputs   = 3
	errCodeVerificationFailed    = 4
	errCodeInternal              = 5
	errCodeNullPointer           = 6
	errCodeAllocationFailed      = 7
	errCodeDeserializationFailed = 8
)

// Sentinel errors for Barretenberg operations
var (
	// ErrInvalidVKey indicates the verification key data is malformed or invalid.
	ErrInvalidVKey = errors.New("barretenberg: invalid verification key")

	// ErrInvalidProof indicates the proof data is malformed or invalid.
	ErrInvalidProof = errors.New("barretenberg: invalid proof")

	// ErrInvalidPublicInputs indicates the public inputs are malformed, have wrong count,
	// or contain invalid field elements.
	ErrInvalidPublicInputs = errors.New("barretenberg: invalid public inputs")

	// ErrVerificationFailed indicates the proof did not verify against the given
	// verification key and public inputs. This is not an error in the sense of
	// a malfunction - it means the proof is simply invalid.
	ErrVerificationFailed = errors.New("barretenberg: verification failed")

	// ErrInternal indicates an internal error in the Barretenberg library.
	ErrInternal = errors.New("barretenberg: internal error")

	// ErrNullPointer indicates a null pointer was passed to a function.
	ErrNullPointer = errors.New("barretenberg: null pointer")

	// ErrAllocationFailed indicates memory allocation failed.
	ErrAllocationFailed = errors.New("barretenberg: allocation failed")

	// ErrDeserializationFailed indicates deserialization of data failed.
	ErrDeserializationFailed = errors.New("barretenberg: deserialization failed")

	// ErrClosed indicates an operation was attempted on a closed resource.
	ErrClosed = errors.New("barretenberg: resource is closed")

	// ErrInvalidFieldElement indicates a public input string could not be parsed
	// as a valid field element.
	ErrInvalidFieldElement = errors.New("barretenberg: invalid field element")
)

// errorFromCode converts a C error code to a Go error.
func errorFromCode(code int, detail string) error {
	var baseErr error
	switch code {
	case errCodeSuccess:
		return nil
	case errCodeInvalidVKey:
		baseErr = ErrInvalidVKey
	case errCodeInvalidProof:
		baseErr = ErrInvalidProof
	case errCodeInvalidPublicInputs:
		baseErr = ErrInvalidPublicInputs
	case errCodeVerificationFailed:
		baseErr = ErrVerificationFailed
	case errCodeNullPointer:
		baseErr = ErrNullPointer
	case errCodeAllocationFailed:
		baseErr = ErrAllocationFailed
	case errCodeDeserializationFailed:
		baseErr = ErrDeserializationFailed
	default:
		baseErr = ErrInternal
	}

	if detail != "" {
		return &wrappedError{base: baseErr, detail: detail}
	}
	return baseErr
}

// wrappedError wraps a base error with additional detail from the C library.
type wrappedError struct {
	base   error
	detail string
}

func (e *wrappedError) Error() string {
	return e.base.Error() + ": " + e.detail
}

func (e *wrappedError) Unwrap() error {
	return e.base
}
