package config

import (
	"os"
	"path/filepath"
	"testing"

	"clickhouse-manager/internal/database"
	"clickhouse-manager/internal/system"
)

func TestXMLValidation(t *testing.T) {
	for _, tc := range []struct {
		content string
		valid   bool
	}{
		{"<clickhouse><tcp_port>9000</tcp_port></clickhouse>", true},
		{"", false}, {"<!-- comment -->", false}, {"<a/><b/>", false},
		{"<clickhouse>", false}, {"<a></b>", false}, {"outside<a/>", false},
	} {
		if got := ValidateXMLContent(tc.content); got.Valid != tc.valid {
			t.Errorf("%q: %+v", tc.content, got)
		}
	}
	result := ValidateXMLContent("<clickhouse>\n<port>9000</other>\n</clickhouse>")
	if result.Line != 2 {
		t.Fatalf("expected line 2, got %+v", result)
	}
}

func TestConfigRejectsNonlocalPaths(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, path := range []string{"../outside.xml", filepath.Join(t.TempDir(), "outside.xml")} {
		if _, err := ReadConfigFile(path); err == nil {
			t.Errorf("read accepted %s", path)
		}
		if err := SaveConfigFile(path, "<clickhouse/>", "test", "admin"); err == nil {
			t.Errorf("save accepted %s", path)
		}
		if _, err := ListConfigBackups(path); err == nil {
			t.Errorf("backup list accepted %s", path)
		}
	}
}

func TestConfigRestoresAfterValidationFailure(t *testing.T) {
	if !system.IsLinux() {
		t.Skip("server configuration validation runs on Linux")
	}
	t.Chdir(t.TempDir())
	t.Setenv("PATH", t.TempDir())
	dir := GetConfigDir()
	path := filepath.Join(dir, "config.xml")
	original := "<clickhouse><tcp_port>9000</tcp_port></clickhouse>"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfigFile("config.xml", "<clickhouse/>", "test", "admin"); err == nil {
		t.Fatal("missing validator should fail")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != original {
		t.Fatalf("original config not restored: %q, %v", content, err)
	}
	if err := SaveConfigFile("new.xml", "<clickhouse/>", "test", "admin"); err == nil {
		t.Fatal("missing validator should fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "new.xml")); !os.IsNotExist(err) {
		t.Fatalf("invalid new file left behind: %v", err)
	}
}

func TestRollbackUsesOriginalPath(t *testing.T) {
	if system.IsLinux() {
		t.Skip("successful save requires an installed server validator on Linux")
	}
	t.Chdir(t.TempDir())
	if err := database.InitDB("panel.db"); err != nil {
		t.Fatal(err)
	}
	db, _ := database.DB.DB()
	defer db.Close()
	dir := GetConfigDir()
	path := filepath.Join(dir, "config.xml")
	backup := database.ConfigBackup{FilePath: path, Content: "<clickhouse><tcp_port>9000</tcp_port></clickhouse>"}
	if err := database.DB.Create(&backup).Error; err != nil {
		t.Fatal(err)
	}
	if err := RollbackConfig(backup.ID, "admin"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != backup.Content {
		t.Fatalf("rollback missed original file: %q, %v", content, err)
	}
}
