package postgresql

import (
	"reflect"
	"strings"
	"testing"
)

// testStringer provides deterministic fmt.Stringer output 提供稳定的 fmt.Stringer 输出
type testStringer struct{}

// String returns a fixed string 返回固定字符串
func (testStringer) String() string {
	return "stringer-value"
}

// TestConfigDataSourceName verifies PostgreSQL DSN construction 测试 PostgreSQL DSN 构建
func TestConfigDataSourceName(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "defaults",
			cfg:  Config{},
			want: "postgres://localhost:5432?sslmode=disable",
		},
		{
			name: "full config",
			cfg: Config{
				Host:     "db.local",
				Port:     15432,
				Username: "user",
				Password: "pass",
				Database: "app",
				SSLMode:  "require",
			},
			want: "postgres://user:pass@db.local:15432/app?sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.dataSourceName(); got != tt.want {
				t.Fatalf("dataSourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseStorageURL verifies storage URL parsing and table extraction 测试存储 URL 解析与表名提取
func TestParseStorageURL(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		wantDSN   string
		wantTable string
		wantErr   bool
	}{
		{
			name:    "empty url",
			wantErr: true,
		},
		{
			name:      "keyword dsn",
			rawURL:    "host=localhost user=postgres dbname=app",
			wantDSN:   "host=localhost user=postgres dbname=app",
			wantTable: defaultTableName,
		},
		{
			name:      "url table parameter",
			rawURL:    "postgres://user:pass@localhost:5432/app?sslmode=disable&table=sessions",
			wantDSN:   "postgres://user:pass@localhost:5432/app?sslmode=disable",
			wantTable: "sessions",
		},
		{
			name:      "table name priority",
			rawURL:    "postgres://localhost/app?table=low&tableName=middle&table_name=high",
			wantDSN:   "postgres://localhost/app",
			wantTable: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDSN, gotTable, err := parseStorageURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseStorageURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotDSN != tt.wantDSN || gotTable != tt.wantTable {
				t.Fatalf("parseStorageURL() = (%q, %q), want (%q, %q)", gotDSN, gotTable, tt.wantDSN, tt.wantTable)
			}
		})
	}
}

// TestValueToBytes verifies storage value conversion 测试存储值字节转换
func TestValueToBytes(t *testing.T) {
	raw := []byte("bytes")
	tests := []struct {
		name  string
		value any
		want  []byte
	}{
		{name: "nil", value: nil, want: []byte{}},
		{name: "string", value: "text", want: []byte("text")},
		{name: "bytes", value: raw, want: raw},
		{name: "stringer", value: testStringer{}, want: []byte("stringer-value")},
		{name: "default", value: 42, want: []byte("42")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := valueToBytes(tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("valueToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRedisPatternToLike verifies wildcard conversion 测试通配符转换
func TestRedisPatternToLike(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "empty", pattern: "", want: "%"},
		{name: "star", pattern: "user:*", want: "user:%"},
		{name: "question", pattern: "user:?", want: "user:_"},
		{name: "escaped star", pattern: `user:\*`, want: "user:*"},
		{name: "like literals", pattern: `percent_%`, want: `percent\_\%`},
		{name: "trailing escape", pattern: `path\`, want: `path\\`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redisPatternToLike(tt.pattern); got != tt.want {
				t.Fatalf("redisPatternToLike() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTableIdentifierHelpers verifies table identifier parsing helpers 测试表标识符辅助函数
func TestTableIdentifierHelpers(t *testing.T) {
	tests := []struct {
		name       string
		tableName  string
		wantSchema string
		wantName   string
		wantQuoted string
	}{
		{
			name:       "default",
			wantName:   defaultTableName,
			wantQuoted: `"dtoken_storage"`,
		},
		{
			name:       "schema table",
			tableName:  "public.sessions",
			wantSchema: "public",
			wantName:   "sessions",
			wantQuoted: `"public"."sessions"`,
		},
		{
			name:       "trim spaces",
			tableName:  " audit . token ",
			wantSchema: "audit",
			wantName:   "token",
			wantQuoted: `"audit"."token"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTableIdentifier(tt.tableName)
			if got.schema != tt.wantSchema || got.name != tt.wantName || got.quoted != tt.wantQuoted {
				t.Fatalf("parseTableIdentifier() = %#v, want schema=%q name=%q quoted=%q", got, tt.wantSchema, tt.wantName, tt.wantQuoted)
			}
		})
	}

	if got := quoteIdentifier(`a"b`); got != `"a""b"` {
		t.Fatalf("quoteIdentifier() = %q, want %q", got, `"a""b"`)
	}
	if got := indexName(tableIdentifier{schema: "public", name: "sessions"}); got != "public_sessions_expires_at_idx" {
		t.Fatalf("indexName() = %q, want %q", got, "public_sessions_expires_at_idx")
	}
	if got := indexName(tableIdentifier{name: strings.Repeat("x", 80)}); len(got) != 63 || !strings.HasSuffix(got, "_expires_at_idx") {
		t.Fatalf("indexName(long) = %q, want length 63 with suffix", got)
	}
}

// TestStorageAccessorsAndValidation verifies nil-safe accessors and validation 测试空值安全访问与校验
func TestStorageAccessorsAndValidation(t *testing.T) {
	var nilStorage *Storage
	if err := nilStorage.Close(); err != nil {
		t.Fatalf("Close(nil) error = %v", err)
	}
	if nilStorage.GetDB() != nil {
		t.Fatal("GetDB(nil) should return nil")
	}
	if nilStorage.TableName() != "" {
		t.Fatal("TableName(nil) should return empty string")
	}
	if err := nilStorage.ensureReady(); err == nil {
		t.Fatal("ensureReady(nil) error = nil, want error")
	}

	storage := NewStorageFromDB(nil, "public.sessions")
	if storage.GetDB() != nil {
		t.Fatal("GetDB() should return nil db")
	}
	if got := storage.TableName(); got != `"public"."sessions"` {
		t.Fatalf("TableName() = %q, want %q", got, `"public"."sessions"`)
	}
	if err := storage.ensureReady(); err == nil {
		t.Fatal("ensureReady() error = nil, want nil db error")
	}

	if _, err := NewStorageFromConfig(nil); err == nil {
		t.Fatal("NewStorageFromConfig(nil) error = nil, want error")
	}
}

// TestFirstNonEmpty verifies first non-empty selection 测试首个非空值选择
func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "value", "next"); got != "value" {
		t.Fatalf("firstNonEmpty() = %q, want %q", got, "value")
	}
	if got := firstNonEmpty("", "  "); got != "" {
		t.Fatalf("firstNonEmpty(empty) = %q, want empty", got)
	}
}
