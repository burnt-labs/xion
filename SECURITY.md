# Security Policy

## Introduction

Security researchers are essential in identifying vulnerabilities that may impact the Xion Network. If you have discovered a security vulnerability in the Xion chain or any repository managed by Burnt Labs, we encourage you to notify us using one of the methods outlined below.

We take all security bugs seriously. If confirmed upon investigation, we will patch it within a reasonable amount of time and release a public security bulletin discussing the impact and credit the discoverer.

## Standard Priority Bug

For a bug that is non-sensitive and/or operational in nature rather than a critical vulnerability, please add it as a [GitHub issue](https://github.com/burnt-labs/xion/issues/new).

## Critical Bug or Security Issue

If you're here because you're trying to figure out how to notify us of a security issue, please use one of the following methods:

* **Email**: [security@burnt.com](mailto:security@burnt.com)

Please avoid opening public issues on GitHub that contain information about a potential security vulnerability as this makes it difficult to reduce the impact and harm of valid security issues.

## Submit Vulnerability Report

When reporting a vulnerability, please include the following details to aid in our assessment:

- Type of vulnerability
- Description of the vulnerability
- Steps to reproduce the issue
- Impact of the issue
- Explanation of how an attacker could exploit it
- Any potential mitigations or workarounds

## Coordinated Vulnerability Disclosure Policy

We ask security researchers to keep vulnerabilities and communications around vulnerability submissions private and confidential until a patch is developed. In addition to this, we ask that you:

- Allow us a reasonable amount of time to correct or address security vulnerabilities
- Avoid exploiting any vulnerabilities that you discover
- Demonstrate good faith by not disrupting or degrading Xion's network, data, or services
- Refrain from testing vulnerabilities on our publicly accessible environments, including but not limited to:
  - Xion mainnet
  - Xion testnet
  - Public-facing applications and services

## Vulnerability Disclosure Process

Xion uses the following disclosure process:

1. **Initial Report**: Submit your vulnerability report via email or GitHub Security
2. **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
3. **Investigation**: Our security team will investigate and confirm the vulnerability
4. **Assessment**: We will evaluate the vulnerability and inform you of its severity and the estimated time frame for resolution
5. **Fix Development**: We will develop and test a fix for the vulnerability in private repositories
6. **Coordination**: For critical issues, we will coordinate with affected parties and the CosmWasm community. Critical vulnerabilities affecting CosmWasm components will be reported to the CosmWasm security team through their non-public channels before public disclosure
7. **Community Notification**: We notify the community that a security release is coming, to give users and validators time to prepare their systems for the update. Notifications can include Discord messages, tweets, and emails to partners and validators
8. **Public Disclosure**: After a fix is deployed, we will publish a security bulletin with details and credit. Once releases are available, we notify the community again through the same channels

This process can take some time. Every effort will be made to handle the bug in as timely a manner as possible. However, it's important that we follow the process described above to ensure that disclosures are handled consistently and to keep Xion and the projects running on it secure.

Should a security issue require a network upgrade, additional time may be needed to raise a governance proposal and complete the upgrade.

## Severity Characterization

| Severity     | Description                                                             |
|--------------|-------------------------------------------------------------------------|
| **CRITICAL** | Immediate threat to critical systems (e.g. funds at risk, network compromise) |
| **HIGH**     | Significant impact on major functionality or security controls         |
| **MEDIUM**   | Impacts minor features or exposes non-sensitive data                    |
| **LOW**      | Minimal impact or informational issues                                  |

## Scope

This security policy applies to:
- Xion Daemon (xiond)
- All CosmWasm-related components
- Smart contract execution environment
- All modules and dependencies within the Xion blockchain
- All repositories managed by Burnt Labs for the Xion ecosystem

## Commitment to CosmWasm Community

We are committed to sharing security issues and bugs with the CosmWasm community. Critical vulnerabilities affecting CosmWasm components will be reported to the CosmWasm security team through their non-public channels before public disclosure.

## Recognition

We appreciate responsible disclosure and will credit security researchers who help us improve the security of Xion. Recognition will be included in our security bulletins and may be featured in our communications.

