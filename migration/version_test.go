package migration

import (
	"context"
	"errors"
	"testing"
	"time"

	dtoken "github.com/Zany2/dtoken-go"
	jsoncodec "github.com/Zany2/dtoken-go/codec/json"
	adapter "github.com/Zany2/dtoken-go/contract"
	"github.com/Zany2/dtoken-go/nonce"
	"github.com/Zany2/dtoken-go/storage/memory"
)

func TestAuthUpgradeSamePrefixMatchesRuntimeAndCapabilities(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	putEncoded(t, store, codec, "same:user:token:legacy", adapter.TokenInfo{AuthType: "user:", LoginID: "account", Timeout: -1, ActiveTimeout: -1}, 0)
	putEncoded(t, store, codec, "same:user:disable:banned", adapter.DisableInfo{DisableReason: "keep"}, 0)
	put(t, store, "same:user:nonce:challenge", time.Now().Unix(), time.Hour)
	auth, err := dtoken.NewBuilder().AuthType("user").KeyPrefix("same:").Storage(store).SetCodec(codec).AutoRenew(false).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	challenges, err := nonce.NewDefaultNonceManager("user:", "same:", store)
	if err != nil {
		t.Fatal(err)
	}
	if auth.IsLogin(ctx, "legacy") || auth.IsDisable(ctx, "banned") || challenges.Verify(ctx, "challenge") {
		t.Fatal("runtime read the unversioned namespace")
	}
	if !store.Exists(ctx, "same:user:nonce:challenge") {
		t.Fatal("legacy nonce consumed before migration")
	}

	plan, err := PrepareAuthUpgrade(ctx, store, Options{AuthType: "user", SourcePrefix: "same:", TargetPrefix: "same:"}, codec)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Summary(); got.SourceNamespace != "same:user:" || got.TargetNamespace != "same:user:v2:" || got.Records != 3 {
		t.Fatalf("summary = %+v", got)
	}
	if result, err := plan.Apply(ctx); err != nil || result.Written != 3 {
		t.Fatalf("Apply = %+v, %v", result, err)
	}
	if !auth.IsLogin(ctx, "legacy") || !auth.IsDisable(ctx, "banned") || !challenges.Verify(ctx, "challenge") {
		t.Fatal("runtime cannot read migrated auth/capability data")
	}
	if challenges.Verify(ctx, "challenge") || !store.Exists(ctx, "same:user:nonce:challenge") {
		t.Fatal("nonce consumed the source or allowed a replay")
	}
	token, err := auth.LoginID(ctx, "new-account")
	if err != nil {
		t.Fatal(err)
	}
	if !store.Exists(ctx, "same:user:v2:token:"+token) || store.Exists(ctx, "same:user:token:"+token) {
		t.Fatal("login wrote an unversioned key")
	}
}

func TestAuthUpgradeRejectsOccupiedVersionedTarget(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	put(t, store, "same:user:v2:custom:value", "keep", 0)
	plan, err := PrepareAuthUpgrade(ctx, store, Options{AuthType: "user", SourcePrefix: "same:", TargetPrefix: "same:"}, codec)
	if !errors.Is(err, ErrDestinationNotEmpty) || plan != nil {
		t.Fatalf("occupied target = %v, %v", plan, err)
	}
	value, err := store.Get(ctx, "same:user:v2:custom:value")
	if err != nil || value != "keep" {
		t.Fatalf("occupied data changed: %v, %v", value, err)
	}
}

func TestAuthUpgradeExcludesCurrentSourceSubtree(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	putEncoded(t, store, codec, "old:user:token:legacy", adapter.TokenInfo{LoginID: "account"}, time.Hour)
	put(t, store, "old:user:v2:custom:value", "current", 0)
	plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Summary(); got.Records != 1 {
		t.Fatalf("copied versioned source data: %+v", got)
	}
	if result, err := plan.Apply(ctx); err != nil || result.Written != 1 {
		t.Fatalf("Apply = %+v, %v", result, err)
	}
	if store.Exists(ctx, "new:user:v2:v2:custom:value") || !store.Exists(ctx, "old:user:v2:custom:value") {
		t.Fatal("current source subtree moved or nested")
	}
}

func TestTokenUpgradeRemainsUnversionedStaging(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	putEncoded(t, store, codec, "old:user:token:legacy", adapter.TokenInfo{LoginID: "account"}, time.Hour)
	plan, err := PrepareTokenUpgrade(ctx, store, copyOptions(), codec)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Summary(); got.TargetNamespace != "new:user:" {
		t.Fatalf("staging namespace = %+v", got)
	}
	if _, err := plan.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !store.Exists(ctx, "new:user:token:legacy") || store.Exists(ctx, "new:user:v2:token:legacy") {
		t.Fatal("staging unexpectedly performed runtime cutover")
	}
}
