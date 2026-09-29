package system

import (
	"os"
	"strings"
	"testing"
)

func TestUnsupportedServiceManagement(t *testing.T) {
	if IsLinux() {
		t.Skip("non-Linux behavior")
	}
	for _, action := range []func() error{DefaultServiceController.Start, DefaultServiceController.Stop, DefaultServiceController.Restart, DefaultServiceController.Reload, DefaultServiceController.Enable, DefaultServiceController.Disable} {
		if err := action(); err == nil {
			t.Fatal("unsupported service operation reported success")
		}
	}
}

func TestConfigValidationDoesNotReportFakeSuccess(t *testing.T) {
	if IsLinux() && fileExists("/etc/clickhouse-server") {
		t.Skip("host has a real ClickHouse configuration")
	}
	t.Chdir(t.TempDir())
	t.Setenv("PATH", t.TempDir())
	if _, err := DefaultServiceController.ValidateConfig(); err == nil {
		t.Fatal("missing configuration reported success")
	}
	if err := os.MkdirAll("data/clickhouse_config", 0755); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{"", "<a/><b/>", "text<a/>", "<clickhouse/>"} {
		if err := os.WriteFile("data/clickhouse_config/config.xml", []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := DefaultServiceController.ValidateConfig()
		if err == nil {
			t.Fatalf("reported success without a validator: %q", content)
		}
		if content == "<clickhouse/>" && !strings.Contains(err.Error(), "Unsupported") {
			t.Fatalf("missing validator not identified: %v", err)
		}
	}
}
