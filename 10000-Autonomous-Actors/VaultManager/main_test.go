package main

import (
	"context"
	"testing"

	vaultv1 "OlympusGCP-Vault/40000-Communication-Contracts/430-Protocol-Definitions/000-gen/vault/v1"
	"connectrpc.com/connect"
)

func TestVaultServerAdvanced(t *testing.T) {
	tempDir := t.TempDir()
	server := NewVaultServer(tempDir)
	ctx := context.Background()

	key := "config/db"
	
	// Write Version 1
	res1, _ := server.VaultWrite(ctx, connect.NewRequest(&vaultv1.VaultWriteRequest{Key: key, Value: "pass1"}))
	if res1.Msg.Version != 1 {
		t.Errorf("Expected version 1, got %d", res1.Msg.Version)
	}

	// Write Version 2
	res2, _ := server.VaultWrite(ctx, connect.NewRequest(&vaultv1.VaultWriteRequest{Key: key, Value: "pass2"}))
	if res2.Msg.Version != 2 {
		t.Errorf("Expected version 2, got %d", res2.Msg.Version)
	}

	// Read Current
	readRes, _ := server.VaultRead(ctx, connect.NewRequest(&vaultv1.VaultReadRequest{Key: key}))
	if readRes.Msg.Value != "pass2" {
		t.Errorf("Expected current value 'pass2', got '%s'", readRes.Msg.Value)
	}

	// Read Version 1
	ver1Res, _ := server.GetSecretVersion(ctx, connect.NewRequest(&vaultv1.GetSecretVersionRequest{Key: key, Version: 1}))
	if ver1Res.Msg.Value != "pass1" {
		t.Errorf("Expected version 1 value 'pass1', got '%s'", ver1Res.Msg.Value)
	}

	// List Versions
	listRes, _ := server.ListSecretVersions(ctx, connect.NewRequest(&vaultv1.ListSecretVersionsRequest{Key: key}))
	if len(listRes.Msg.Versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(listRes.Msg.Versions))
	}
}
