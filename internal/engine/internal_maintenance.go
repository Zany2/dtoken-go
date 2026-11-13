package engine

import "sync"

// managerMaintenance owns coalesced work for token lifecycle generations. managerMaintenance 持有按 Token 生命周期合并的维护任务。
type managerMaintenance struct {
	maintenanceMu  sync.Mutex                       // maintenanceMu protects in-flight login maintenance tasks. maintenanceMu 保护执行中的登录维护任务。
	maintenanceSeq uint64                           // maintenanceSeq identifies token maintenance generations. maintenanceSeq 标识 Token 维护任务代次。
	maintenance    map[string]loginMaintenanceState // maintenance coalesces login maintenance by token. maintenance 按 Token 合并登录维护任务。
}

// loginMaintenanceState tracks one token maintenance generation and its latest activity. loginMaintenanceState 跟踪 Token 维护任务代次及最新活跃时间。
type loginMaintenanceState struct {
	generation uint64      // generation identifies the token lifecycle task. generation 标识 Token 生命周期任务。
	activeAt   int64       // activeAt stores the latest validation timestamp. activeAt 存储最近一次校验时间戳。
	record     tokenRecord // record binds merged activity to the checked lifecycle. record 将合并的活跃时间绑定到已校验生命周期。
}

// beginLoginMaintenance reserves or replaces one in-flight maintenance generation for a token. beginLoginMaintenance 为 Token 登记或替换一个执行中的维护任务代次。
func (m *Manager) beginLoginMaintenance(tokenValue string, record *tokenRecord, activeAt int64, replace bool) (uint64, bool) {
	if tokenValue == "" || record == nil || record.Revoked || m.closed.Load() {
		return 0, false
	}

	// Coalesce concurrent maintenance requests for the same token. 合并同一 Token 的并发维护请求。
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	if state, exists := m.maintenance[tokenValue]; exists && !replace {
		// Never merge activity across token lifecycles. 不合并其他 Token 生命周期的活跃时间。
		if !tokenRecordsMatchLifecycle(record, &state.record) {
			return 0, false
		}

		// Preserve the latest request activity while sharing the existing task. 复用已有任务时保留最近一次请求活跃时间。
		if activeAt > state.activeAt {
			state.activeAt = activeAt
			m.maintenance[tokenValue] = state
		}
		return 0, false
	}
	if m.maintenance == nil {
		m.maintenance = make(map[string]loginMaintenanceState)
	}
	m.maintenanceSeq++
	generation := m.maintenanceSeq
	m.maintenance[tokenValue] = loginMaintenanceState{generation: generation, activeAt: activeAt, record: *record}
	return generation, true
}

// isLoginMaintenanceCurrent reports whether a token task still owns its generation. isLoginMaintenanceCurrent 判断 Token 维护任务是否仍持有当前代次。
func (m *Manager) isLoginMaintenanceCurrent(tokenValue string, generation uint64) bool {
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	state, exists := m.maintenance[tokenValue]
	return exists && generation != 0 && state.generation == generation
}

// getLoginMaintenanceActiveAt returns the latest activity owned by a task generation. getLoginMaintenanceActiveAt 返回任务代次持有的最新活跃时间。
func (m *Manager) getLoginMaintenanceActiveAt(tokenValue string, generation uint64) (int64, bool) {
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	state, exists := m.maintenance[tokenValue]
	if !exists || generation == 0 || state.generation != generation {
		return 0, false
	}
	return state.activeAt, true
}

// finishLoginMaintenanceActiveWrite completes a stable activity write. finishLoginMaintenanceActiveWrite 完成稳定的活跃写入。
func (m *Manager) finishLoginMaintenanceActiveWrite(tokenValue string, generation uint64, writtenAt int64) bool {
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	state, exists := m.maintenance[tokenValue]
	if !exists || generation == 0 || state.generation != generation {
		return true
	}
	if state.activeAt > writtenAt {
		return false
	}

	// Delete atomically with the stable check so a later request creates a new task. 与稳定性检查原子删除，使后续请求能够创建新任务。
	delete(m.maintenance, tokenValue)
	return true
}

// finishLoginMaintenance releases a completed token maintenance generation. finishLoginMaintenance 释放已完成的 Token 维护任务代次。
func (m *Manager) finishLoginMaintenance(tokenValue string, generation uint64) {
	m.maintenanceMu.Lock()
	defer m.maintenanceMu.Unlock()
	if state, exists := m.maintenance[tokenValue]; exists && state.generation == generation {
		delete(m.maintenance, tokenValue)
	}
}

// cancelLoginMaintenance invalidates pending maintenance for a token lifecycle. cancelLoginMaintenance 使 Token 当前生命周期的待执行维护任务失效。
func (m *Manager) cancelLoginMaintenance(tokenValue string) {
	if tokenValue == "" {
		return
	}
	m.maintenanceMu.Lock()
	delete(m.maintenance, tokenValue)
	m.maintenanceMu.Unlock()
}
