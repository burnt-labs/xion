# XION Security Policy

This repository is the XION chain-node asset in the
[Blockchain / DLT bug bounty program](https://github.com/burnt-labs/bug-bounty/blob/main/programs/blockchain.md).
This file adds repository-specific reporting and coordination details. The
published [`burnt-labs/bug-bounty`](https://github.com/burnt-labs/bug-bounty)
program is canonical for scope, severity, eligibility, exclusions, rewards, and
testing authorization; where the documents differ, the program terms govern.

Do **not** open public GitHub issues, pull requests, discussions, or comments
containing vulnerability details.

## Reporting a Vulnerability

Use GitHub Private Vulnerability Reporting:

- [Report a vulnerability in `burnt-labs/xion`](https://github.com/burnt-labs/xion/security/advisories/new)

If GitHub private reporting is unavailable, email
[security@burnt.com](mailto:security@burnt.com) with a minimal, non-public
report.

## Report Requirements

Include as much of the following as possible:

- Affected component, branch, tag or commit, and deployed environment
- Vulnerability type and concise summary
- End-to-end reproduction steps and proof of concept
- Exploit preconditions and assumptions
- Demonstrated impact
- Suggested severity and rationale
- Intended disclosure timeline, if any

Automated scanner output without a demonstrated exploit path is not sufficient.

## Response Targets

We aim to acknowledge reports within **5 business days** and provide a triage
decision within **14 days**.

Active exploitation, or confirmed attacker awareness of an unpatched
vulnerability, escalates the issue to Critical **response handling** —
prioritization, coordination, and disclosure timing — regardless of its
original classification. That escalation does not change the finding's severity
assessment or reward eligibility.

## Coordinated Disclosure

Keep vulnerability details private until Burnt Labs confirms a fix or mitigation
has been deployed and disclosure is coordinated. Where appropriate, Burnt Labs
will publish a GitHub Security Advisory and credit finders, reporters, analysts,
and remediation contributors using GitHub advisory credits.

If a security issue requires a network upgrade, additional time may be needed to
raise a governance proposal and complete the upgrade.

## Downstream Notification

Burnt Labs notifies downstream consumers before a security fix becomes publicly
visible. A fix in a public repository is itself a disclosure: the vulnerability
can be derived from the patch, and every unpatched deployment is exposed from
that moment rather than from a later announcement.

"Publicly visible" means the earliest point at which the patch can be read by
anyone outside the embargo — a commit pushed to a public branch, a public pull
request, a tagged release, or a published advisory. Security fixes are developed
in the temporary private fork attached to a draft GitHub Security Advisory. A
public pull request is opened only after the notice period below has run, or
under the active-exploitation exception.

### Who Is Notified

- Validators and node operators running XION mainnet or XION testnet, for issues
  affecting the chain
- Teams that registered a security contact for an in-scope Burnt Labs
  repository
- The CosmWasm security team, through non-public channels, for critical issues
  affecting CosmWasm components

### How and When

Recipients are added to the draft GitHub Security Advisory before publication.
Where a recipient cannot be reached that way, Burnt Labs emails the contact
address registered with [security@burnt.com](mailto:security@burnt.com).

Notice is a minimum of seven days before the fix becomes publicly visible. The
exception is a vulnerability under active exploitation: Burnt Labs ships the fix
first and notifies as quickly as it can. Where remediation requires a network
upgrade, notification precedes the governance proposal.

Validators, node operators, and teams building on an in-scope repository can
register a security contact by emailing
[security@burnt.com](mailto:security@burnt.com) with a contact address and the
network or repository they operate.

## Scope, Severity, and Rewards

The canonical Blockchain / DLT program lists every eligible repository and the
fork-delta rule for Burnt-maintained dependencies. Scope applies to the current
mainnet release. Findings affecting only deprecated versions, or already
remediated in the currently deployed release, are not eligible.

Only **High** and **Critical** findings are reward eligible. Burnt Labs does not
publish reward amounts. The canonical program governs KYC, duplicate handling,
severity assessment, and all other reward terms.

## Proof of Concept

An end-to-end proof of concept is required. Unit tests using `setupKeeper(t)` or
similar harnesses bypass transaction encoding, routing, the ante handler chain,
and block execution; they do not demonstrate on-chain exploitability on their
own.

Run the proof of concept against a locally running XION node configured with
mainnet parameters, the XION ante handler chain, module set, and governance
configuration. Execute the attack through standard transaction broadcast.
Broadcast acceptance alone is not sufficient: show inclusion in a block, the
successful execution result, and the resulting state change or security impact.

## Authentication Impact Scope

Authentication weaknesses whose impact is limited to accounts created after the
attack is established — and which cannot affect the funds, state, or
authentication of an account funded and operational before the attack began —
are capped at **Medium**. A High or Critical authentication finding must
demonstrate unauthorized impact on a pre-existing funded account.

## Permissioned Chain Policy

XION mainnet operates with `code_upload_access: Nobody`. Uploading new contract
code requires governance approval. An attack that depends on uploading
attacker-controlled contract code to mainnet is out of scope. A finding that is
exploitable through code already approved for mainnet is not excluded by this
rule.

## Privileged Actor Policy

Findings are classified at **Medium at most** when the attack must begin with
control of governance, a module authority, validator or operator credentials,
or another privileged role — or requires that holder to cooperate — and the
demonstrated action is already within that role's intended authority. This
includes validators deliberately supplying unusual inputs, extreme timestamps,
delayed responses, or off-spec data to consensus rounds.

The cap does not apply when a flaw lets an attacker who starts without that
privilege obtain it or bypass its authorization check, or lets a legitimately
held limited role perform actions outside its intended permissions. Those
findings are assessed by demonstrated impact. This policy does not authorize
researchers to acquire or exercise production privileges they do not
legitimately control, or to test with production privileges they do control.

## Repository-Specific Exclusions

In addition to the canonical program exclusions, the following are out of scope
for this repository:

- Third-party infrastructure, RPC providers, and external dependencies
- Public RPC, REST, gRPC, and Tendermint RPC endpoints that expose chain state by
  design
- Upstream dependency code. Burnt-maintained fork deltas are covered only under
  the separately listed fork repositories in the canonical program
- Third-party contracts deployed on XION
- Attacks requiring new malicious contract code to be uploaded to mainnet
- Governance attacks requiring a malicious proposal to pass
- Denial of service recoverable through a software patch, coordinated validator
  restart, or governance parameter update. A chain halt requiring a hard fork
  remains in scope under the canonical Critical definition
- Theoretical findings without a working end-to-end proof of concept
- Attacks whose execution cost exceeds the demonstrated harm
- Best practices, gas optimizations, missing events, and informational findings

## Responsible Disclosure and Safe Harbor

Do not test against XION mainnet, public XION testnets, public RPC
infrastructure, or other production systems. Use a locally running node or
infrastructure you control. Do not access or disclose user data, disrupt
services, or exploit beyond what is necessary to confirm the finding.

Naming this repository as an asset establishes eligibility, not permission to
test a production deployment. Good-faith research within the authorized local or
researcher-controlled environments is covered by the canonical program's safe
harbor. Reporting a vulnerability encountered incidentally is always welcome.

## Frequently Raised Non-Issues

The following design decisions are intentional.

### DKIM Public Keys Stored On-Chain

The `x/dkim` module stores RSA public keys on-chain — the same data that a mail
server operator publishes in DNS TXT records. Storing them on-chain enables
trustless DKIM verification inside ZK circuits. No private key material is
stored in the module or on-chain state.

### Bank `MsgSend` Platform-Fee Exemption

The platform fee applied to `MsgSend` transactions can be set to zero for
specific addresses, such as protocol contracts, through a governance parameter.
This is an intentional administrative mechanism. Changing the exemption list
requires an on-chain governance vote and is auditable in transaction history.

## Recognition

Burnt Labs credits researchers who help improve XION security. Recognition may
be included in GitHub Security Advisories, release notes, and public security
bulletins after coordinated disclosure.
