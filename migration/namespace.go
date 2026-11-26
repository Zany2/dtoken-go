// Package migration provides explicit offline namespace copying and credential conversion. Package migration 提供显式离线命名空间复制与凭据转换。
package migration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// ErrDestinationNotEmpty prevents merging a snapshot into an occupied namespace. ErrDestinationNotEmpty 防止将快照合并到已有数据的命名空间。
var ErrDestinationNotEmpty = errors.New("migration destination namespace is not empty")

// ErrPlanUsed prevents replaying a snapshot after an application attempt. ErrPlanUsed 防止在尝试执行后重放快照。
var ErrPlanUsed = errors.New("migration plan has already been applied or attempted")

// Options selects two prefixes for the same authentication system in one storage backend. Options 选择同一存储中同一认证体系的两个前缀。
type Options struct {
	// AuthType uses the same normalization as the auth builder; empty selects its default. AuthType 与认证 Builder 使用相同规范化规则，空值选取默认体系。
	AuthType string
	// SourcePrefix identifies the existing namespace and must be explicit. SourcePrefix 标识已有命名空间，必须显式指定。
	SourcePrefix string
	// TargetPrefix identifies an empty destination namespace and must be explicit. TargetPrefix 标识空目标命名空间，必须显式指定。
	TargetPrefix string
}

// Summary describes a prepared snapshot without exposing credentials or values. Summary 描述预检快照，不暴露凭据或存储值。
type Summary struct {
	// SourceNamespace is the exact source namespace. SourceNamespace 是实际源命名空间。
	SourceNamespace string
	// TargetNamespace is the exact destination namespace. TargetNamespace 是实际目标命名空间。
	TargetNamespace string
	// Records counts planned writes, including extensions added by credential conversion. Records 统计计划写入，包含凭据转换新增的扩展记录。
	Records int
	// Permanent counts records without expiration. Permanent 统计永久记录。
	Permanent int
	// Expiring counts records with a finite deadline. Expiring 统计有限期记录。
	Expiring int
	// Skipped counts records that vanished or expired during preparation. Skipped 统计预检期间消失或过期的记录。
	Skipped int
	// UpgradedTokens counts converted token primary records. UpgradedTokens 统计转换后的 Token 主记录。
	UpgradedTokens int
	// UpdatedRefreshTokens counts refresh records with normalized identity or lifecycle bindings. UpdatedRefreshTokens 统计规范化身份或生命周期绑定的刷新记录。
	UpdatedRefreshTokens int
	// AddedTokenExtras counts extension records split from inline token data. AddedTokenExtras 统计从 Token 内联数据分离的扩展记录。
	AddedTokenExtras int
	// UpgradedRestrictions counts restriction records with normalized keys or ownership. UpgradedRestrictions 统计规范化键或归属后的封禁记录。
	UpgradedRestrictions int
}

// Result reports acknowledged writes and records expired before their write; failures may leave partial target data. Result 报告已确认写入及写前过期记录，失败时目标可能存在部分数据。
type Result struct {
	// Written counts successfully acknowledged writes. Written 统计已确认成功的写入。
	Written int
	// Expired counts records skipped after their captured deadline passed. Expired 统计超过捕获截止时间而跳过的记录。
	Expired int
}

// Plan is created by a preparation function, owns an in-memory snapshot and borrows storage until Apply finishes; it must not be copied. Plan 由预检函数创建，持有内存快照并借用存储至 Apply 完成，不可复制。
type Plan struct {
	storage adapter.Storage
	scanner adapter.ScannerStorage
	summary Summary
	records []record
	mu      sync.Mutex // mu serializes application attempts. mu 串行化执行尝试。
	used    bool
}

type record struct {
	key      string
	value    any
	deadline time.Time // deadline is zero only for explicitly permanent source keys. deadline 仅对源存储明确永久的键为零。
}

// Prepare copies unversioned namespaces without conversion; use PrepareAuthUpgrade for current-runtime cutover and keep all writers stopped. Prepare 不转换格式地复制无版本空间；切换到当前运行时须使用 PrepareAuthUpgrade，并保持所有写入方停写。
func Prepare(ctx context.Context, storage adapter.Storage, opts Options) (*Plan, error) {
	source, err := namespace(opts.SourcePrefix, opts.AuthType)
	if err != nil {
		return nil, err
	}
	target, err := namespace(opts.TargetPrefix, opts.AuthType)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(source, target) || strings.HasPrefix(target, source) {
		return nil, fmt.Errorf("%w: source and target namespaces must not overlap", derror.ErrInvalidParam)
	}
	return prepareNamespaces(ctx, storage, source, target, "")
}

// prepareNamespaces captures source keys while optionally excluding an already-versioned subtree. prepareNamespaces 捕获源键，并可排除已有版本空间的子树。
func prepareNamespaces(ctx context.Context, storage adapter.Storage, source, target, excluded string) (*Plan, error) {
	if utils.IsNil(storage) {
		return nil, fmt.Errorf("%w: storage is required", derror.ErrInvalidParam)
	}
	scanner, ok := storage.(adapter.ScannerStorage)
	if !ok {
		return nil, derror.ErrStorageCapabilityUnsupported
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Scan literally after enumeration so glob characters in configured names cannot widen ownership. 枚举后按字面前缀筛选，避免配置中的 glob 字符扩大归属范围。
	keys, err := scanner.Keys(ctx, "*")
	if err != nil {
		return nil, err
	}
	slices.Sort(keys)
	keys = slices.Compact(keys)
	plan := &Plan{storage: storage, scanner: scanner, summary: Summary{SourceNamespace: source, TargetNamespace: target}}
	for _, key := range keys {
		if strings.HasPrefix(key, target) {
			return nil, ErrDestinationNotEmpty
		}
	}
	for _, key := range keys {
		if !strings.HasPrefix(key, source) || excluded != "" && strings.HasPrefix(key, excluded) {
			continue
		}
		if err = ctx.Err(); err != nil {
			return nil, err
		}

		// Anchor TTL before the read, conservatively charging backend and preparation latency. TTL 读取前记录时间，保守扣除存储与预检耗时。
		started := time.Now()
		ttl, err := storage.TTL(ctx, key)
		if err != nil {
			return nil, err
		}
		if ttl != adapter.TTLNoExpire && ttl <= 0 {
			if ttl < 0 && ttl != adapter.TTLNotFound {
				return nil, fmt.Errorf("%w: invalid storage TTL", derror.ErrInvalidParam)
			}
			plan.summary.Skipped++
			continue
		}
		entry := record{key: target + strings.TrimPrefix(key, source)}
		if ttl > 0 {
			entry.deadline = started.Add(ttl)
		}
		value, err := storage.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if value == nil || !entry.deadline.IsZero() && !time.Now().Before(entry.deadline) {
			plan.summary.Skipped++
			continue
		}
		entry.value, err = snapshotValue(value)
		if err != nil {
			return nil, err
		}
		plan.records = append(plan.records, entry)
		if entry.deadline.IsZero() {
			plan.summary.Permanent++
		} else {
			plan.summary.Expiring++
		}
	}
	plan.summary.Records = len(plan.records)
	return plan, nil
}

// Summary returns a value-only copy of the prepared counts and namespaces. Summary 返回预检计数和命名空间的值副本。
func (p *Plan) Summary() Summary { return p.summary }

// Apply copies once into an empty target, without deleting source data, overwriting existing values, or rolling back partial writes; writers must remain stopped. Apply 向空目标执行一次复制，不删除源数据、不覆盖已有值、不回滚部分写入；调用方须保持停写。
func (p *Plan) Apply(ctx context.Context) (Result, error) {
	if p == nil || p.storage == nil || p.scanner == nil {
		return Result{}, fmt.Errorf("%w: plan must be created by Prepare", derror.ErrInvalidParam)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	var result Result
	if p.used {
		return result, ErrPlanUsed
	}
	p.used = true
	defer func() { p.records = nil }()
	if err := ctx.Err(); err != nil {
		return result, err
	}

	// Recheck the entire destination, including unrelated keys added since preparation. 重新检查整个目标空间，包括预检后新增的不相关键。
	keys, err := p.scanner.Keys(ctx, "*")
	if err != nil {
		return result, err
	}
	for _, key := range keys {
		if strings.HasPrefix(key, p.summary.TargetNamespace) {
			return result, ErrDestinationNotEmpty
		}
	}
	atomicStorage, atomic := p.storage.(adapter.AtomicStorage)
	for i, entry := range p.records {
		if err = ctx.Err(); err != nil {
			return result, err
		}
		if !atomic {
			value, readErr := p.storage.Get(ctx, entry.key)
			if readErr != nil {
				return result, readErr
			}
			if value != nil {
				return result, ErrDestinationNotEmpty
			}
		}
		ttl := time.Duration(0)
		if !entry.deadline.IsZero() {
			ttl = time.Until(entry.deadline)
			if ttl <= 0 {
				result.Expired++
				continue
			}
		}
		if atomic {
			var saved bool
			saved, err = atomicStorage.SetIfAbsent(ctx, entry.key, entry.value, ttl)
			if err == nil && !saved {
				return result, ErrDestinationNotEmpty
			}
		} else {
			// Ordinary adapters retain a Get/Set race; the offline stop-writes requirement is essential. 普通适配器保留 Get/Set 竞态，离线停写是必要条件。
			err = p.storage.Set(ctx, entry.key, entry.value, ttl)
		}
		if err != nil {
			return result, fmt.Errorf("migration record %d: %w", i+1, err)
		}
		result.Written++
	}
	return result, nil
}

// namespace shares builder normalization without changing authentication identity. namespace 复用 Builder 规范化，不改变认证身份。
func namespace(prefix, authType string) (string, error) {
	if strings.TrimSpace(prefix) == "" {
		return "", fmt.Errorf("%w: explicit prefix is required", derror.ErrInvalidParam)
	}
	cfg := config.DefaultConfig().SetKeyPrefix(prefix)
	if authType != "" {
		cfg.SetAuthType(authType)
	}
	if err := cfg.Validate(); err != nil {
		return "", fmt.Errorf("%w: %v", derror.ErrInvalidParam, err)
	}
	return utils.LegacyStorageNamespace(cfg.KeyPrefix, cfg.AuthType), nil
}

// snapshotValue detaches byte buffers and rejects mutable objects that the framework does not store directly. snapshotValue 分离字节缓冲区，拒绝框架不会直接存储的可变对象。
func snapshotValue(value any) (any, error) {
	switch value := value.(type) {
	case []byte:
		return bytes.Clone(value), nil
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return value, nil
	default:
		return nil, fmt.Errorf("%w: unsupported stored value type %T", derror.ErrInvalidParam, value)
	}
}
