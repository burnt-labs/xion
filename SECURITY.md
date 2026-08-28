# XION Security Policy

Security researchers are essential to keeping XION safe. If you believe you have found a vulnerability in XION or an in-scope Burnt Labs repository, report it privately through the process below.

Do **not** open public GitHub issues, pull requests, discussions, or comments containing vulnerability details.

## Reporting a Vulnerability

Use GitHub Private Vulnerability Reporting when available for the affected repository:

- [Report a vulnerability in `burnt-labs/xion`](https://github.com/burnt-labs/xion/security/advisories/new)

If GitHub private reporting is unavailable for the affected repository, email [security@burnt.com](mailto:security@burnt.com) with a minimal, non-public report.

## Report Requirements

Include as much of the following as possible:

- Affected repository, component, branch/tag/commit, and deployed environment
- Vulnerability type and concise summary
- End-to-end reproduction steps
- Proof of concept
- Exploit preconditions and assumptions
- Demonstrated impact
- Suggested severity and rationale
- Whether you intend to publish details and your disclosure timeline

Automated scanner output without a demonstrated exploit path is not sufficient.

## Response Targets

We aim to acknowledge reports within 5 business days and provide a triage decision within 14 days. Critical reports may be handled faster.

## Coordinated Disclosure

Keep vulnerability details private until Burnt Labs confirms a fix or mitigation has been deployed and disclosure is coordinated. Where appropriate, Burnt Labs will publish a [GitHub Security Advisory](https://docs.github.com/en/code-security/security-advisories) and credit finders, reporters, analysts, and remediation contributors using GitHub advisory credits.

If a security issue requires a network upgrade, additional time may be needed to raise a governance proposal and complete the upgrade.

## Downstream Notification

Coordinated disclosure sets out what a reporter owes Burnt Labs. This section sets out what Burnt Labs owes the people running its code.

A fix appearing in a public repository is disclosure, whether or not an advisory has been published alongside it. Anyone watching the repository can derive the vulnerability from the patch, and every unpatched deployment is exposed from that moment rather than from the announcement. Burnt Labs therefore notifies downstream consumers before a fix becomes publicly visible, not after.

"Publicly visible" means the earliest moment the patch can be read by anyone outside the embargo, whichever comes first: a commit pushed to a public branch, a pull request opened against a public repository, a tagged release, or a published advisory. It is not the merge. Security fixes are therefore developed on the private fork attached to a draft GitHub Security Advisory, and the public pull request is opened only once the notice period below has run or the active-exploitation exception applies.

### Who is notified

- Validators and node operators running XION mainnet or testnet, for any issue affecting the chain.
- Teams that have registered a security contact for an in-scope Burnt Labs repository. [`burnt-labs/abstract-account`](https://github.com/burnt-labs/abstract-account) and [`burnt-labs/barretenberg-go`](https://github.com/burnt-labs/barretenberg-go) are both used outside this repository.
- The CosmWasm security team, through their non-public channels, for critical vulnerabilities affecting CosmWasm components.

### How and when

Notification goes out through the GitHub Security Advisory for the affected repository, by adding recipients to the draft advisory before it is published. Where a recipient cannot be reached that way, Burnt Labs emails the contact address that recipient registered with [security@burnt.com](mailto:security@burnt.com).

Advance notice is a minimum of seven days before the fix becomes publicly visible. The exception is a vulnerability under active exploitation, where Burnt Labs will ship the fix first and notify as quickly as it can — the calculation changes once attackers already have what the notification would give them.

Where remediation requires a network upgrade, notification precedes the governance proposal. A proposal to upgrade is itself a disclosure that something is worth upgrading for.

### Registering a security contact

Anyone the section above commits us to notifying can register: validators and node operators running XION mainnet or XION testnet, and teams building on an in-scope repository. Email [security@burnt.com](mailto:security@burnt.com) with a contact address and what you run — the network, for a validator or node operator; the repository you consume, for a team building on one. Registration does not affect whether an issue gets fixed. It determines whether you hear about the fix before everyone else does.

## Safe Harbor

Good-faith research within this policy is authorized. Do not exploit beyond confirmation, do not access or disclose user data, do not disrupt production systems, and stop testing immediately if you accidentally access data or systems you did not intend to access.

## No Production Testing

Do not test against XION mainnet, XION testnet, public RPC infrastructure, production web applications, or other public production systems in a way that could affect users, funds, validators, or service availability.

## Rewards

Rewards are determined by demonstrated impact, quality of the proof of concept, and exploitability under real-world constraints. GitHub Security Advisories are used for intake, coordination, disclosure, and credits; reward decisions and payments are handled by Burnt Labs.

### Blockchain / DLT

| Severity | Reward |
| --- | --- |
| Critical | $50,000 – $250,000 |
| High | $20,000 |
| Medium | $5,000 |
| Low | $1,000 |

### Core Protocol Contracts

| Severity | Reward |
| --- | --- |
| Critical | $50,000 – $250,000 |
| High | $20,000 |
| Medium | $5,000 |
| Low | $1,000 |

### Websites and Applications

| Severity | Reward |
| --- | --- |
| Critical | $5,000 – $25,000 |
| High | $2,500 |
| Medium | $1,500 |
| Low | $500 |

## Program Tracks

### Blockchain / DLT

This program covers vulnerabilities in the XION chain node and supporting protocol infrastructure: custom Cosmos SDK modules, the abstract account system, and the ZK cryptographic library.

In-scope repositories:

- [`burnt-labs/xion`](https://github.com/burnt-labs/xion)
- [`burnt-labs/abstract-account`](https://github.com/burnt-labs/abstract-account)
- [`burnt-labs/barretenberg-go`](https://github.com/burnt-labs/barretenberg-go)

Scope applies to the current mainnet release. Vulnerabilities affecting only deprecated or end-of-life versions are not eligible. Vulnerabilities present in previous releases that have been remediated in the current mainnet deployment are not eligible, regardless of whether the fix was publicly announced. Reporters are responsible for verifying exploitability against the currently deployed version before submission.

#### Blockchain / DLT Severity Definitions

**Critical** — Direct, permanent, and irrecoverable theft or loss of user funds at protocol scale. Unauthorized minting. Chain halt or consensus failure requiring a hard fork to resolve. Complete bypass of abstract account authentication enabling arbitrary transaction authorization.

**High** — Theft or freezing of user funds affecting individual accounts. Significant authentication bypass with demonstrated exploitability.

**Medium** — Limited fund loss or temporary disruption requiring specific preconditions. Attacks requiring privileged-party cooperation or colluding action are capped at Medium regardless of theoretical downstream impact. Partial authentication bypass requiring secondary conditions.

**Low** — A valid, reproducible code-level issue with no direct risk to funds or chain safety, but representing a meaningful hardening opportunity. Must include a specific code reference and a clear explanation of why the improvement reduces security risk.

Authentication weaknesses where the impact is limited to accounts created after the attack is established — and which cannot affect the funds, state, or authentication of any account that was funded and operational before the attack began — are capped at Medium regardless of the authentication mechanism involved. A valid High or Critical authentication report must demonstrate unauthorized impact on a pre-existing funded account.

#### Blockchain / DLT Proof of Concept Requirements

End-to-end proof of concept is required. Unit tests using `setupKeeper(t)` or similar test harnesses that bypass transaction encoding, routing, and the ante handler chain are not sufficient to demonstrate on-chain exploitability.

The proof of concept must run against a locally running XION blockchain node configured with mainnet parameters. This is the same setup used in the end-to-end test suite in this repository: a full node running with the XION ante handler chain, module set, and governance configuration matching mainnet. The attack must be executed via standard transaction broadcast, such as `BroadcastTxSync` or an equivalent RPC call, against that node. Simulated environments that model chain state without running a full node are not accepted.

### Core Protocol Contracts

This program covers vulnerabilities in the core smart contracts deployed on XION: the MetaAccount contract and the treasury. These contracts are governance-deployed and form the foundation of XION's account abstraction and fee infrastructure.

In-scope contracts:

- [`contracts/account`](https://github.com/burnt-labs/contracts/tree/main/contracts/account)
- [`contracts/treasury`](https://github.com/burnt-labs/contracts/tree/main/contracts/treasury)

Treasury vulnerabilities in fee grant issuance functions are eligible only where the attack results in direct, uncapped extraction of funds from the treasury contract's XION balance to an attacker-controlled address without requiring privileged setup. Reports targeting bounded fee grant operations — grants issued with explicit allowance limits and expiration enforced at the contract level — are not eligible, because the grant-issuance design intentionally delegates authorization to the calling application layer.

Scope is limited exclusively to the two contracts linked above. All other contracts in the repository — including marketplace, asset, NFT, and other contracts — are not in scope under this program. Scope applies to contracts deployed on the current mainnet. Vulnerabilities affecting only deprecated deployments are not eligible. Vulnerabilities present in previous contract deployments that have been remediated in the currently deployed bytecode are not eligible, regardless of whether the fix was publicly announced. Reporters are responsible for verifying exploitability against the current deployed contract version before submission.

#### Core Protocol Contract Severity Definitions

**Critical** — Direct, permanent, and irrecoverable theft or loss of funds held in or routed through in-scope contracts at meaningful scale. Complete bypass of MetaAccount authentication enabling unauthorized transaction execution, where the proof of concept demonstrates actual movement of funds from a pre-existing victim account to an attacker-controlled address using only attacker-controlled keys. Permanent contract state corruption with no recovery path.

**High** — Theft or freezing of funds affecting individual accounts. Authentication bypass with demonstrated exploitability against an existing account. Permanent disruption of core contract functionality.

**Medium** — Limited fund loss requiring specific preconditions. Attacks requiring a privileged party — contract admin or governance — to take self-destructive or colluding action are capped at Medium regardless of downstream impact. Temporary disruption recoverable by governance.

**Low** — A valid, reproducible code-level issue with no direct risk to funds, but representing a meaningful hardening opportunity. Must include a specific code reference and a clear explanation of the improvement.

#### Core Protocol Contract Proof of Concept Requirements

End-to-end proof of concept is required. Tests that mock contract state or bypass CosmWasm message routing — including `cw-multi-test` environments and harnesses that stub bank, staking, or IBC modules — are not sufficient to demonstrate exploitability.

The proof of concept must run against a locally running XION blockchain node configured with mainnet parameters. This is the same setup used in the end-to-end test suite in this repository: a full node running with the governance-deployed contract bytecode, the XION ante handler chain, and the module configuration matching mainnet. The attack must be executed via standard transaction broadcast against that node. Simulated environments that model contract state without running a full node are not accepted.

### Websites and Applications

This program covers vulnerabilities in production web properties operated by Burnt Labs, including the XION web app and associated user-facing interfaces, where a vulnerability could result in user harm, credential compromise, or unauthorized transaction execution.

Developer portals, dashboard frontends, third-party services, and third-party infrastructure are not in scope unless explicitly identified in writing by Burnt Labs.

#### Websites and Applications Severity Definitions

**Critical** — Full account takeover or unauthorized transaction execution that does not require the victim to explicitly approve or confirm a security-sensitive action. The attack must succeed through passive exploitation, such as page load or background request, or by embedding the attack within what appears to the victim as a routine, expected interaction where the victim takes no step they would reasonably identify as security-relevant. Reports where the victim must be tricked into explicitly granting access, authorizing a transaction, or clicking through a confirmation are not Critical.

**High** — Account or session compromise requiring limited, non-suspicious user interaction such as visiting a link or loading a page. Cross-site scripting with demonstrated ability to initiate or manipulate transactions. Significant data exposure affecting individual users.

**Medium** — Attacks that require the victim to take a meaningful, security-relevant action to trigger, such as clicking through a confirmation, explicitly granting access, or following multi-step instructions. Limited data exposure or access control bypass requiring specific preconditions. Cross-site request forgery with demonstrated impact on account state.

**Low** — Valid security issue with no direct risk to accounts or user data but representing a meaningful hardening opportunity. Must include a clear reproduction path and explanation of the risk.

#### Websites and Applications Proof of Concept Requirements

All reports must include a proof of concept demonstrating the vulnerability against a staging or other non-production environment, consistent with the No Production Testing policy above. Where no staging target exists for the affected system, include a local reproduction and note the limitation in the report. Screenshots or video walkthroughs showing end-to-end exploitation are expected for High and Critical severity reports.

Reports consisting only of automated scanner output without demonstrated exploitability will not be rewarded.

## Permissioned Chain Policy

XION mainnet operates with `code_upload_access: Nobody`. Contract deployment requires a governance proposal. This is a fundamental architectural constraint, not a bypass target.

Any attack vector that requires an attacker to deploy a malicious contract on mainnet is out of scope, regardless of technical validity. This includes amplification attacks through attacker-deployed contracts, exploit chains initiated from attacker-deployed contracts, and any narrative beginning with "an attacker deploys a contract that..."

## Privileged Actor Policy

Attacks that require governance, module authority, validators, contract admins, or other privileged parties to take self-destructive or colluding action are classified at Medium at most, regardless of downstream impact. The threat model assumes privileged actors operate within specified protocol parameters and according to their roles.

## Out of Scope

### Assets

- Third-party infrastructure, RPC providers, and external dependencies
- Public blockchain RPC, REST, gRPC, and Tendermint RPC endpoints that expose blockchain state by design and are operated by validators and node operators as a public service
- Upstream dependencies, including CosmWasm, the Cosmos SDK, IBC, and the Barretenberg C library, unless the vulnerability originates in in-scope Burnt Labs code
- Third-party contracts deployed on XION by external teams
- Example, demo, marketplace, asset, NFT, and other contracts outside the two in-scope core protocol contracts
- Developer portals, dashboard frontends, third-party services, and third-party infrastructure unless explicitly identified in writing by Burnt Labs

### Vulnerability Classes

- Attacks requiring malicious contract deployment on mainnet
- Governance attacks requiring a malicious proposal to pass
- Denial of service attacks of any form, including single-transaction resource exhaustion, node crashes, and chain halts recoverable by software patch, coordinated validator restart, or governance parameter update. Chain halts that require a hard fork to resolve remain in scope under the Critical severity definition.
- Denial of service requiring sustained attacker resource expenditure proportional to harm
- Treasury fee grant issuance where grants are bounded by explicit allowance limits and expiration, without demonstrating direct uncapped extraction of funds from the treasury's XION balance to an attacker-controlled address
- Clickjacking where transaction signing provides a second confirmation layer
- Open redirects after authentication that do not result in token or credential leakage
- Self-XSS requiring the attacker to execute code in their own browser session
- Issues requiring physical access to the victim's device
- Social engineering
- Missing security headers where no exploitable impact can be demonstrated
- Theoretical vulnerabilities without a working end-to-end proof of concept
- Attacks in which the cost incurred by the attacker to execute the exploit exceeds the demonstrable harm caused to the protocol or its users
- Reports submitted at a severity level that does not match this policy's severity definitions. Severity is assessed by Burnt Labs based on demonstrated impact under real-world constraints. Reports submitted at an incorrect or inflated severity are not eligible as submitted and will be closed. Burnt Labs does not reclassify, adjust, or negotiate severity on behalf of reporters. Reporters are responsible for accurate severity assessment at submission.
- Best practices, gas optimizations, missing events, and informational findings

## Frequently Raised Non-Issues

The following design decisions are sometimes reported as vulnerabilities but are intentional and will not be changed.

### DKIM public keys stored on-chain

The `x/dkim` module stores RSA public keys on-chain. These are public keys — the same data that any mail server operator publishes in DNS TXT records. Storing them on-chain enables trustless DKIM verification inside ZK circuits and is a core feature of the XION email-based account system. There is no private key material stored anywhere in the module or on-chain state.

### Bank `MsgSend` platform fee exemption is a governance parameter

The platform fee applied to `MsgSend` transactions can be set to zero for specific addresses, such as protocol contracts, via a governance parameter. This is an intentional administrative mechanism, not a privilege escalation. Any change to the exemption list requires an on-chain governance vote and is fully auditable in the transaction history.

## Recognition

We appreciate responsible disclosure and will credit security researchers who help improve XION security. Recognition may be included in GitHub Security Advisories, release notes, and public security bulletins after coordinated disclosure.
