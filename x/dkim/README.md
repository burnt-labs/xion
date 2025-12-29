# DKIM Cosmos SDK Module

This Cosmos SDK module mimics DKIM (DomainKeys Identified Mail) functionality by securely storing information found in email headers, specifically DKIM public keys and hashes associated with specific domains. This allows verification of email authenticity on the blockchain. The module also implements a method to calculate Poseidon hashes of public keys, providing a secure and efficient way to verify the authenticity of a public key that signed an email.

## Overview

The DKIM module enables the storage, querying, and management of DKIM public keys and their associated Poseidon hashes. It provides gRPC and REST endpoints for:

- Retrieving module parameters.
- Querying DKIM public keys for a given selector and domain.
- Calculating the Poseidon hash of a public key.
- Governance-managed adding and removal of DKIM public keys.

## Features

### 1. Dkim record generation

The module includes functionality to generate a dkim record for a given domain and selector.Refer to the example usage section for more details.

### 2. Governance-Controlled Key Management

DKIM public keys are managed by the Cosmos governance module, allowing for secure addition and removal of DKIM public keys by authorized entities.

### 3. Parameter Management

The module parameters can be updated via governance to adjust module behavior as necessary.

## gRPC Endpoints

The module provides the following gRPC service methods for querying and interacting with DKIM data.

### `Query` Service

#### 1. `Params`

Retrieves the current parameters of the DKIM module.

- **Request**: `QueryParamsRequest`
- **Response**: `QueryParamsResponse`

#### 2. `DkimPubKey`

Fetches the DKIM public key and its Poseidon hash for a specified selector and domain.

- **Request**: `QueryDkimPubKeyRequest`
  - `selector`: The DKIM selector (e.g., a unique identifier for the public key in DNS records).
  - `domain`: The associated domain for the DKIM key.
- **Response**: `QueryDkimPubKeyResponse`
  - `dkim_pubkey`: The stored DKIM public key.
  - `poseidon_hash`: Poseidon hash of the public key.

### `Msg` Service

#### 1. `UpdateParams`

Allows governance to update module parameters.

- **Request**: `MsgUpdateParams`
  - `authority`: Address of the governance account.
  - `params`: New module parameters.
- **Response**: `MsgUpdateParamsResponse`

#### 2. `AddDkimPubKey`

Allows governance to add one or more DKIM public keys.

- **Request**: `MsgAddDkimPubKey`
  - `authority`: Address of the governance account.
  - `dkim_pubkeys`: List of DKIM public keys to be added.
- **Response**: `MsgAddDkimPubKeyResponse`

#### 3. `RemoveDkimPubKey`

Allows governance to remove a DKIM public key for a specific selector and domain.

- **Request**: `MsgRemoveDkimPubKey`
  - `authority`: Address of the governance account.
  - `selector`: DKIM selector to remove.
  - `domain`: Associated domain of the DKIM key.
- **Response**: `MsgRemoveDkimPubKeyResponse`

## Data Structures

### Messages

- **`QueryParamsRequest` / `QueryParamsResponse`**: For querying and retrieving module parameters.
- **`QueryDkimPubKeyRequest` / `QueryDkimPubKeyResponse`**: For querying DKIM public keys and Poseidon hashes.

### Governance Messages

- **`MsgUpdateParams` / `MsgUpdateParamsResponse`**: Used by governance to update parameters.
- **`MsgAddDkimPubKey` / `MsgAddDkimPubKeyResponse`**: For adding new DKIM public keys via governance.
- **`MsgRemoveDkimPubKey` / `MsgRemoveDkimPubKeyResponse`**: For removing existing DKIM public keys.

## Example Usage

### Adding a DKIM Public Key

To add a new DKIM public key:

1. Create and submit a `MsgAddDkimPubKey` message with the authority and DKIM public key details.

### Querying a DKIM Public Key

To fetch the stored DKIM public key for a given selector and domain, call `QueryDkimPubKey` with the appropriate request parameters.

### Generating a Dkim record

To generate the dkim record for a domain,selector pair, use the `gdkim` CLI query command, passing the domain and selectors as arguments.

```bash
xiond query dkim gdkim <domain> <selector>
```

## License

This module is open-sourced under the [MIT License](LICENSE).
