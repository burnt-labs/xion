# XION Module

The XION module provides core platform functionality for the XION blockchain, including platform fee management, WebAuthn signature validation utilities, and support for XION's Abstract Account architecture.

## Overview

XION is a blockchain designed for consumer applications with a focus on user experience through Abstract Accounts. The XION module serves as the foundational layer providing essential platform services and utilities.

## Key Features

### 1. Platform Fee Management

The XION module manages platform fees collected on transactions, providing governance-controlled fee parameters:

#### Platform Percentage

- **Purpose**: Sets the percentage of transaction fees collected as platform revenue
- **Default**: 0% (configurable via governance)
- **Range**: 0-10000 basis points (0-100%)
- **Governance**: Adjustable through `MsgSetPlatformPercentage` proposals

#### Platform Minimums

- **Purpose**: Sets minimum fee amounts required for transactions
- **Default**: Empty (no minimums)
- **Multi-denomination**: Supports different minimums for different token denominations
- **Governance**: Adjustable through `MsgSetPlatformMinimum` proposals

### 2. WebAuthn Signature Validation

The module provides cryptographic validation utilities for WebAuthn signatures used in XION's Abstract Account system:

#### Security Architecture

**Important**: These functions are **UTILITY FUNCTIONS ONLY** - they do not provide authentication or authorization for XION accounts.

XION uses an Abstract Account architecture where:

1. **User accounts are smart contracts** deployed on the XION blockchain
2. **WebAuthn credentials are stored in contract state** during account creation
3. **Authorization happens at the contract level** - each contract validates signatures against its stored credentials
4. **This module provides signature validation utilities** that contracts can use

#### Available Functions

- **`WebAuthNVerifyRegister`**: Validates WebAuthn registration data cryptographically
- **`WebAuthNVerifyAuthenticate`**: Validates WebAuthn authentication assertions cryptographically

#### Security Considerations

The client-controlled parameters (like `rp` - Relying Party URL) in these functions are safe because:

- These functions only validate cryptographic signatures
- They do not grant access to accounts or funds
- Real authorization happens at the Abstract Account contract level
- An attacker cannot use their credential to access another user's account because:
  - The victim's contract only recognizes the victim's stored credentials
  - The attacker's credential is not stored in the victim's contract
  - Contract-level validation prevents unauthorized operations

For detailed security documentation, see the inline comments in `keeper/grpc_query.go`.

## Module Structure

```text
x/xion/
├── client/cli/          # CLI commands for platform fee queries and transactions
├── keeper/              # State management and business logic
│   ├── grpc_query.go   # Query handlers (with detailed security docs)
│   ├── keeper.go       # Core keeper functionality
│   └── msg_server.go   # Transaction message handlers
├── types/               # Protocol buffer definitions and types
│   ├── webauthn.go     # WebAuthn utility functions
│   └── ...
└── module.go           # Cosmos SDK module definition
```

## Queries

### Platform Fee Queries

```bash
# Get current platform percentage
xiond query xion platform-percentage

# Get platform minimum fees
xiond query xion platform-minimum
```

### WebAuthn Validation Queries

```bash
# Validate WebAuthn registration data
xiond query xion webauthn-register [addr] [challenge] [rp] [data]

# Validate WebAuthn authentication assertion
xiond query xion webauthn-authenticate [addr] [challenge] [rp] [credential] [data]
```

## Transactions

### Platform Fee Management (Governance Only)

```bash
# Set platform percentage (requires governance proposal)
xiond tx gov submit-proposal [proposal.json]

# Example proposal to set 5% platform fee:
{
  "messages": [
    {
      "@type": "/burnt.xion.v1.MsgSetPlatformPercentage",
      "authority": "xion10d07y265gmmuvt4z0w9aw880jnsr700jdufnyd",
      "platform_percentage": 500
    }
  ],
  "metadata": "",
  "deposit": "100uxion",
  "title": "Set Platform Fee to 5%",
  "summary": "Proposal to set platform fee percentage to 5%"
}
```

## Integration with Other Modules

### Abstract Accounts

- Provides WebAuthn signature validation utilities for Abstract Account contracts
- Contracts use these utilities to validate user authentication assertions
- Security is enforced at the contract level, not in this module

### Fee Abstraction

- Works with the `feeabs` module to enable alternative fee payment methods
- Platform fees are collected in addition to gas fees

### Global Fee

- Integrates with the `globalfee` module for minimum fee requirements
- Platform minimums work alongside global minimums

## Development and Testing

### Running Tests

```bash
# Run unit tests for the XION module
make test-unit

# Run integration tests
make test-integration-xion-send-platform-fee
make test-integration-xion-min-default
make test-integration-xion-min-zero
```

### Key Integration Tests

- **Platform Fee Tests**: Verify fee collection and percentage calculations
- **Minimum Fee Tests**: Ensure minimum fee requirements are enforced
- **WebAuthn Tests**: Validate signature verification utilities
- **Abstract Account Tests**: Test integration with Abstract Account contracts

## Configuration

### Genesis Parameters

```json
{
  "platform_percentage": "0",
  "platform_minimums": []
}
```

### Governance Parameters

- `PlatformPercentage`: Controlled by governance via `MsgSetPlatformPercentage`
- `PlatformMinimums`: Controlled by governance via `MsgSetPlatformMinimum`

## Security

### Platform Fees

- Fee collection is deterministic and transparent
- All fee changes require governance approval
- Platform fees are separate from validator rewards

### WebAuthn Utilities

- Functions validate only cryptographic signatures
- No authentication or authorization is provided by this module
- Security boundaries are clearly documented in code
- Designed to prevent misunderstanding of security model

## Module Dependencies

- **Cosmos SDK**: Standard blockchain functionality
- **WebAuthn Libraries**: `github.com/go-webauthn/webauthn` for cryptographic validation
- **WASM**: Integration with CosmWasm for Abstract Account contracts
- **Bank Module**: For fee collection and transfers
- **Auth Module**: For account management integration

## Versioning

This module follows semantic versioning and is part of the XION blockchain's core module set. Breaking changes to the module interface require coordinated upgrades across the network.
