package types

const (
	// Gas constants for JWT/JWS verification queries.
	// These are charged to prevent free DoS via Stargate-whitelisted or
	// CosmWasm-callable query endpoints.

	// JWTVerifyBaseGas is the flat overhead charged on every ValidateJWT call.
	JWTVerifyBaseGas uint64 = 50_000
	// JWTVerifyPerByteGas is charged per byte of the stored key for ValidateJWT.
	JWTVerifyPerByteGas uint64 = 10

	// JWSVerifyBaseGas is the flat overhead charged on every VerifyJWS call.
	JWSVerifyBaseGas uint64 = 50_000
	// JWSVerifyPerByteGas is charged per byte of the stored key for VerifyJWS.
	JWSVerifyPerByteGas uint64 = 10
)
