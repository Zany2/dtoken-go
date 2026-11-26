package engine

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	listener "github.com/Zany2/dtoken-go/event"
)

// splitPermissionMatch retains the previous matching algorithm as a compatibility baseline. splitPermissionMatch 保留原匹配算法作为兼容性基线。
func splitPermissionMatch(pattern, permission string) bool {
	if pattern == PermissionWildcard || pattern == permission {
		return true
	}
	if !strings.Contains(pattern, PermissionWildcard) {
		return false
	}
	separator := PermissionSeparator
	if strings.Contains(pattern, "/") {
		separator = "/"
	}
	patterns, permissions := strings.Split(pattern, separator), strings.Split(permission, separator)
	if len(patterns) != len(permissions) {
		return false
	}
	for i, part := range patterns {
		if part != PermissionWildcard && part != permissions[i] {
			return false
		}
	}
	return true
}

// TestPermissionSegmentCompatibility checks separators, empty segments and literal partial wildcards. TestPermissionSegmentCompatibility 检查分隔符、空分段及字面量部分通配符。
func TestPermissionSegmentCompatibility(t *testing.T) {
	values := []string{"", "*", "**", "a*", "a", "b", ":", "/", "用户", "a:*:read", "a/*/read", "a:*/read", "a/*:read", "a::", "a//", "a:read:extra", "a/read/extra", "a:b:read", "a/b/read", "a:x/read", "a:", "a/"}
	for _, pattern := range values {
		for _, permission := range values {
			if got, want := defaultPermissionMatcher(pattern, permission), splitPermissionMatch(pattern, permission); got != want {
				t.Fatalf("match(%q, %q) = %v, want %v", pattern, permission, got, want)
			}
		}
	}
}

// FuzzPermissionSegmentCompatibility compares arbitrary strings with the original split semantics. FuzzPermissionSegmentCompatibility 对任意字符串比较原有拆分语义。
func FuzzPermissionSegmentCompatibility(f *testing.F) {
	for _, pair := range [][2]string{{"user:*", "user:read"}, {"user/*", "user/"}, {"a:*:read", "a::read"}, {"a:*/read", "a:x/read"}, {"*", ""}, {"用户/*", "用户/读取"}} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, pattern, permission string) {
		if got, want := defaultPermissionMatcher(pattern, permission), splitPermissionMatch(pattern, permission); got != want {
			t.Fatalf("match(%q, %q) = %v, want %v", pattern, permission, got, want)
		}
	})
}

// previousRemoveAccessValues retains the former two-set removal pipeline for comparison. previousRemoveAccessValues 保留原先双集合删除流程用于对照。
func previousRemoveAccessValues(items, values []string) ([]string, []string) {
	if len(items) == 0 || len(values) == 0 {
		return items, nil
	}
	existing := make(map[string]struct{}, len(items))
	for _, item := range items {
		existing[item] = struct{}{}
	}
	removed := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := existing[value]; value != "" && ok {
			removed = append(removed, value)
			delete(existing, value)
		}
	}
	if len(removed) == 0 {
		return items, removed
	}
	removeSet := make(map[string]struct{}, len(removed))
	for _, value := range removed {
		removeSet[value] = struct{}{}
	}
	var kept []string
	for _, item := range items {
		if _, ok := removeSet[item]; !ok {
			kept = append(kept, item)
		}
	}
	return kept, removed
}

// TestAccessRemovalCompatibility preserves list order, actual-change order and caller-owned arrays. TestAccessRemovalCompatibility 保留列表顺序、实际变更顺序及调用方数组。
func TestAccessRemovalCompatibility(t *testing.T) {
	lists := [][]string{nil, {}, {""}, {"a"}, {"a", "b", "c"}, {"b", "a", "b", "", "c"}, {"b", "missing", "a", "b", ""}, {"用户", "*", "a:*"}}
	for _, items := range lists {
		for _, values := range lists {
			beforeItems, beforeValues := slices.Clone(items), slices.Clone(values)
			kept, removed := removeAccessValues(items, values)
			wantKept, wantRemoved := previousRemoveAccessValues(items, values)
			if !reflect.DeepEqual(kept, wantKept) || !slices.Equal(removed, wantRemoved) {
				t.Fatalf("remove(%v, %v) = %v/%v, want %v/%v", items, values, kept, removed, wantKept, wantRemoved)
			}
			if !reflect.DeepEqual(items, beforeItems) || !reflect.DeepEqual(values, beforeValues) {
				t.Fatal("removal modified caller-owned input")
			}
			if len(removed) == 0 && len(items) > 0 && &kept[0] != &items[0] {
				t.Fatal("unchanged list was copied")
			}
		}
	}
	items := []string{"a", "b", "a", "c"}
	kept, removed := removeAccessValues(items, items[1:3])
	if !slices.Equal(kept, []string{"c"}) || !slices.Equal(removed, []string{"b", "a"}) || !slices.Equal(items, []string{"a", "b", "a", "c"}) {
		t.Fatalf("overlapping inputs = %v/%v, original=%v", kept, removed, items)
	}
}

// TestAccessRemovalBroadRequests checks sparse matches, duplicates and empty values in oversized requests. TestAccessRemovalBroadRequests 检查超大请求中的稀疏命中、重复值及空值。
func TestAccessRemovalBroadRequests(t *testing.T) {
	items := []string{"a", "b", "a", "", "用户"}
	missing := make([]string, 1024)
	for i := range missing {
		missing[i] = fmt.Sprintf("missing:%d", i)
	}
	for _, suffix := range [][]string{nil, {"", ""}, {"b", "b", ""}, {"用户", "a", "b", "a"}} {
		values := append(slices.Clone(missing), suffix...)
		beforeItems, beforeValues := slices.Clone(items), slices.Clone(values)
		kept, removed := removeAccessValues(items, values)
		wantKept, wantRemoved := previousRemoveAccessValues(items, values)
		if !reflect.DeepEqual(kept, wantKept) || !slices.Equal(removed, wantRemoved) {
			t.Fatalf("suffix %v: got %v/%v, want %v/%v", suffix, kept, removed, wantKept, wantRemoved)
		}
		if !reflect.DeepEqual(items, beforeItems) || !reflect.DeepEqual(values, beforeValues) {
			t.Fatal("removal modified caller-owned input")
		}
		if len(removed) == 0 && &kept[0] != &items[0] {
			t.Fatal("unchanged list was copied")
		}
	}
}

// TestAccessRemovalPublicEvents checks both access kinds and both public addressing modes. TestAccessRemovalPublicEvents 检查权限和角色两类数据及两类公开定位方式。
func TestAccessRemovalPublicEvents(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{"permissions", "roles"} {
		for _, byToken := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/token=%v", kind, byToken), func(t *testing.T) {
				mgr := newTestManager(t, nil)
				token, err := mgr.Login(ctx, "access-removal")
				if err != nil {
					t.Fatal(err)
				}
				sess, err := mgr.getSession(ctx, "access-removal")
				if err != nil {
					t.Fatal(err)
				}
				original := []string{"a", "b", "c", "b", ""}
				remove, get := mgr.RemovePermissions, mgr.GetPermissions
				event, key := listener.EventPermissionChange, listener.ExtraKeyPermissions
				id := "access-removal"
				if kind == "permissions" {
					sess.Permissions = original
					if byToken {
						remove = mgr.RemovePermissionsByToken
					}
				} else {
					sess.Roles = original
					remove, get = mgr.RemoveRoles, mgr.GetRoles
					event, key = listener.EventRoleChange, listener.ExtraKeyRoles
					if byToken {
						remove = mgr.RemoveRolesByToken
					}
				}
				if byToken {
					id = token
				}
				if err = mgr.saveToStorage(ctx, mgr.getSessionKey(sess.LoginID), *sess); err != nil {
					t.Fatal(err)
				}
				storage := &managerSetCountingStorage{Storage: mgr.storage}
				mgr.storage = storage
				var changes [][]string
				mgr.GetEventManager().RegisterFuncWithConfig(event, func(data *listener.EventData) {
					changes = append(changes, data.Extra[key].([]string))
				}, listener.ListenerConfig{Async: false})
				for i := 0; i < 2; i++ {
					if err = remove(ctx, id, []string{"b", "", "missing", "a", "b"}); err != nil {
						t.Fatal(err)
					}
				}
				if !reflect.DeepEqual(changes, [][]string{{"b", "a"}}) || storage.setCount(mgr.getSessionKey(sess.LoginID)) != 1 {
					t.Fatalf("changes = %v; writes = %d", changes, storage.setCount(mgr.getSessionKey(sess.LoginID)))
				}
				if got, err := get(ctx, sess.LoginID); err != nil || !slices.Equal(got, []string{"c", ""}) {
					t.Fatalf("remaining access values = %v, %v", got, err)
				}
			})
		}
	}
}

var accessAlgorithmMatchSink bool    // accessAlgorithmMatchSink retains benchmark match results. accessAlgorithmMatchSink 保留基准匹配结果。
var accessAlgorithmListSink []string // accessAlgorithmListSink retains benchmark list results. accessAlgorithmListSink 保留基准列表结果。

// BenchmarkPermissionSegments compares streaming and split matching on wildcard paths. BenchmarkPermissionSegments 对比通配路径的逐段与拆分匹配。
func BenchmarkPermissionSegments(b *testing.B) {
	for _, pattern := range []string{"tenant:*:record:*", "tenant/*/record/*"} {
		permission := strings.ReplaceAll(pattern, "*", "read")
		for _, method := range []struct {
			name  string
			match func(string, string) bool
		}{{"segments", defaultPermissionMatcher}, {"split", splitPermissionMatch}} {
			b.Run(pattern+"/"+method.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					accessAlgorithmMatchSink = method.match(pattern, permission)
				}
			})
		}
	}
}

// BenchmarkAccessRemoval compares narrow and broad requests against small and large account lists. BenchmarkAccessRemoval 对比窄范围及大批次请求在小型与大型账号列表中的成本。
func BenchmarkAccessRemoval(b *testing.B) {
	for _, size := range []int{8, 1024} {
		items := make([]string, size)
		for i := range items {
			items[i] = fmt.Sprintf("permission:%d", i)
		}
		largeRequest := make([]string, size+1024)
		for i := range largeRequest {
			largeRequest[i] = fmt.Sprintf("permission:%d", i)
		}
		for batch, values := range [][]string{{"missing"}, {items[0], items[size-1]}, items, largeRequest} {
			for _, method := range []struct {
				name   string
				remove func([]string, []string) ([]string, []string)
			}{{"request set", removeAccessValues}, {"previous", previousRemoveAccessValues}} {
				b.Run(fmt.Sprintf("%d/batch=%d/%s", size, batch, method.name), func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						accessAlgorithmListSink, _ = method.remove(items, values)
					}
				})
			}
		}
	}
}
