package main

import (
	"context"
	"testing"

	vaultv1 "OlympusGCP-Vault/40000-Communication-Contracts/430-Protocol-Definitions/000-gen/vault/v1"
	"connectrpc.com/connect"
)

func TestVaultServer(t *testing.T) {
	tempDir := t.TempDir()
	server := NewVaultServer(tempDir)

	ctx := context.Background()

	// Test Write
	writeReq := connect.NewRequest(&vaultv1.VaultWriteRequest{
		Key:   "test-key",
		Value: "test-value",
	})
	_, err := server.VaultWrite(ctx, writeReq)
	if err != nil {
		t.Fatalf("VaultWrite failed: %v", err)
	}

	// Test Read
	readReq := connect.NewRequest(&vaultv1.VaultReadRequest{
		Key: "test-key",
	})
	readRes, err := server.VaultRead(ctx, readReq)
	if err != nil {
		t.Fatalf("VaultRead failed: %v", err)
	}
	if readRes.Msg.Value != "test-value" {
		t.Errorf("Expected value 'test-value', got '%s'", readRes.Msg.Value)
	}

	// Test Read Non-existent
	readReqMissing := connect.NewRequest(&vaultv1.VaultReadRequest{
		Key: "missing-key",
	})
	_, err = server.VaultRead(ctx, readReqMissing)
	if err == nil {
		t.Error("Expected error for missing key, got nil")
	}

	// Test persistence
	server2 := NewVaultServer(tempDir)
	readRes2, err := server2.VaultRead(ctx, readReq)
	if err != nil {
		t.Fatalf("Persistence check failed: %v", err)
	}
	if readRes2.Msg.Value != "test-value" {
		t.Errorf("Persistence failed: expected 'test-value', got '%s'", readRes2.Msg.Value)
	}
}

func TestListSecrets(t *testing.T) {
	tempDir := t.TempDir()
	server := NewVaultServer(tempDir)
	ctx := context.Background()

	keys := []string{"app/db", "app/api", "web/ui"}
	for _, k := range keys {
		server.VaultWrite(ctx, connect.NewRequest(&vaultv1.VaultWriteRequest{Key: k, Value: "secret"}))
	}

	// Test list with prefix
	listReq := connect.NewRequest(&vaultv1.ListSecretsRequest{Prefix: "app/"})
	listRes, err := server.ListSecrets(ctx, listReq)
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}

	if len(listRes.Msg.Keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(listRes.Msg.Keys))
	}
}
