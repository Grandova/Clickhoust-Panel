package clickhouse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"clickhouse-manager/internal/config"
)

type ClickHouseUser struct {
	Name            string   `json:"name"`
	Storage         string   `json:"storage"`
	AuthType        string   `json:"auth_type"`
	HostIP          []string `json:"host_ip"`
	DefaultDatabase string   `json:"default_database"`
	Profile         string   `json:"profile"`
	Quota           string   `json:"quota"`
	Grants          []string `json:"grants"`
}

type CreateUserRequest struct {
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	AllowedHosts    []string `json:"allowed_hosts"`
	DefaultDatabase string   `json:"default_database"`
	Profile         string   `json:"profile"`
	Quota           string   `json:"quota"`
	AllPrivileges   bool     `json:"all_privileges"`
}

type AlterUserRequest struct {
	Username        string   `json:"username"`
	NewPassword     string   `json:"new_password"`
	AllowedHosts    []string `json:"allowed_hosts"`
	DefaultDatabase string   `json:"default_database"`
	Profile         string   `json:"profile"`
	Quota           string   `json:"quota"`
}

type GrantRequest struct {
	Username        string   `json:"username"`
	Privileges      []string `json:"privileges"` // SELECT, INSERT, ALTER, CREATE, DROP, etc., or ALL
	Database        string   `json:"database"`   // * or specific
	Table           string   `json:"table"`      // * or specific
	WithGrantOption bool     `json:"with_grant_option"`
}

var safeIdentRegexp = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

func (s *ClickHouseService) ListUsers(ctx context.Context) ([]ClickHouseUser, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			name,
			storage,
			toString(auth_type),
			host_ip,
			default_database,
			default_roles_all
		FROM system.users
		ORDER BY name
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		// Fallback for any older ClickHouse versions where storage column might differ
		queryLegacy := `
			SELECT 
				name,
				'unknown',
				toString(auth_type),
				host_ip,
				default_database,
				default_roles_all
			FROM system.users
			ORDER BY name
		`
		var legacyErr error
		rows, legacyErr = conn.Query(ctx, queryLegacy)
		if legacyErr != nil {
			return nil, err
		}
	}
	defer rows.Close()

	users := make([]ClickHouseUser, 0)
	for rows.Next() {
		var u ClickHouseUser
		var hostIPs []string
		var defaultRolesAll uint8
		if err := rows.Scan(&u.Name, &u.Storage, &u.AuthType, &hostIPs, &u.DefaultDatabase, &defaultRolesAll); err != nil {
			return nil, err
		}
		u.HostIP = hostIPs

		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range users {
		grants, err := s.GetUserGrants(ctx, users[i].Name)
		if err != nil {
			return nil, err
		}
		users[i].Grants = grants
	}
	return users, nil
}

func (s *ClickHouseService) GetUserGrants(ctx context.Context, username string) ([]string, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, fmt.Sprintf("SHOW GRANTS FOR `%s`", escapeIdent(username)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	grants := make([]string, 0)
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
}

func (s *ClickHouseService) CreateUser(ctx context.Context, req CreateUserRequest) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("CREATE USER `%s`", escapeIdent(req.Username))
	if req.Password != "" {
		sql += fmt.Sprintf(" IDENTIFIED BY '%s'", escapeString(req.Password))
	} else {
		sql += " NOT IDENTIFIED"
	}

	if len(req.AllowedHosts) > 0 {
		var hostClauses []string
		for _, h := range req.AllowedHosts {
			hostClauses = append(hostClauses, fmt.Sprintf("'%s'", escapeString(h)))
		}
		sql += fmt.Sprintf(" HOST IP %s", strings.Join(hostClauses, ", "))
	}

	if req.DefaultDatabase != "" {
		sql += fmt.Sprintf(" DEFAULT DATABASE `%s`", escapeIdent(req.DefaultDatabase))
	}

	if req.Profile != "" {
		sql += fmt.Sprintf(" SETTINGS PROFILE `%s`", escapeIdent(req.Profile))
	}

	if err := conn.Exec(ctx, sql); err != nil {
		return err
	}

	if req.AllPrivileges {
		_ = conn.Exec(ctx, fmt.Sprintf("GRANT ALL ON *.* TO `%s` WITH GRANT OPTION", escapeIdent(req.Username)))
	}

	return nil
}

func (s *ClickHouseService) DropUser(ctx context.Context, username string) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}
	return conn.Exec(ctx, fmt.Sprintf("DROP USER IF EXISTS `%s`", escapeIdent(username)))
}

func (s *ClickHouseService) AlterUser(ctx context.Context, req AlterUserRequest) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}

	// 检查该用户是否为 users_xml (例如 default 用户)
	var storage string
	_ = conn.QueryRow(ctx, "SELECT storage FROM system.users WHERE name = ?", req.Username).Scan(&storage)

	if storage == "users_xml" || req.Username == "default" {
		return s.alterXMLUser(ctx, req)
	}

	err = s.alterSQLUser(ctx, req)
	if err != nil && (strings.Contains(err.Error(), "users_xml") || strings.Contains(err.Error(), "495")) {
		return s.alterXMLUser(ctx, req)
	}
	return err
}

func (s *ClickHouseService) alterSQLUser(ctx context.Context, req AlterUserRequest) error {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}

	sql := fmt.Sprintf("ALTER USER `%s`", escapeIdent(req.Username))
	var clauses []string

	if req.NewPassword != "" {
		clauses = append(clauses, fmt.Sprintf("IDENTIFIED BY '%s'", escapeString(req.NewPassword)))
	}

	if req.DefaultDatabase != "" {
		clauses = append(clauses, fmt.Sprintf("DEFAULT DATABASE `%s`", escapeIdent(req.DefaultDatabase)))
	}

	if req.Profile != "" {
		clauses = append(clauses, fmt.Sprintf("SETTINGS PROFILE `%s`", escapeIdent(req.Profile)))
	}

	if len(clauses) == 0 {
		return fmt.Errorf("未指定需要修改的用户属性")
	}

	sql += " " + strings.Join(clauses, " ")
	return conn.Exec(ctx, sql)
}

func (s *ClickHouseService) alterXMLUser(ctx context.Context, req AlterUserRequest) error {
	safeUser := req.Username
	if !safeIdentRegexp.MatchString(safeUser) {
		return fmt.Errorf("用户名包含非法字符")
	}

	cfgDir := config.GetConfigDir()
	usersDDir := filepath.Join(cfgDir, "users.d")
	if err := os.MkdirAll(usersDDir, 0755); err != nil {
		return fmt.Errorf("创建 ClickHouse users.d 配置目录失败: %w", err)
	}

	targetFile := filepath.Join(usersDDir, "ch_manager_user_"+safeUser+".xml")

	var sb strings.Builder
	sb.WriteString("<clickhouse>\n")
	sb.WriteString("    <users>\n")
	sb.WriteString(fmt.Sprintf("        <%s>\n", safeUser))

	if req.NewPassword == "" {
		sb.WriteString("            <password replace=\"replace\"></password>\n")
		sb.WriteString("            <password_sha256_hex remove=\"remove\"/>\n")
		sb.WriteString("            <password_double_sha1_hex remove=\"remove\"/>\n")
	} else {
		hash := sha256.Sum256([]byte(req.NewPassword))
		hashHex := hex.EncodeToString(hash[:])
		sb.WriteString(fmt.Sprintf("            <password_sha256_hex replace=\"replace\">%s</password_sha256_hex>\n", hashHex))
		sb.WriteString("            <password remove=\"remove\"/>\n")
		sb.WriteString("            <password_double_sha1_hex remove=\"remove\"/>\n")
	}

	if req.DefaultDatabase != "" {
		sb.WriteString(fmt.Sprintf("            <default_database replace=\"replace\">%s</default_database>\n", req.DefaultDatabase))
	}

	sb.WriteString(fmt.Sprintf("        </%s>\n", safeUser))
	sb.WriteString("    </users>\n")
	sb.WriteString("</clickhouse>\n")

	if err := os.WriteFile(targetFile, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("写入 ClickHouse 覆盖配置文件失败 (%s): %w", targetFile, err)
	}

	// 触发 ClickHouse 热重载配置
	conn, err := s.GetConn(ctx)
	if err == nil {
		_ = conn.Exec(ctx, "SYSTEM RELOAD CONFIG")
	}

	return nil
}

func (s *ClickHouseService) GrantPrivileges(ctx context.Context, req GrantRequest) error {
	if (req.Database == "" || req.Database == "*") && req.Table != "" && req.Table != "*" {
		return fmt.Errorf("指定表名时必须选择具体数据库")
	}
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}

	isAll := false
	var privList []string
	for _, p := range req.Privileges {
		upper := strings.ToUpper(strings.TrimSpace(p))
		if upper == "ALL" || upper == "ALL PRIVILEGES" {
			isAll = true
			break
		}
		if upper != "" {
			privList = append(privList, upper)
		}
	}

	target := "*.*"
	if req.Database != "" && req.Database != "*" {
		target = fmt.Sprintf("`%s`.*", escapeIdent(req.Database))
		if req.Table != "" && req.Table != "*" {
			target = fmt.Sprintf("`%s`.`%s`", escapeIdent(req.Database), escapeIdent(req.Table))
		}
	}

	var sql string
	if isAll {
		sql = fmt.Sprintf("GRANT ALL ON %s TO `%s`", target, escapeIdent(req.Username))
	} else {
		if len(privList) == 0 {
			privList = []string{"SELECT"}
		}
		sql = fmt.Sprintf("GRANT %s ON %s TO `%s`", strings.Join(privList, ", "), target, escapeIdent(req.Username))
	}

	if req.WithGrantOption {
		sql += " WITH GRANT OPTION"
	}

	return conn.Exec(ctx, sql)
}

func (s *ClickHouseService) RevokePrivileges(ctx context.Context, req GrantRequest) error {
	if (req.Database == "" || req.Database == "*") && req.Table != "" && req.Table != "*" {
		return fmt.Errorf("指定表名时必须选择具体数据库")
	}
	conn, err := s.GetConn(ctx)
	if err != nil {
		return err
	}

	isAll := false
	var privList []string
	for _, p := range req.Privileges {
		upper := strings.ToUpper(strings.TrimSpace(p))
		if upper == "ALL" || upper == "ALL PRIVILEGES" {
			isAll = true
			break
		}
		if upper != "" {
			privList = append(privList, upper)
		}
	}

	target := "*.*"
	if req.Database != "" && req.Database != "*" {
		target = fmt.Sprintf("`%s`.*", escapeIdent(req.Database))
		if req.Table != "" && req.Table != "*" {
			target = fmt.Sprintf("`%s`.`%s`", escapeIdent(req.Database), escapeIdent(req.Table))
		}
	}

	var sql string
	if isAll {
		sql = fmt.Sprintf("REVOKE ALL ON %s FROM `%s`", target, escapeIdent(req.Username))
	} else {
		if len(privList) == 0 {
			privList = []string{"SELECT"}
		}
		sql = fmt.Sprintf("REVOKE %s ON %s FROM `%s`", strings.Join(privList, ", "), target, escapeIdent(req.Username))
	}

	return conn.Exec(ctx, sql)
}

func (s *ClickHouseService) ListProfiles(ctx context.Context) ([]string, error) {
	conn, err := s.GetConn(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx, "SELECT name FROM system.settings_profiles ORDER BY name")
	if err != nil {
		return []string{"default", "readonly"}, nil
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			list = append(list, name)
		}
	}
	return list, nil
}
