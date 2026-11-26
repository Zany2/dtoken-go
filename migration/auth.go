package migration

import (
	"context"
	"strings"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// PrepareAuthUpgrade converts unversioned auth data into the current versioned namespace; all writers must remain stopped through cutover. PrepareAuthUpgrade 将无版本认证数据转换到当前版本空间；所有写入方须保持停写至切换完成。
func PrepareAuthUpgrade(ctx context.Context, storage adapter.Storage, opts Options, codec adapter.Codec) (*Plan, error) {
	if utils.IsNil(codec) {
		return nil, derror.ErrInvalidParam
	}
	source, err := namespace(opts.SourcePrefix, opts.AuthType)
	if err != nil {
		return nil, err
	}
	target, err := namespace(opts.TargetPrefix, opts.AuthType)
	if err != nil {
		return nil, err
	}

	// A versioned target can share its configured prefix with the legacy source; existing target data is still rejected. 版本化目标可与旧源共用配置前缀，但仍拒绝已有目标数据。
	plan, err := prepareNamespaces(ctx, storage, source, target+utils.StorageVersionSegment, source+utils.StorageVersionSegment)
	if err != nil {
		return nil, err
	}
	plan, err = upgradeTokenPlan(ctx, plan, opts.AuthType, codec)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any)
	for _, entry := range plan.records {
		key := strings.TrimPrefix(entry.key, plan.summary.TargetNamespace)
		if strings.HasPrefix(key, engine.DisableKeyPrefix) {
			values[key] = entry.value
		}
	}
	changes, err := engine.UpgradeDisableSnapshot(ctx, codec, values)
	if err != nil {
		return nil, err
	}

	// Rename only target snapshot entries, retaining each source deadline and leaving storage untouched. 仅重命名目标快照条目，沿用各自源截止时间，不修改存储。
	for i := range plan.records {
		key := strings.TrimPrefix(plan.records[i].key, plan.summary.TargetNamespace)
		if change, exists := changes[key]; exists {
			plan.records[i].key = plan.summary.TargetNamespace + change.Key
			plan.records[i].value = change.Value
			plan.summary.UpgradedRestrictions++
		}
	}
	return plan, nil
}
