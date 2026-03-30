package types

const (
	// Gas constants for JWS verification queries.
	// ValidateJWT (Stargate-whitelisted) does NOT charge explicit gas — see
	// query_validate_jwt.go for the rationale. VerifyJWS is not whitelisted
	// and charges gas to bound verification cost.

	// JWSVerifyBaseGas is the flat overhead charged on every VerifyJWS call.
	JWSVerifyBaseGas uint64 = 50_000
	// JWSVerifyPerByteGas is charged per byte of the stored key for VerifyJWS.
	JWSVerifyPerByteGas uint64 = 10
)
