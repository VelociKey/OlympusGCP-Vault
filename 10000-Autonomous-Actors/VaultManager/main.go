package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	vaultv1 "OlympusGCP-Vault/40000-Communication-Contracts/430-Protocol-Definitions/000-gen/vault/v1"
	"OlympusGCP-Vault/40000-Communication-Contracts/430-Protocol-Definitions/000-gen/vault/v1/vaultv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type SecretHistory struct {
	Versions []string `json:"versions"`
}

type VaultServer struct {
	mu          sync.RWMutex
	secrets     map[string]string
	history     map[string][]string // key -> history of values
	storagePath string
}

func NewVaultServer(storageDir string) *VaultServer {
	path := filepath.Join(storageDir, "secrets.json")
	s := &VaultServer{
		secrets:     make(map[string]string),
		history:     make(map[string][]string),
		storagePath: path,
	}
	s.load()
	return s
}

func (s *VaultServer) load() {
	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		slog.Info("No existing secrets found, starting fresh", "path", s.storagePath)
		return
	}
	
	var store struct {
		Secrets map[string]string   `json:"secrets"`
		History map[string][]string `json:"history"`
	}
	
	if err := json.Unmarshal(data, &store); err != nil {
		slog.Error("Failed to unmarshal secrets", "error", err)
		return
	}
	s.secrets = store.Secrets
	s.history = store.History
}

func (s *VaultServer) save() {
	store := struct {
		Secrets map[string]string   `json:"secrets"`
		History map[string][]string `json:"history"`
	}{
		Secrets: s.secrets,
		History: s.history,
	}
	
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal secrets", "error", err)
		return
	}
	if err := os.WriteFile(s.storagePath, data, 0644); err != nil {
		slog.Error("Failed to save secrets", "error", err)
	}
}

func (s *VaultServer) VaultWrite(ctx context.Context, req *connect.Request[vaultv1.VaultWriteRequest]) (*connect.Response[vaultv1.VaultWriteResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := req.Msg.Key
	val := req.Msg.Value
	slog.Info("VaultWrite", "key", key)
	
	s.secrets[key] = val
	s.history[key] = append(s.history[key], val)
	s.save()

	version := int32(len(s.history[key]))
	return connect.NewResponse(&vaultv1.VaultWriteResponse{Version: version}), nil
}

func (s *VaultServer) VaultRead(ctx context.Context, req *connect.Request[vaultv1.VaultReadRequest]) (*connect.Response[vaultv1.VaultReadResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slog.Info("VaultRead", "key", req.Msg.Key)
	val, ok := s.secrets[req.Msg.Key]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("secret not found: %s", req.Msg.Key))
	}

	version := int32(len(s.history[req.Msg.Key]))
	return connect.NewResponse(&vaultv1.VaultReadResponse{Value: val, Version: version}), nil
}

func (s *VaultServer) GetSecretVersion(ctx context.Context, req *connect.Request[vaultv1.GetSecretVersionRequest]) (*connect.Response[vaultv1.VaultReadResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slog.Info("GetSecretVersion", "key", req.Msg.Key, "version", req.Msg.Version)
	history, ok := s.history[req.Msg.Key]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("secret not found: %s", req.Msg.Key))
	}

	idx := int(req.Msg.Version) - 1
	if idx < 0 || idx >= len(history) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("version %d not found for %s", req.Msg.Version, req.Msg.Key))
	}

	return connect.NewResponse(&vaultv1.VaultReadResponse{Value: history[idx], Version: req.Msg.Version}), nil
}

func (s *VaultServer) ListSecretVersions(ctx context.Context, req *connect.Request[vaultv1.ListSecretVersionsRequest]) (*connect.Response[vaultv1.ListSecretVersionsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slog.Info("ListSecretVersions", "key", req.Msg.Key)
	history, ok := s.history[req.Msg.Key]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("secret not found: %s", req.Msg.Key))
	}

	var versions []int32
	for i := range history {
		versions = append(versions, int32(i+1))
	}

	return connect.NewResponse(&vaultv1.ListSecretVersionsResponse{Versions: versions}), nil
}

func (s *VaultServer) ListSecrets(ctx context.Context, req *connect.Request[vaultv1.ListSecretsRequest]) (*connect.Response[vaultv1.ListSecretsResponse], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slog.Info("ListSecrets", "prefix", req.Msg.Prefix)
	var keys []string
	for k := range s.secrets {
		if req.Msg.Prefix == "" || (len(k) >= len(req.Msg.Prefix) && k[:len(req.Msg.Prefix)] == req.Msg.Prefix) {
			keys = append(keys, k)
		}
	}

	return connect.NewResponse(&vaultv1.ListSecretsResponse{Keys: keys}), nil
}

func (s *VaultServer) TestIAMPolicy(ctx context.Context, req *connect.Request[vaultv1.TestIAMPolicyRequest]) (*connect.Response[vaultv1.TestIAMPolicyResponse], error) {
	slog.Info("TestIAMPolicy", "identity", req.Msg.Identity, "action", req.Msg.Action)
	return connect.NewResponse(&vaultv1.TestIAMPolicyResponse{
		Allowed: true,
		Reason:  "YOLO Mode: All access granted",
	}), nil
}

func main() {
	storageDir := "../../60000-Information-Storage/VaultData"
	server := NewVaultServer(storageDir)
	mux := http.NewServeMux()
	path, handler := vaultv1connect.NewVaultServiceHandler(server)
	mux.Handle(path, handler)

	port := "8092"
	slog.Info("VaultManager starting", "port", port)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 3 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		slog.Error("Server failed", "error", err)
	}
}
