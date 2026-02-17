# OlympusGCP-Vault

High-intent security cluster for the Olympus fleet, providing Secret Management, KMS, and IAM intent-based tools.

## Domain: 01000 (Security & Identity)

## Philosophy: MCP-Native
This cluster rejects wire-compatibility with GCP APIs. Instead, it provides a Go-based **MCP Vault Bridge** that exposes semantic tools:
*   `vault_read(key)`: Fetches secret values from local `.jebnf` (local) or GCP Secret Manager (prod).
*   `vault_write(key, value)`: Standardized secret updates.

## Storage
- Local secrets are stored in `60000-Information-Storage/900-LocalVault/secrets.jebnf`.
