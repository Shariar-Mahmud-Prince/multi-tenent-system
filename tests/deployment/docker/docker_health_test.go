package docker_test

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestDocker_ContainersRunningAndHealthy(t *testing.T) {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Status}}\t{{.Ports}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("docker command not accessible: %v", err)
	}

	lines := strings.Split(string(out), "\n")
	foundContainers := map[string]bool{
		"tenant_postgres": false,
		"tenant_redis":    false,
		"tenant_app":      false,
		"tenant_caddy":    false,
	}

	for _, line := range lines {
		for name := range foundContainers {
			if strings.Contains(line, name) {
				foundContainers[name] = true

				// Assert private network security rules
				if name == "tenant_redis" {
					// Redis should NEVER be exposed on 0.0.0.0
					if strings.Contains(line, "0.0.0.0:6379") {
						t.Errorf("SECURITY RISK: tenant_redis is exposed on 0.0.0.0 public interface!")
					}
				}
				if name == "tenant_postgres" {
					// Postgres should NEVER be exposed on 0.0.0.0:5432
					if strings.Contains(line, "0.0.0.0:5432") {
						t.Errorf("SECURITY RISK: tenant_postgres is exposed on 0.0.0.0:5432 public interface!")
					}
				}
			}
		}
	}

	for name, running := range foundContainers {
		if !running {
			t.Errorf("CRITICAL CONTAINER MISSING: container %q is not running", name)
		}
	}
}

func TestDocker_ApplicationHealthzContract(t *testing.T) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:8080/healthz")
	if err != nil {
		t.Fatalf("failed to query app healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /healthz, got %d", resp.StatusCode)
	}

	var payload struct {
		Status   string `json:"status"`
		Database string `json:"database"`
		Redis    bool   `json:"redis"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("invalid healthz JSON: %v", err)
	}

	if payload.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", payload.Status)
	}
	if payload.Database != "connected" {
		t.Errorf("expected database 'connected', got %q", payload.Database)
	}
	if !payload.Redis {
		t.Errorf("expected redis 'true', got %v", payload.Redis)
	}
}
