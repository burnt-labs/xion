# Security Policy

This policy covers the XION chain node (`xiond`), its custom Cosmos SDK modules,
and this repository's integration with the abstract account system. The
abstract account module and authenticator contract infrastructure are maintained
in [`burnt-labs/abstract-account`](https://github.com/burnt-labs/abstract-account)
and are governed by that repository's policy and the published bug bounty
program.

It supplements the
[organization-wide policy](https://github.com/burnt-labs/.github/blob/main/SECURITY.md),
which governs anything not addressed here.

## Reporting a Vulnerability

**Do not open a public GitHub issue for a security vulnerability.**

| Type of finding                  | How to report                                         |
| -------------------------------- | ----------------------------------------------------- |
| Security vulnerability           | Email [security@burnt.com](mailto:security@burnt.com)  |
| Non-sensitive or operational bug | Open a [GitHub issue](https://github.com/burnt-labs/xion/issues/new) |

You may also submit a private report through the repository's
**Security → Report a vulnerability** tab.

Include the type of vulnerability, affected version, steps to reproduce, impact,
how an attacker would exploit it, and any known mitigations.

We acknowledge receipt within **5 business days** and provide a triage decision
within **14 days**. Active exploitation, or confirmed attacker awareness of an
unpatched vulnerability, escalates the issue to Critical handling regardless of
its original classification.

## Proof of Concept Requirements

**Reports must include an end-to-end proof of concept.** Severity is assessed on
demonstrated impact under real-world constraints, not theoretical worst-case
scenarios.

Unit tests using `setupKeeper(t)` or similar harnesses bypass transaction
encoding, routing, and the ante handler chain, and do not demonstrate on-chain
exploitability on their own.

The proof of concept should run against a **locally running XION node configured
with the parameters and module configuration of the currently deployed mainnet
release**. The attack should be executed via standard transaction broadcast and
must confirm that the transaction was included in a block and succeeded during
`DeliverTx`. A `BroadcastTxSync` response alone confirms only `CheckTx` and is
not sufficient. Simulated environments that model chain state without running a
full node do not demonstrate exploitability.

## Permissioned Chain Policy

XION mainnet operates with `code_upload_access: Nobody`. Uploading new contract
code requires governance approval. **This is a fundamental architectural
constraint, not a bypass target.** Instantiating an already-approved code ID is
governed separately by that code ID's instantiate permission and is not excluded
by this rule.

Any attack vector requiring an attacker to upload new malicious contract code on
mainnet is out of scope, regardless of technical validity. This includes
amplification attacks and exploit chains that depend on attacker-controlled
bytecode not already approved for mainnet.

## Privileged Actor Policy

Attacks requiring a privileged party — governance, a module authority, or a
validator — to take self-destructive or colluding action are classified at
**Medium at most**, regardless of downstream impact. This includes validators
supplying unusual inputs, extreme timestamps, delayed responses, or off-spec
data to consensus rounds. The threat model assumes privileged actors operate
within the specified protocol parameters.

## Authentication Impact Scope

Authentication weaknesses whose impact is limited to accounts created after the
attack is established — and which cannot affect the funds, state, or
authentication of any account funded and operational before the attack began —
are capped at **Medium**, regardless of the authentication mechanism involved. A
High or Critical authentication finding must demonstrate unauthorized impact on
a pre-existing funded account.

## Out of Scope

**Assets**

- Smart contracts — see [`burnt-labs/contracts`](https://github.com/burnt-labs/contracts/blob/main/SECURITY.md)
- Frontend applications and web properties
- Third-party infrastructure, RPC providers, and external dependencies
- Public blockchain RPC, REST, gRPC, and Tendermint RPC endpoints — these expose
  blockchain state by design and are operated by validators and node operators
  as a public service
- Upstream dependencies — vulnerabilities in CosmWasm, the Cosmos SDK, IBC, or
  the Barretenberg C library are not eligible here; only code originating in
  this repository is covered

**Vulnerability classes**

- Attacks requiring the upload of new malicious contract code on mainnet
- Denial of service of any form, including single-transaction resource
  exhaustion, node crashes, and chain halts recoverable via a software patch,
  coordinated validator restart, or governance parameter update. Chain halts
  requiring a hard fork to resolve remain in scope
- Governance attacks requiring a malicious proposal to pass
- Theoretical vulnerabilities without a working end-to-end proof of concept
- Attacks where the attacker's cost to execute exceeds the demonstrable harm to
  the protocol or its users
- Findings affecting only deprecated or end-of-life versions, or already
  remediated in the currently deployed mainnet version, regardless of whether
  the fix was publicly announced
- Best practices, gas optimizations, missing events, and informational findings

Reporters are responsible for verifying exploitability against the currently
deployed version before submission.

## Severity Characterization

| Severity     | Description                                                                                                     |
| ------------ | --------------------------------------------------------------------------------------------------------------- |
| **CRITICAL** | Direct, permanent, irrecoverable theft or loss of user funds at protocol scale. Unauthorized minting. Chain halt or consensus failure requiring a hard fork. Complete bypass of abstract account authentication enabling arbitrary transaction authorization |
| **HIGH**     | Theft or freezing of user funds affecting individual accounts. Significant authentication bypass with demonstrated exploitability |
| **MEDIUM**   | Limited fund loss or temporary disruption requiring specific preconditions. Attacks requiring privileged-party cooperation. Partial authentication bypass requiring secondary conditions |
| **LOW**      | Valid, reproducible code-level issue with no direct risk to funds or chain safety, representing a meaningful hardening opportunity. Must include a specific code reference |

Severity is assessed by Burnt Labs based on demonstrated impact under real-world
constraints. Reports submitted at a severity that does not match the definitions
above are assessed as written; we do not reclassify or negotiate severity on a
reporter's behalf.

## Responsible Disclosure

- Do not exploit a vulnerability beyond what is necessary to confirm it exists
- **Do not test against XION mainnet.** Testing that targets live production
  systems will disqualify the report
- Do not access, modify, or exfiltrate user data
- Do not disclose publicly before a fix is confirmed and deployed

## Commitment to the CosmWasm Community

We are committed to sharing security issues and bugs with the CosmWasm
community. Critical vulnerabilities affecting CosmWasm components are reported
to the CosmWasm security team through their non-public channels before public
disclosure.

## Safe Harbor

Burnt Labs will not pursue legal action against researchers who report
vulnerabilities in good faith under this policy, do not exploit beyond what is
necessary to confirm the finding, do not access or disclose user data, and do
not disrupt production systems.

Authorization to actively test extends only to assets named in a published Burnt
Labs bug bounty program. Testing systems outside that scope is not authorized.
Reporting a vulnerability you encountered incidentally is always welcome.

## Frequently Raised Non-Issues

The following design decisions are sometimes reported as vulnerabilities but are
intentional and will not be changed.

### DKIM public keys stored on-chain

The `x/dkim` module stores RSA **public** keys on-chain — the same data any mail
server operator publishes in DNS TXT records. Storing them on-chain enables
trustless DKIM verification inside ZK circuits and is a core feature of the XION
email-based account system. No private key material is stored in the module or
in on-chain state.

### Bank MsgSend platform fee exemption is a governance parameter

The platform fee applied to `MsgSend` transactions can be set to zero for
specific addresses (for example, protocol contracts) via a governance parameter.
This is an intentional administrative mechanism, not a privilege escalation. Any
change to the exemption list requires an on-chain governance vote and is fully
auditable in the transaction history.
