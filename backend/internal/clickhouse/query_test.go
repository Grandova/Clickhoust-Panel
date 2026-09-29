package clickhouse

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"clickhouse-manager/internal/config"
)

func TestQueryClassification(t *testing.T) {
	for _, sql := range []string{"SELECT 1;", "WITH 1 AS x SELECT x", "-- comment\nSELECT 1", "/* comment */ EXPLAIN SELECT 1", "DESC system.one"} {
		if !isSelectRegex.MatchString(sql) {
			t.Errorf("misclassified query: %s", sql)
		}
	}
	for _, sql := range []string{"INSERT INTO t VALUES (1)", "CREATE TABLE t (n UInt8) ENGINE=Memory", "SELECTED"} {
		if isSelectRegex.MatchString(sql) {
			t.Errorf("misclassified command: %s", sql)
		}
	}
}

func TestIPv6Port(t *testing.T) {
	listener, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 unavailable: %v", err)
	}
	defer listener.Close()
	if !CheckPortOpen("::1", listener.Addr().(*net.TCPAddr).Port, time.Second) {
		t.Fatal("IPv6 listener was not detected")
	}
}

func TestUnsupportedProtocol(t *testing.T) {
	svc := &ClickHouseService{config: ConnectionConfig{Host: "127.0.0.1", Port: 9000, Protocol: "unsupported"}}
	if _, err := svc.GetConn(context.Background()); err == nil {
		t.Fatal("unsupported protocol accepted")
	}
}

func TestClickHouseIntegration(t *testing.T) {
	host := os.Getenv("CH_TEST_HOST")
	if host == "" {
		t.Skip("set CH_TEST_HOST to run against an independent ClickHouse server")
	}
	port, err := strconv.Atoi(os.Getenv("CH_TEST_PORT"))
	if err != nil {
		t.Fatal(err)
	}
	svc := &ClickHouseService{config: ConnectionConfig{Host: host, Port: port, User: os.Getenv("CH_TEST_USER"), Database: "default", Protocol: os.Getenv("CH_TEST_PROTOCOL"), Secure: os.Getenv("CH_TEST_SECURE") == "1"}}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	conn, err := svc.GetConn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, sql := range []string{"SELECT 1 AS value;", "WITH 2 AS value SELECT value", "-- comment\nSELECT 3 AS value", "/* comment */ SELECT 4 AS value", "EXPLAIN SYNTAX SELECT 1", "DESCRIBE TABLE system.one", "SHOW CREATE TABLE system.one"} {
		t.Run(sql, func(t *testing.T) {
			result, err := svc.Query(ctx, sql, 10)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Columns) == 0 || len(result.Rows) == 0 {
				t.Fatalf("missing result: %+v", result)
			}
		})
	}
	t.Run("metadata and monitoring", func(t *testing.T) {
		if _, err := svc.ListDatabases(ctx); err != nil {
			t.Error("databases:", err)
		}
		if _, err := svc.ListTables(ctx, "system"); err != nil {
			t.Error("tables:", err)
		}
		if _, err := svc.GetColumns(ctx, "system", "one"); err != nil {
			t.Error("columns:", err)
		}
		if _, err := svc.GetProcesses(ctx); err != nil {
			t.Error("processes:", err)
		}
		if _, err := svc.GetDisks(ctx); err != nil {
			t.Error("disks:", err)
		}
		if _, err := svc.GetClusters(ctx); err != nil {
			t.Error("clusters:", err)
		}
		if _, err := svc.GetReplicas(ctx); err != nil {
			t.Error("replicas:", err)
		}
		if _, err := svc.GetDictionaries(ctx); err != nil {
			t.Error("dictionaries:", err)
		}
		if _, err := svc.GetParts(ctx, "system", "", false); err != nil {
			t.Error("parts:", err)
		}
		if _, err := svc.GetMerges(ctx); err != nil {
			t.Error("merges:", err)
		}
		if _, err := svc.GetMutations(ctx, "system", ""); err != nil {
			t.Error("mutations:", err)
		}
		if _, err := svc.ListUsers(ctx); err != nil {
			t.Error("users:", err)
		}
		if _, err := svc.ListProfiles(ctx); err != nil {
			t.Error("profiles:", err)
		}
		if _, err := svc.GetMetrics(ctx); err != nil {
			t.Error("metrics:", err)
		}
		if _, err := svc.GetData(ctx, BrowserDataRequest{Database: "system", Table: "one", Page: 1, PageSize: 100}); err != nil {
			t.Error("browse:", err)
		}
		if err := svc.Execute(ctx, "SYSTEM FLUSH LOGS"); err != nil {
			t.Error("flush logs:", err)
		}
		if _, _, err := svc.GetQueryLog(ctx, 0, 10, 0, ""); err != nil {
			t.Error("query log:", err)
		}
	})

	t.Run("database user permissions", func(t *testing.T) {
		name := "panel_user_" + strconv.FormatInt(time.Now().UnixNano(), 10)
		manager := name + "_manager"
		if err := svc.CreateDatabase(ctx, name, "Atomic", ""); err != nil {
			t.Fatal(err)
		}
		defer svc.DropDatabase(context.Background(), name)
		for _, user := range []string{name, manager} {
			if err := svc.CreateUser(ctx, CreateUserRequest{Username: user, Password: "old'password", DefaultDatabase: name}); err != nil {
				t.Fatal(err)
			}
			defer svc.DropUser(context.Background(), user)
		}
		if err := svc.CreateUser(ctx, CreateUserRequest{Username: name}); err == nil {
			t.Fatal("duplicate user creation reported success")
		}
		if err := svc.Execute(ctx, "GRANT SELECT, INSERT, CREATE TABLE ON `"+name+"`.* TO `"+manager+"` WITH GRANT OPTION"); err != nil {
			t.Fatal(err)
		}
		cfg := svc.GetConfig()
		cfg.User, cfg.Password, cfg.Database = manager, "old'password", name
		limited := &ClickHouseService{config: cfg}
		limitedConn, err := limited.GetConn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer limitedConn.Close()
		if err := limited.GrantPrivileges(ctx, GrantRequest{Username: name, Database: "*", Table: "*", Privileges: []string{"ALL"}}); err == nil {
			t.Fatal("limited manager granted ALL")
		}
		req := GrantRequest{Username: name, Database: name, Table: "*", Privileges: []string{"SELECT", "INSERT", "CREATE TABLE"}}
		if err := limited.GrantPrivileges(ctx, req); err != nil {
			t.Fatal("database-scoped grant:", err)
		}
		users, err := svc.ListUsers(ctx)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, user := range users {
			if user.Name == name && strings.Contains(strings.Join(user.Grants, " "), "CREATE TABLE") {
				found = true
			}
		}
		if !found {
			t.Fatal("user grants missing from list")
		}
		cfg.User = name
		client := &ClickHouseService{config: cfg}
		clientConn, err := client.GetConn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer clientConn.Close()
		for _, sql := range []string{"CREATE TABLE online_ip (id UInt64) ENGINE=Memory", "INSERT INTO online_ip VALUES (1)"} {
			if err := client.Execute(ctx, sql); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := client.Query(ctx, "SELECT * FROM online_ip", 10); err != nil {
			t.Fatal(err)
		}
		req.Privileges = []string{"CREATE TABLE"}
		if err := limited.RevokePrivileges(ctx, req); err != nil {
			t.Fatal(err)
		}
		if err := client.Execute(ctx, "CREATE TABLE denied (id UInt64) ENGINE=Memory"); err == nil {
			t.Fatal("revoked CREATE TABLE remained usable")
		}
		if err := svc.AlterUser(ctx, AlterUserRequest{Username: name, NewPassword: "new'password", DefaultDatabase: name}); err != nil {
			t.Fatal("password and database change:", err)
		}
		oldClient := &ClickHouseService{config: cfg}
		if oldConn, err := oldClient.GetConn(ctx); err == nil {
			oldConn.Close()
			t.Fatal("old password remained usable")
		}
		cfg.Password = "new'password"
		newClient := &ClickHouseService{config: cfg}
		newConn, err := newClient.GetConn(ctx)
		if err != nil {
			t.Fatal("new password:", err)
		}
		newConn.Close()
	})

	t.Run("table lifecycle", func(t *testing.T) {
		name := "panel_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := svc.CreateDatabase(ctx, name, "Atomic", "integration test"); err != nil {
			t.Fatal(err)
		}
		defer svc.DropDatabase(context.Background(), name)
		if _, err := svc.VisualCreateTable(ctx, VisualCreateTableRequest{Database: name, TableName: "events", Engine: "MergeTree", Columns: []VisualColumn{{Name: "id", Type: "UInt64"}, {Name: "value", Type: "String"}}, OrderBy: "id", ExecuteNow: true}); err != nil {
			t.Fatal(err)
		}
		if err := svc.Execute(ctx, "INSERT INTO `"+name+"`.events VALUES (1, 'first'), (2, 'second')"); err != nil {
			t.Fatal(err)
		}
		data, err := svc.GetData(ctx, BrowserDataRequest{Database: name, Table: "events", Page: 1, PageSize: 1, SortField: "id", SortOrder: "DESC"})
		if err != nil {
			t.Fatal(err)
		}
		if len(data.Rows) != 1 || data.Rows[0]["value"] != "second" {
			t.Fatalf("unexpected sorted page: %+v", data)
		}
		if _, err := svc.ListTables(ctx, name); err != nil {
			t.Fatal(err)
		}
		if err := svc.AddColumn(ctx, AddColumnRequest{Database: name, Table: "events", ColumnName: "extra", Type: "String"}); err != nil {
			t.Fatal(err)
		}
		if err := svc.DropColumn(ctx, name, "events", "extra"); err != nil {
			t.Fatal(err)
		}
		column := "quoted`\\column"
		comment := "owner's \\ directory"
		if err := svc.AddColumn(ctx, AddColumnRequest{Database: name, Table: "events", ColumnName: column, Type: "String", Comment: comment}); err != nil {
			t.Fatal(err)
		}
		columns, err := svc.GetColumns(ctx, name, "events")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, col := range columns {
			if col.Name == column && col.Comment == comment {
				found = true
			}
		}
		if !found {
			t.Fatalf("quoted column or comment changed: %+v", columns)
		}
		if err := svc.DropColumn(ctx, name, "events", column); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.ImportData(ctx, ImportDataRequest{Database: name, Table: "events", Format: "CSV", Data: "3,third"}); err != nil {
			t.Fatal("import:", err)
		}
		data, err = svc.GetData(ctx, BrowserDataRequest{Database: name, Table: "events", Page: 1, PageSize: 10})
		if err != nil || len(data.Rows) != 3 {
			t.Fatalf("imported rows missing: %+v, %v", data, err)
		}
		data, err = svc.GetData(ctx, BrowserDataRequest{Database: name, Table: "events", Page: 1, PageSize: 10, Where: "id > 1"})
		if err != nil || data.TotalRows != 2 || len(data.Rows) != 2 {
			t.Fatalf("filtered count or rows incorrect: %+v, %v", data, err)
		}
		if _, err := svc.GetData(ctx, BrowserDataRequest{Database: name, Table: "events", Where: "missing_column = 1"}); err == nil {
			t.Fatal("invalid filter silently accepted")
		}
	})

	empty, err := svc.Query(ctx, "SELECT 1 WHERE 0", 10)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Rows == nil || len(empty.Rows) != 0 {
		t.Fatalf("empty rows must be []: %+v", empty.Rows)
	}
	limited, err := svc.Query(ctx, "SELECT number FROM numbers(20)", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited.Rows) != 3 {
		t.Fatalf("row limit not applied: %d", len(limited.Rows))
	}
	result, err := svc.Query(ctx, "SELECT queryID() AS id", 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Rows[0]["id"] != result.QueryID {
		t.Fatalf("query ID mismatch: %+v", result)
	}
	result, err = svc.QueryWithDatabase(ctx, "SELECT currentDatabase() AS db FROM one", 10, "system")
	if err != nil {
		t.Fatal(err)
	}
	if result.Rows[0]["db"] != "system" {
		t.Fatalf("database selection ignored: %+v", result)
	}
	result, err = svc.Query(ctx, "SELECT currentDatabase() AS db", 10)
	if err != nil {
		t.Fatal(err)
	}
	if result.Rows[0]["db"] != "default" {
		t.Fatalf("shared database changed: %+v", result)
	}
	if _, err := svc.Query(ctx, "SELECT missing_column FROM system.one", 10); err == nil {
		t.Fatal("invalid query succeeded")
	}
	result, err = svc.Query(ctx, "SELECT toUInt64('18446744073709551615') AS big, [toInt64('-9223372036854775808')] AS nested, toFloat64('nan') AS special, toDateTime64('2026-01-01 00:00:00.123456', 6) AS precise, toIPv4('127.0.0.1') AS ip, toUInt128('340282366920938463463374607431768211455') AS huge", 1)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result.Rows)
	if err != nil {
		t.Fatal("result cannot be encoded as JSON:", err)
	}
	var decoded []map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0]["big"] != "18446744073709551615" || decoded[0]["nested"].([]interface{})[0] != "-9223372036854775808" || decoded[0]["special"] != "NaN" || decoded[0]["precise"] != "2026-01-01 00:00:00.123456" || decoded[0]["ip"] != "127.0.0.1" || decoded[0]["huge"] != "340282366920938463463374607431768211455" {
		t.Fatalf("result precision changed: %s", encoded)
	}
}

func TestBrowserPageOverflow(t *testing.T) {
	svc := &ClickHouseService{}
	if _, err := svc.GetData(context.Background(), BrowserDataRequest{Database: "db", Table: "table", Page: int(^uint(0) >> 1), PageSize: 5000}); err == nil {
		t.Fatal("overflowing page accepted")
	}
}

func TestGrantScopeDoesNotExpand(t *testing.T) {
	svc := &ClickHouseService{}
	req := GrantRequest{Username: "user", Privileges: []string{"SELECT"}, Database: "*", Table: "events"}
	if err := svc.GrantPrivileges(context.Background(), req); err == nil || !strings.Contains(err.Error(), "具体数据库") {
		t.Fatalf("grant scope was not rejected: %v", err)
	}
	if err := svc.RevokePrivileges(context.Background(), req); err == nil || !strings.Contains(err.Error(), "具体数据库") {
		t.Fatalf("revoke scope was not rejected: %v", err)
	}
}

func TestAlterXMLUser(t *testing.T) {
	svc := &ClickHouseService{}
	req := AlterUserRequest{
		Username:    "default",
		NewPassword: "test_secret_password",
	}
	err := svc.alterXMLUser(context.Background(), req)
	if err != nil {
		t.Fatalf("alterXMLUser failed: %v", err)
	}
	cfgDir := config.GetConfigDir()
	targetFile := filepath.Join(cfgDir, "users.d", "ch_manager_user_default.xml")
	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read generated xml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "<password_sha256_hex replace=\"replace\">") {
		t.Errorf("xml does not contain password_sha256_hex: %s", content)
	}
	if !strings.Contains(content, "<password remove=\"remove\"/>") {
		t.Errorf("xml does not contain remove password: %s", content)
	}
	_ = os.Remove(targetFile)
}
