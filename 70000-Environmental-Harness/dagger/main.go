package main

import (
	"context"
	"dagger/olympusgcp-vault/internal/dagger"
)

type OlympusGCPVault struct{}

func (m *OlympusGCPVault) HelloWorld(ctx context.Context) string {
	return "Hello from OlympusGCP-Vault!"
}

func main() {
	dagger.Serve()
}
