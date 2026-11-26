package migration

import (
	"context"
	"slices"
	"strings"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/internal/engine"
	"github.com/Zany2/dtoken-go/internal/utils"
)

// PrepareTokenUpgrade stages credential conversion in an unversioned target; use PrepareAuthUpgrade for current-runtime cutover. PrepareTokenUpgrade 在无版本目标空间暂存凭据转换；切换到当前运行时须使用 PrepareAuthUpgrade。
func PrepareTokenUpgrade(ctx context.Context, storage adapter.Storage, opts Options, codec adapter.Codec) (*Plan, error) {
	if utils.IsNil(codec) {
		return nil, derror.ErrInvalidParam
	}
	plan, err := Prepare(ctx, storage, opts)
	if err != nil {
		return nil, err
	}
	return upgradeTokenPlan(ctx, plan, opts.AuthType, codec)
}

// upgradeTokenPlan converts captured credentials without rescanning storage. upgradeTokenPlan 不重新扫描存储地转换已捕获凭据。
func upgradeTokenPlan(ctx context.Context, plan *Plan, authType string, codec adapter.Codec) (*Plan, error) {
	cfg := config.DefaultConfig()
	if authType != "" {
		cfg.SetAuthType(authType)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	values := make(map[string]any, len(plan.records))
	indices := make(map[string]int, len(plan.records))
	for i, entry := range plan.records {
		key := strings.TrimPrefix(entry.key, plan.summary.TargetNamespace)
		values[key], indices[key] = entry.value, i
	}
	changes, err := engine.UpgradeTokenSnapshot(ctx, codec, cfg.AuthType, values)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(changes))
	for key := range changes {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		change := changes[key]
		if i, exists := indices[key]; exists {
			plan.records[i].value = change.Value
			if strings.HasPrefix(key, config.TokenKeyPrefix) {
				plan.summary.UpgradedTokens++
			} else {
				plan.summary.UpdatedRefreshTokens++
			}
			continue
		}
		original := plan.records[indices[change.TTLSource]]
		plan.records = append(plan.records, record{key: plan.summary.TargetNamespace + key, value: change.Value, deadline: original.deadline})
		plan.summary.AddedTokenExtras++
		plan.summary.Records++
		if original.deadline.IsZero() {
			plan.summary.Permanent++
		} else {
			plan.summary.Expiring++
		}
	}
	return plan, nil
}
