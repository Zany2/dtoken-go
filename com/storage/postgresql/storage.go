// Author records daixk as original author at 2026/4/30 16:00:00. Author 记录 daixk 为原始作者，创建时间为 2026/4/30 16:00:00。
package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// ErrKeyNotFound indicates the key is missing or expired ErrKeyNotFound 表示键不存在或已过期
var ErrKeyNotFound = errors.New("key not found")

// TTL constants define PostgreSQL TTL sentinel values TTL 常量定义 PostgreSQL TTL 哨兵值
const (
	// TTLNoExpire indicates the key has no expiration TTLNoExpire 表示键永不过期
	TTLNoExpire = adapter.TTLNoExpire
	// TTLNotFound indicates the key does not exist TTLNotFound 表示键不存在
	TTLNotFound = adapter.TTLNotFound

	// defaultTableName stores the default PostgreSQL table name defaultTableName 存储默认 PostgreSQL 表名
	defaultTableName = "dtoken_storage"
)

// Config defines PostgreSQL storage configuration Config 定义 PostgreSQL 存储配置
type Config struct {
	// Host specifies PostgreSQL server host Host 指定 PostgreSQL 服务地址
	Host string
	// Port specifies PostgreSQL server port Port 指定 PostgreSQL 服务端口
	Port int
	// Username specifies PostgreSQL username Username 指定 PostgreSQL 用户名
	Username string
	// Password specifies PostgreSQL password Password 指定 PostgreSQL 密码
	Password string
	// Database specifies PostgreSQL database name Database 指定 PostgreSQL 数据库名
	Database string
	// SSLMode specifies PostgreSQL sslmode SSLMode 指定 PostgreSQL SSL 模式
	SSLMode string
	// TableName specifies storage table name TableName 指定存储表名，支持 schema.table
	TableName string
	// MaxOpenConns specifies max open connections MaxOpenConns 指定最大打开连接数
	MaxOpenConns int
	// MaxIdleConns specifies max idle connections MaxIdleConns 指定最大空闲连接数
	MaxIdleConns int
	// ConnMaxLifetime specifies connection max lifetime ConnMaxLifetime 指定连接最长生命周期
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime specifies connection max idle time ConnMaxIdleTime 指定连接最长空闲时间
	ConnMaxIdleTime time.Duration
}

// Storage implements PostgreSQL backed storage Storage 实现 PostgreSQL 存储
type Storage struct {
	db    *sql.DB         // db stores PostgreSQL database handle db 存储 PostgreSQL 数据库句柄
	table tableIdentifier // table stores quoted table metadata table 存储已转义的表元数据
}

// Interface assertion keeps storage contract checked at compile time 接口断言在编译期检查存储契约
var _ adapter.Storage = (*Storage)(nil)
var _ adapter.AtomicStorage = (*Storage)(nil)
var _ adapter.FullStorage = (*Storage)(nil)

// tableIdentifier stores parsed table metadata tableIdentifier 存储解析后的表元数据
type tableIdentifier struct {
	schema string // schema stores optional schema name schema 存储可选 schema 名称
	name   string // name stores table name name 存储表名
	quoted string // quoted stores SQL quoted table name quoted 存储 SQL 转义后的表名
}

// NewStorage creates storage from a PostgreSQL URL NewStorage 通过 PostgreSQL URL 创建存储
func NewStorage(rawURL string) (*Storage, error) {
	dsn, tableName, err := parseStorageURL(rawURL)
	if err != nil {
		return nil, err
	}

	// Open database handle 打开数据库句柄
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgresql storage: %w", err)
	}

	// Initialize table before returning 初始化表后再返回
	storage := NewStorageFromDB(db, tableName)
	if err = storage.Init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return storage, nil
}

// NewStorageFromConfig creates storage from config NewStorageFromConfig 通过配置创建存储
func NewStorageFromConfig(cfg *Config) (*Storage, error) {
	if cfg == nil {
		return nil, errors.New("postgresql config is nil")
	}

	// Open database handle from config 根据配置打开数据库句柄
	db, err := sql.Open("pgx", cfg.dataSourceName())
	if err != nil {
		return nil, fmt.Errorf("failed to open postgresql storage: %w", err)
	}

	// Apply connection pool settings 应用连接池设置
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	// Initialize table before returning 初始化表后再返回
	storage := NewStorageFromDB(db, cfg.TableName)
	if err = storage.Init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return storage, nil
}

// NewStorageFromDB creates storage from an existing sql.DB NewStorageFromDB 从已有 sql.DB 创建存储
func NewStorageFromDB(db *sql.DB, tableName string) *Storage {
	return &Storage{
		db:    db,
		table: parseTableIdentifier(tableName),
	}
}

// Init initializes storage table Init 初始化存储表
func (s *Storage) Init(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if err := s.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to postgresql: %w", err)
	}

	// Create storage table 创建存储表
	createTableQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			"key" TEXT PRIMARY KEY,
			"value" BYTEA NOT NULL,
			expires_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`, s.table.quoted)
	if _, err := s.db.ExecContext(ctx, createTableQuery); err != nil {
		return fmt.Errorf("failed to init postgresql storage table: %w", err)
	}

	// Migrate old TEXT value column to BYTEA 迁移旧版 TEXT value 列为 BYTEA
	if err := s.migrateValueColumn(ctx); err != nil {
		return err
	}

	// Create expiration index 创建过期时间索引
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s
		ON %s (expires_at)
	`, quoteIdentifier(indexName(s.table)), s.table.quoted)
	if _, err := s.db.ExecContext(ctx, indexQuery); err != nil {
		return fmt.Errorf("failed to init postgresql storage index: %w", err)
	}

	// Cleanup expired rows during startup 启动时清理已过期数据
	return s.deleteExpired(ctx)
}

// Set stores a key value pair Set 设置键值对
func (s *Storage) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	// Convert value to bytes 转换存储值为字节
	bytesValue := valueToBytes(value)
	var expiresAt any
	if expiration > 0 {
		expiresAt = time.Now().UTC().Add(expiration)
	}

	// Upsert key and expiration 写入或更新键值与过期时间
	query := fmt.Sprintf(`
		INSERT INTO %s ("key", "value", expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT ("key") DO UPDATE SET
			"value" = EXCLUDED."value",
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
	`, s.table.quoted)

	_, err := s.db.ExecContext(ctx, query, key, bytesValue, expiresAt)
	return err
}

// Get retrieves the value for a key Get 获取指定键的值
func (s *Storage) Get(ctx context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	// Read only non-expired rows 只读取未过期数据
	query := fmt.Sprintf(`
		SELECT "value"
		FROM %s
		WHERE "key" = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`, s.table.quoted)

	var value []byte
	err := s.db.QueryRowContext(ctx, query, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		_ = s.deleteExpiredKey(ctx, key)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return value, nil
}

// GetAndDelete atomically gets and deletes a key GetAndDelete 原子获取并删除键
func (s *Storage) GetAndDelete(ctx context.Context, key string) (any, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	// Delete and return the value in one statement 单条语句删除并返回值
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE "key" = $1 AND (expires_at IS NULL OR expires_at > NOW())
		RETURNING "value"
	`, s.table.quoted)

	var value []byte
	err := s.db.QueryRowContext(ctx, query, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		_ = s.deleteExpiredKey(ctx, key)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return value, nil
}

// Delete removes one or more keys Delete 删除一个或多个键
func (s *Storage) Delete(ctx context.Context, keys ...string) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	// Build parameterized IN list 构建参数化 IN 列表
	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, key := range keys {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = key
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE "key" IN (%s)`, s.table.quoted, strings.Join(placeholders, ","))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Exists checks whether a key exists Exists 检查键是否存在
func (s *Storage) Exists(ctx context.Context, key string) bool {
	if err := s.ensureReady(); err != nil {
		return false
	}

	// Check only non-expired rows 只检查未过期数据
	query := fmt.Sprintf(`
		SELECT 1
		FROM %s
		WHERE "key" = $1 AND (expires_at IS NULL OR expires_at > NOW())
		LIMIT 1
	`, s.table.quoted)

	var exists int
	err := s.db.QueryRowContext(ctx, query, key).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		_ = s.deleteExpiredKey(ctx, key)
		return false
	}
	return err == nil
}

// Keys gets all keys matching pattern Keys 获取匹配模式的所有键
func (s *Storage) Keys(ctx context.Context, pattern string) ([]string, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}

	// Convert Redis wildcard pattern to SQL LIKE pattern 将 Redis 通配符转换为 SQL LIKE 模式
	query := fmt.Sprintf(`
		SELECT "key"
		FROM %s
		WHERE (expires_at IS NULL OR expires_at > NOW())
			AND "key" LIKE $1 ESCAPE E'\\'
		ORDER BY "key"
	`, s.table.quoted)

	rows, err := s.db.QueryContext(ctx, query, redisPatternToLike(pattern))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect matched keys 收集匹配的键
	keys := make([]string, 0)
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return keys, s.deleteExpired(ctx)
}

// Expire sets the expiration for a key Expire 设置键的过期时间
func (s *Storage) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	if expiration <= 0 {
		// Delete immediately for non-positive expiration 非正数过期时间表示立即删除
		query := fmt.Sprintf(`
			DELETE FROM %s
			WHERE "key" = $1 AND (expires_at IS NULL OR expires_at > NOW())
		`, s.table.quoted)

		result, err := s.db.ExecContext(ctx, query, key)
		if err != nil {
			return err
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return ErrKeyNotFound
		}
		return nil
	}

	// Update expiration only for existing non-expired rows 只更新存在且未过期数据的过期时间
	query := fmt.Sprintf(`
		UPDATE %s
		SET expires_at = $2, updated_at = NOW()
		WHERE "key" = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`, s.table.quoted)

	result, err := s.db.ExecContext(ctx, query, key, time.Now().UTC().Add(expiration))
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		_ = s.deleteExpiredKey(ctx, key)
		return ErrKeyNotFound
	}
	return nil
}

// TTL gets the remaining lifetime for a key TTL 获取键的剩余生存时间
func (s *Storage) TTL(ctx context.Context, key string) (time.Duration, error) {
	if err := s.ensureReady(); err != nil {
		return 0, err
	}

	// Read expiration for non-expired row 读取未过期数据的过期时间
	query := fmt.Sprintf(`
		SELECT expires_at
		FROM %s
		WHERE "key" = $1 AND (expires_at IS NULL OR expires_at > NOW())
	`, s.table.quoted)

	var expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, key).Scan(&expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		_ = s.deleteExpiredKey(ctx, key)
		return TTLNotFound, nil
	}
	if err != nil {
		return 0, err
	}
	if !expiresAt.Valid {
		return TTLNoExpire, nil
	}

	// Calculate remaining duration 计算剩余时间
	ttl := time.Until(expiresAt.Time)
	if ttl <= 0 {
		_ = s.deleteExpiredKey(ctx, key)
		return TTLNotFound, nil
	}
	return ttl, nil
}

// Clear removes all stored data Clear 清空所有数据
func (s *Storage) Clear(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}

	query := fmt.Sprintf("TRUNCATE TABLE %s", s.table.quoted)
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Ping checks the PostgreSQL connection Ping 检查 PostgreSQL 连接
func (s *Storage) Ping(ctx context.Context) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	return s.db.PingContext(ctx)
}

// Close closes the PostgreSQL connection Close 关闭 PostgreSQL 连接
func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// GetDB returns the PostgreSQL db handle GetDB 返回 PostgreSQL 数据库句柄
func (s *Storage) GetDB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

// TableName returns the quoted storage table name TableName 返回已转义的存储表名
func (s *Storage) TableName() string {
	if s == nil {
		return ""
	}
	return s.table.quoted
}

// migrateValueColumn migrates old text value column to bytea migrateValueColumn 将旧版 text value 列迁移为 bytea
func (s *Storage) migrateValueColumn(ctx context.Context) error {
	dataType, err := s.valueColumnDataType(ctx)
	if err != nil {
		return err
	}
	if dataType == "" || dataType == "bytea" {
		return nil
	}

	// Convert textual payloads to UTF-8 bytes 转换文本载荷为 UTF-8 字节
	query := fmt.Sprintf(`
		ALTER TABLE %s
		ALTER COLUMN "value" TYPE BYTEA
		USING convert_to("value"::TEXT, 'UTF8')
	`, s.table.quoted)
	if _, err = s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to migrate postgresql storage value column: %w", err)
	}
	return nil
}

// valueColumnDataType gets the current value column data type valueColumnDataType 获取当前 value 列类型
func (s *Storage) valueColumnDataType(ctx context.Context) (string, error) {
	var (
		query string // query stores metadata query query 存储元数据查询语句
		args  []any  // args stores query args args 存储查询参数
	)

	// Query information_schema by schema and table 按 schema 和表名查询 information_schema
	if s.table.schema == "" {
		query = `
			SELECT data_type
			FROM information_schema.columns
			WHERE table_schema = current_schema()
				AND table_name = $1
				AND column_name = 'value'
		`
		args = []any{s.table.name}
	} else {
		query = `
			SELECT data_type
			FROM information_schema.columns
			WHERE table_schema = $1
				AND table_name = $2
				AND column_name = 'value'
		`
		args = []any{s.table.schema, s.table.name}
	}

	var dataType string
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&dataType)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to inspect postgresql storage value column: %w", err)
	}
	return dataType, nil
}

// deleteExpiredKey deletes expired row for one key deleteExpiredKey 删除单个已过期键
func (s *Storage) deleteExpiredKey(ctx context.Context, key string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE "key" = $1 AND expires_at <= NOW()`, s.table.quoted)
	_, err := s.db.ExecContext(ctx, query, key)
	return err
}

// deleteExpired deletes all expired rows deleteExpired 删除所有过期行
func (s *Storage) deleteExpired(ctx context.Context) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE expires_at <= NOW()", s.table.quoted)
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// ensureReady checks storage dependencies ensureReady 检查存储依赖是否可用
func (s *Storage) ensureReady() error {
	if s == nil || s.db == nil {
		return errors.New("postgresql storage db is nil")
	}
	if s.table.quoted == "" {
		return errors.New("postgresql storage table name is empty")
	}
	return nil
}

// dataSourceName builds PostgreSQL connection URL dataSourceName 构建 PostgreSQL 连接地址
func (c *Config) dataSourceName() string {
	port := c.Port
	if port == 0 {
		port = 5432
	}
	host := c.Host
	if host == "" {
		host = "localhost"
	}
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	// Build URL with optional credentials 构建带可选凭据的 URL
	dsn := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
	}
	if c.Username != "" || c.Password != "" {
		dsn.User = url.UserPassword(c.Username, c.Password)
	}
	if c.Database != "" {
		dsn.Path = "/" + c.Database
	}

	query := dsn.Query()
	query.Set("sslmode", sslMode)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

// parseStorageURL extracts dsn and optional table name parseStorageURL 提取连接地址和可选表名
func parseStorageURL(rawURL string) (string, string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", "", errors.New("postgresql url is empty")
	}

	// Keep keyword DSN unchanged 保留关键字形式 DSN
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" {
		return rawURL, defaultTableName, nil
	}

	query := parsed.Query()
	tableName := firstNonEmpty(query.Get("table_name"), query.Get("tableName"), query.Get("table"))
	if tableName == "" {
		tableName = defaultTableName
	}

	// Remove storage-only query params before connecting 连接前移除仅用于存储配置的参数
	query.Del("table_name")
	query.Del("tableName")
	query.Del("table")
	parsed.RawQuery = query.Encode()

	return parsed.String(), tableName, nil
}

// valueToBytes converts storage value to bytes valueToBytes 转换存储值为字节
func valueToBytes(value any) []byte {
	switch v := value.(type) {
	case nil:
		return []byte{}
	case string:
		return []byte(v)
	case []byte:
		return v
	case fmt.Stringer:
		return []byte(v.String())
	default:
		return []byte(fmt.Sprint(v))
	}
}

// redisPatternToLike converts Redis wildcard pattern to SQL LIKE redisPatternToLike 将 Redis 通配符转换为 SQL LIKE
func redisPatternToLike(pattern string) string {
	if pattern == "" {
		pattern = "*"
	}

	var builder strings.Builder
	escaped := false
	for _, char := range pattern {
		if escaped {
			writeLikeLiteral(&builder, char)
			escaped = false
			continue
		}

		switch char {
		case '\\':
			escaped = true
		case '*':
			builder.WriteByte('%')
		case '?':
			builder.WriteByte('_')
		default:
			writeLikeLiteral(&builder, char)
		}
	}

	if escaped {
		builder.WriteString(`\\`)
	}
	return builder.String()
}

// writeLikeLiteral writes escaped LIKE literal writeLikeLiteral 写入转义后的 LIKE 字面量
func writeLikeLiteral(builder *strings.Builder, char rune) {
	switch char {
	case '%', '_', '\\':
		builder.WriteByte('\\')
	}
	builder.WriteRune(char)
}

// parseTableIdentifier parses a table name parseTableIdentifier 解析表名
func parseTableIdentifier(tableName string) tableIdentifier {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		tableName = defaultTableName
	}

	// Support schema.table while keeping simple table names 支持 schema.table，同时保留普通表名
	parts := strings.Split(tableName, ".")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		cleaned = []string{defaultTableName}
	}

	quoted := make([]string, 0, len(cleaned))
	for _, part := range cleaned {
		quoted = append(quoted, quoteIdentifier(part))
	}

	table := tableIdentifier{
		name:   cleaned[len(cleaned)-1],
		quoted: strings.Join(quoted, "."),
	}
	if len(cleaned) > 1 {
		table.schema = cleaned[len(cleaned)-2]
	}
	return table
}

// quoteIdentifier quotes one SQL identifier quoteIdentifier 转义单个 SQL 标识符
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// indexName builds a stable index name indexName 构建稳定的索引名
func indexName(table tableIdentifier) string {
	name := table.name
	if table.schema != "" {
		name = table.schema + "_" + table.name
	}
	name = strings.NewReplacer(`"`, "", ".", "_").Replace(name)
	if len(name) > 48 {
		name = name[:48]
	}
	return name + "_expires_at_idx"
}

// firstNonEmpty returns the first non-empty string firstNonEmpty 返回第一个非空字符串
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
