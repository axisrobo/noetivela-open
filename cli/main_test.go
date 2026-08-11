package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/axisrobo/noetivela-open/sdk/go/client"
)

func TestCLIValidateEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/contracts/validate" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"valid":true,"satisfiable":true,"policy":"legal","candidates":[{"model_ref":"m1","endpoint_ref":"e1","eligible":true}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// Write a contract file.
	dir := t.TempDir()
	ctPath := filepath.Join(dir, "contract.json")
	raw, _ := json.Marshal(client.Contract{Task: "contract_clause_extraction", Modality: []string{"text"}})
	if err := os.WriteFile(ctPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	c := client.New(srv.URL)
	fs := []string{"validate", "-contract", ctPath, "-policy", "legal"}
	if err := runValidate(context.Background(), c, fs); err != nil {
		t.Fatalf("runValidate failed: %v", err)
	}
}
