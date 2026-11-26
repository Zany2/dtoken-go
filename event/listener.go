// @Author daixk 2025/12/22 15:56:00
package event

import (
	"fmt"
	"sync"
	"time"

	adapter "github.com/Zany2/dtoken-go/contract"
)

// EventData defines triggered event data EventData 定义触发事件的数据。
type EventData struct {
	// Event stores event type Event 存储事件类型。
	Event Event
	// AuthType stores auth system type AuthType 存储认证体系类型。
	AuthType string
	// LoginID stores user login id LoginID 存储用户登录 ID。
	LoginID string
	// Device stores device name Device 存储设备标识。
	Device string
	// DeviceID stores device id DeviceID 存储设备 ID。
	DeviceID string
	// Token stores auth token Token 存储认证 Token。
	Token string
	// Extra stores custom data Extra 存储额外自定义数据。
	Extra map[string]any
	// Timestamp stores event unix time Timestamp 存储事件触发时间戳。
	Timestamp int64
}

// String returns event data string String 返回事件数据字符串表示。
func (e *EventData) String() string {
	if e == nil {
		return "Event<nil>"
	}
	return fmt.Sprintf("Event{type=%s,AuthType=%s, loginID=%s, device=%s, deviceId=%s, timestamp=%d}",
		e.Event, e.AuthType, e.LoginID, e.Device, e.DeviceID, e.Timestamp)
}

// cloneEventData copies event fields, the Extra map, and built-in slice payloads. cloneEventData 复制事件字段、Extra 映射和内置切片载荷。
func cloneEventData(data *EventData) *EventData {
	if data == nil {
		return nil
	}

	cloned := *data
	if data.Extra != nil {
		cloned.Extra = make(map[string]any, len(data.Extra))
		for key, value := range data.Extra {
			cloned.Extra[key] = cloneEventExtraValue(value)
		}
	}
	return &cloned
}

// cloneEventExtraValue copies mutable value types used by built-in event payloads. cloneEventExtraValue 复制内置事件载荷使用的可变值类型。
func cloneEventExtraValue(value any) any {
	switch typed := value.(type) {
	case []string:
		if typed == nil {
			return []string(nil)
		}
		cloned := make([]string, len(typed))
		copy(cloned, typed)
		return cloned
	default:
		return value
	}
}

// Listener defines event listener interface Listener 定义事件监听器接口。
type Listener interface {
	// OnEvent handles triggered event OnEvent 处理被触发的事件。
	OnEvent(data *EventData)
}

// ListenerFunc defines listener function adapter ListenerFunc 定义监听器函数适配器。
type ListenerFunc func(data *EventData)

// Interface assertion keeps listener contract checked at compile time 接口断言在编译期检查监听器契约
var _ Listener = ListenerFunc(nil)

// OnEvent implements listener interface OnEvent 实现 Listener 接口
func (f ListenerFunc) OnEvent(data *EventData) {
	if f == nil {
		return
	}
	f(data)
}

// ListenerConfig defines listener config ListenerConfig 定义监听器配置。
type ListenerConfig struct {
	// Async requests pool execution; absent or rejecting pools fall back synchronously. Async 请求协程池执行，未配置池或池拒绝时同步回退。
	Async bool
	// Priority stores dispatch priority; higher values are visited first. Priority 存储分发优先级，数值越高越先遍历。
	Priority int
	// ID stores listener unique id ID 存储监听器唯一标识。
	ID string
}

// listenerEntry stores listener and config together listenerEntry 同时存储监听器和配置。
type listenerEntry struct {
	listener Listener
	config   ListenerConfig
}

// EventFilter defines event filter function EventFilter 定义事件过滤器函数。
type EventFilter func(data *EventData) bool

// EventStats defines event statistics EventStats 定义事件统计信息
type EventStats struct {
	// TotalTriggered stores total count TotalTriggered 存储事件触发总数。
	TotalTriggered int64
	// EventCounts stores count by event EventCounts 存储按事件分类的计数。
	EventCounts map[Event]int64
	// LastTriggered stores last trigger time LastTriggered 存储最后触发时间。
	LastTriggered map[Event]time.Time
}

// Manager defines event listener manager Manager 定义事件监听管理器。
type Manager struct {
	mu              sync.RWMutex
	listeners       map[Event][]listenerEntry
	dispatch        map[Event][]listenerEntry // dispatch stores immutable merged lists for registered concrete events. dispatch 保存已注册具体事件的不可变合并列表。
	panicHandler    func(event Event, data *EventData, recovered any)
	listenerCounter int
	eventStates     map[Event]bool // eventStates stores explicit event enable overrides. eventStates 存储事件启用状态覆盖值。
	defaultEnabled  bool           // defaultEnabled applies when no event override matches. defaultEnabled 用于没有匹配覆盖值的事件。
	asyncMu         sync.Mutex     // asyncMu protects asynchronous task accounting. asyncMu 保护异步任务计数。
	asyncCond       *sync.Cond     // asyncCond wakes lifecycle waiters when tasks finish. asyncCond 在任务完成时唤醒生命周期等待方。
	asyncTasks      int            // asyncTasks counts accepted asynchronous tasks. asyncTasks 统计已接收的异步任务。
	asyncPool       adapter.Pool   // asyncPool is caller-owned and protected by asyncMu. asyncPool 由调用方持有并由 asyncMu 保护。
	filters         []EventFilter  // filters stores global filters filters 存储全局事件过滤器。
	stats           *EventStats    // stats stores event stats stats 存储事件统计。
	enableStats     bool           // enableStats controls stats collection enableStats 控制是否收集统计。
	logger          adapter.Log    // logger stores log adapter logger 存储日志适配器。
}

// NewManager creates event manager NewManager 创建新的事件管理器。
func NewManager(loggers ...adapter.Log) *Manager {
	var logger adapter.Log

	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	} else {
		logger = adapter.NewNopLogger()
	}

	m := &Manager{
		listeners:      make(map[Event][]listenerEntry),
		dispatch:       make(map[Event][]listenerEntry),
		eventStates:    nil,
		defaultEnabled: true,
		filters:        make([]EventFilter, 0),
		stats: &EventStats{
			EventCounts:   make(map[Event]int64),
			LastTriggered: make(map[Event]time.Time),
		},
		// enableStats false means stats disabled enableStats 为 false 表示默认不统计。
		enableStats: false,
		logger:      logger,
	}
	m.asyncCond = sync.NewCond(&m.asyncMu)

	// panicHandler binds initialized logger panicHandler 绑定已初始化 logger。
	m.panicHandler = func(event Event, data *EventData, recovered any) {
		logger.Errorf(
			"listener.Manager: event callback panic recovered, event=%s, panic=%v",
			event, recovered,
		)
	}

	return m
}

// SetAsyncPool binds a caller-owned nonblocking pool before serving; nil or rejection executes synchronously, TriggerSync bypasses it, and the caller stops it after producers and tasks drain. SetAsyncPool 在服务启动前绑定调用方持有的非阻塞池；nil 或拒绝时同步执行，TriggerSync 绕过池，调用方须在生产方及任务排空后关闭池。
func (m *Manager) SetAsyncPool(pool adapter.Pool) {
	m.asyncMu.Lock()
	defer m.asyncMu.Unlock()
	m.asyncPool = pool
}

// SetPanicHandler sets the listener and filter panic handler. SetPanicHandler 设置监听器和过滤器的 panic 处理器。
func (m *Manager) SetPanicHandler(handler func(event Event, data *EventData, recovered any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panicHandler = handler
}

// AddFilter adds a global filter; a panic rejects the event and is sent to the panic handler. AddFilter 添加全局事件过滤器；panic 会拒绝当前事件并交给 panic 处理器。
func (m *Manager) AddFilter(filter EventFilter) {
	if filter == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	filters := make([]EventFilter, len(m.filters)+1)
	copy(filters, m.filters)
	filters[len(m.filters)] = filter
	m.filters = filters
}

// ClearFilters clears all filters ClearFilters 清除所有事件过滤器
func (m *Manager) ClearFilters() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.filters = make([]EventFilter, 0)
}

// NeedsDispatch checks current enablement and observers, including wildcard listeners, filters, statistics and non-noop logging; concurrent configuration may change the result. NeedsDispatch 检查当前启用状态及观察需求，包括通配监听器、过滤器、统计和非空日志；并发配置变更可能改变结果。
func (m *Manager) NeedsDispatch(event Event) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.needsDispatchLocked(event)
}

// needsDispatchLocked preserves every observable effect when deciding whether to skip an event. needsDispatchLocked 决定跳过事件时保留所有可观察行为。
func (m *Manager) needsDispatchLocked(event Event) bool {
	if !m.isEventEnabledLocked(event) {
		return false
	}
	_, noop := m.logger.(*adapter.NopLogger)
	return len(m.listeners[event]) > 0 || len(m.listeners[EventAll]) > 0 ||
		len(m.filters) > 0 || m.enableStats || m.logger != nil && !noop
}

// EnableStats sets stats switch EnableStats 设置事件统计开关。
func (m *Manager) EnableStats(enable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enableStats = enable
}

// GetStats returns stats copy GetStats 返回事件统计副本
func (m *Manager) GetStats() EventStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := EventStats{
		TotalTriggered: m.stats.TotalTriggered,
		EventCounts:    make(map[Event]int64),
		LastTriggered:  make(map[Event]time.Time),
	}

	for event, count := range m.stats.EventCounts {
		stats.EventCounts[event] = count
	}
	for event, t := range m.stats.LastTriggered {
		stats.LastTriggered[event] = t
	}

	return stats
}

// ResetStats resets stats ResetStats 重置事件统计
func (m *Manager) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats = &EventStats{
		EventCounts:   make(map[Event]int64),
		LastTriggered: make(map[Event]time.Time),
	}
}

// EnableEvent replaces the allow-list, while no arguments restore all events. EnableEvent 替换事件白名单，不传参数时恢复全部事件。
func (m *Manager) EnableEvent(events ...Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(events) == 0 {
		m.eventStates = nil
		m.defaultEnabled = true
		return
	}

	m.eventStates = make(map[Event]bool, len(events))
	m.defaultEnabled = false
	for _, event := range events {
		m.eventStates[event] = true
	}
}

// DisableEvent disables selected events without changing unrelated events. DisableEvent 禁用指定事件且不改变其他事件状态。
func (m *Manager) DisableEvent(events ...Event) {
	if len(events) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.eventStates == nil {
		m.eventStates = make(map[Event]bool, len(events))
	}

	for _, event := range events {
		if event == EventAll {
			m.eventStates = map[Event]bool{EventAll: false}
			m.defaultEnabled = false
			continue
		}
		m.eventStates[event] = false
	}
}

// IsEventEnabled checks event enable state IsEventEnabled 检查事件是否启用。
func (m *Manager) IsEventEnabled(event Event) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.isEventEnabledLocked(event)
}

// Register registers an asynchronous listener with default priority. Register 使用默认优先级注册异步监听器。
func (m *Manager) Register(event Event, listener Listener) string {
	return m.RegisterWithConfig(event, listener, ListenerConfig{
		Async:    true,
		Priority: 0,
	})
}

// RegisterWithConfig registers a listener; duplicate explicit IDs return an empty string. RegisterWithConfig 使用配置注册监听器，重复的显式 ID 会返回空字符串。
func (m *Manager) RegisterWithConfig(event Event, listener Listener, config ListenerConfig) string {
	if listener == nil {
		return ""
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a globally unique ID or reject a duplicate explicit ID. 生成全局唯一 ID，或拒绝重复的显式 ID。
	if config.ID == "" {
		for {
			m.listenerCounter++
			config.ID = fmt.Sprintf("listener_%d", m.listenerCounter)
			if !m.hasListenerIDLocked(config.ID) {
				break
			}
		}
	} else if m.hasListenerIDLocked(config.ID) {
		return ""
	}

	entry := listenerEntry{
		listener: listener,
		config:   config,
	}

	// Publish a new slice so an in-flight dispatch keeps its original listener snapshot. 发布新切片，使进行中的分发保留原监听器快照。
	entries := make([]listenerEntry, len(m.listeners[event])+1)
	copy(entries, m.listeners[event])
	entries[len(entries)-1] = entry
	m.listeners[event] = entries

	// Sort by priority 排序监听器优先级。
	m.sortListeners(event)
	m.rebuildDispatchLocked(event)

	return config.ID
}

// RegisterFunc registers an asynchronous function listener. RegisterFunc 注册异步函数监听器。
func (m *Manager) RegisterFunc(event Event, handler func(data *EventData)) string {
	if handler == nil {
		return ""
	}
	return m.Register(event, ListenerFunc(handler))
}

// RegisterFuncWithConfig registers a function listener; duplicate explicit IDs return an empty string. RegisterFuncWithConfig 使用配置注册函数监听器，重复的显式 ID 会返回空字符串。
func (m *Manager) RegisterFuncWithConfig(event Event, handler func(data *EventData), config ListenerConfig) string {
	if handler == nil {
		return ""
	}
	return m.RegisterWithConfig(event, ListenerFunc(handler), config)
}

// Unregister removes listener by id Unregister 根据 ID 移除监听。
func (m *Manager) Unregister(listenerID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for event, entries := range m.listeners {
		for i, entry := range entries {
			if entry.config.ID == listenerID {
				remaining := make([]listenerEntry, len(entries)-1)
				copy(remaining, entries[:i])
				copy(remaining[i:], entries[i+1:])
				if len(remaining) == 0 {
					delete(m.listeners, event)
				} else {
					m.listeners[event] = remaining
				}
				m.rebuildDispatchLocked(event)
				return true
			}
		}
	}

	return false
}

// sortListeners sorts listeners by priority sortListeners 按优先级降序排序监听。
func (m *Manager) sortListeners(event Event) {
	entries := m.listeners[event]

	// Use insertion sort 使用插入排序保持稳定性。
	for i := 1; i < len(entries); i++ {
		key := entries[i]
		j := i - 1
		for j >= 0 && entries[j].config.Priority < key.config.Priority {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = key
	}
}

// rebuildDispatchLocked merges sorted listeners only when registration changes; unknown event names never grow the cache. rebuildDispatchLocked 仅在注册变化时合并已排序监听器，未知事件名不会使缓存增长。
func (m *Manager) rebuildDispatchLocked(event Event) {
	if event == EventAll {
		for registered := range m.listeners {
			if registered != EventAll {
				m.rebuildDispatchLocked(registered)
			}
		}
		return
	}
	entries := m.listeners[event]
	if len(entries) == 0 {
		delete(m.dispatch, event)
		return
	}
	wildcard := m.listeners[EventAll]
	if len(wildcard) == 0 {
		m.dispatch[event] = entries
		return
	}
	merged := make([]listenerEntry, 0, len(entries)+len(wildcard))
	i, j := 0, 0
	for i < len(entries) && j < len(wildcard) {
		// Equal priorities retain specific listeners before wildcard listeners. 相同优先级保留具体监听器先于通配监听器的顺序。
		if entries[i].config.Priority >= wildcard[j].config.Priority {
			merged = append(merged, entries[i])
			i++
		} else {
			merged = append(merged, wildcard[j])
			j++
		}
	}
	merged = append(merged, entries[i:]...)
	m.dispatch[event] = append(merged, wildcard[j:]...)
}

// hasListenerIDLocked checks ID uniqueness while m.mu is held. hasListenerIDLocked 在持有 m.mu 时检查 ID 唯一性。
func (m *Manager) hasListenerIDLocked(listenerID string) bool {
	for _, entries := range m.listeners {
		for _, entry := range entries {
			if entry.config.ID == listenerID {
				return true
			}
		}
	}
	return false
}

// Trigger dispatches on the caller; accepted pool listeners are not awaited, while rejected listeners execute inline. Trigger 在调用方分发，已被池接收的监听器不等待，拒绝的监听器内联执行。
func (m *Manager) Trigger(data *EventData) {
	if data == nil || !m.NeedsDispatch(data.Event) {
		return
	}
	m.trigger(cloneEventData(data), false)
}

// trigger dispatches one owned snapshot and optionally forces inline listeners. trigger 分发独立快照，并按需强制内联执行监听器。
func (m *Manager) trigger(data *EventData, synchronous bool) {
	if data == nil {
		return
	}

	if data.Timestamp == 0 {
		data.Timestamp = time.Now().Unix()
	}

	listenersToCall, filters, enableStats, logger, enabled := m.snapshot(data.Event)
	if !enabled {
		return
	}

	for _, filter := range filters {
		if !m.safeFilter(filter, data) {
			return
		}
	}

	if enableStats {
		m.recordStats(data.Event)
	}

	if logger != nil {
		logger.Infof(
			"listener.Manager.Trigger: event triggered, event=%s, authType=%s, timestamp=%d, listeners=%d",
			data.Event,
			data.AuthType,
			data.Timestamp,
			len(listenersToCall),
		)
	}

	for _, entry := range listenersToCall {
		listener := entry.listener
		listenerData := cloneEventData(data)
		if entry.config.Async && !synchronous {
			m.startAsync(func() {
				m.safeCall(listener, listenerData)
			}, nil)
		} else {
			m.safeCall(listener, listenerData)
		}
	}
}

// snapshot borrows immutable listener and filter lists while copying scalar runtime settings. snapshot 复用不可变监听器及过滤器列表，并复制运行时标量设置。
func (m *Manager) snapshot(event Event) ([]listenerEntry, []EventFilter, bool, adapter.Log, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.needsDispatchLocked(event) {
		return nil, nil, false, nil, false
	}

	listenersToCall := m.dispatch[event]
	if len(listenersToCall) == 0 {
		listenersToCall = m.listeners[EventAll]
	}
	logger := m.logger
	if _, noop := logger.(*adapter.NopLogger); noop {
		logger = nil
	}
	return listenersToCall, m.filters, m.enableStats, logger, true
}

// recordStats records one event trigger recordStats 记录一次事件触发统计。
func (m *Manager) recordStats(event Event) {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalTriggered++
	m.stats.EventCounts[event]++
	m.stats.LastTriggered[event] = now
}

// isEventEnabledLocked checks event state under lock isEventEnabledLocked 在锁内检查事件启用状态。
func (m *Manager) isEventEnabledLocked(event Event) bool {
	if enabled, ok := m.eventStates[event]; ok {
		return enabled
	}
	if enabled, ok := m.eventStates[EventAll]; ok {
		return enabled
	}
	return m.defaultEnabled
}

// TriggerAsync snapshots the payload and default timestamp before asynchronous dispatch tracked by Wait. TriggerAsync 在异步分发前快照载荷及默认时间戳，分发过程由 Wait 跟踪。
func (m *Manager) TriggerAsync(data *EventData) {
	if data == nil || !m.NeedsDispatch(data.Event) {
		return
	}
	snapshot := cloneEventData(data)

	// Capture submission time instead of the eventual worker execution time. 记录提交时间，而非后台任务实际执行时间。
	if snapshot.Timestamp == 0 {
		snapshot.Timestamp = time.Now().Unix()
	}

	m.startAsync(func() {
		m.trigger(snapshot, false)
	}, func() {
		m.trigger(snapshot, true)
	})
}

// TriggerSync executes direct listeners synchronously in priority order without submitting or waiting on pool workers. TriggerSync 按优先级同步执行直接监听器，不提交或等待池任务。
func (m *Manager) TriggerSync(data *EventData) {
	if data == nil || !m.NeedsDispatch(data.Event) {
		return
	}
	snapshot := cloneEventData(data)

	m.trigger(snapshot, true)
}

// startAsync tracks pool execution or a synchronous fallback without creating goroutines. startAsync 跟踪池执行或同步回退，不创建 goroutine。
func (m *Manager) startAsync(task, fallback func()) {
	m.asyncMu.Lock()
	m.asyncTasks++
	pool := m.asyncPool
	m.asyncMu.Unlock()

	run := func(work func()) {
		defer m.finishAsync()
		work()
	}

	if pool != nil {
		if err := pool.Submit(func() { run(task) }); err == nil {
			return
		}
	}

	// Preserve delivery with bounded caller work when no worker accepts it. 无工作线程接收时由调用方执行，保留投递且限制资源扩张。
	if fallback != nil {
		run(fallback)
	} else {
		run(task)
	}
}

// finishAsync marks one asynchronous task complete. finishAsync 标记一个异步任务完成。
func (m *Manager) finishAsync() {
	m.asyncMu.Lock()
	defer m.asyncMu.Unlock()

	m.asyncTasks--
	if m.asyncTasks == 0 {
		m.asyncCond.Broadcast()
	}
}

// safeFilter executes a filter safely and rejects the event after a panic. safeFilter 安全执行过滤器，并在 panic 后拒绝当前事件。
func (m *Manager) safeFilter(filter EventFilter, data *EventData) (allowed bool) {
	if filter == nil {
		return true
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			allowed = false
			m.handlePanic(data.Event, data, recovered)
		}
	}()

	return filter(data)
}

// safeCall executes listener safely safeCall 安全执行监听器并恢复 panic
func (m *Manager) safeCall(listener Listener, data *EventData) {
	if listener == nil || data == nil {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			m.handlePanic(data.Event, data, recovered)
		}
	}()

	listener.OnEvent(data)
}

// handlePanic invokes the configured handler and contains handler failures. handlePanic 调用已配置的处理器并隔离处理器自身故障。
func (m *Manager) handlePanic(event Event, data *EventData, recovered any) {
	m.mu.RLock()
	handler := m.panicHandler
	logger := m.logger
	m.mu.RUnlock()
	if handler == nil {
		return
	}

	defer func() {
		if handlerPanic := recover(); handlerPanic != nil && logger != nil {
			logger.Errorf(
				"listener.Manager: panic handler panic recovered, event=%s, panic=%v",
				event, handlerPanic,
			)
		}
	}()

	handler(event, data, recovered)
}

// Wait waits until the manager has no accepted asynchronous tasks. Wait 等待 Manager 已接收的异步任务全部完成。
func (m *Manager) Wait() {
	m.asyncMu.Lock()
	defer m.asyncMu.Unlock()

	for m.asyncTasks > 0 {
		m.asyncCond.Wait()
	}
}

// Clear clears all listeners Clear 清除所有监听器
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = make(map[Event][]listenerEntry)
	m.dispatch = make(map[Event][]listenerEntry)
}

// ClearEvent clears event listeners ClearEvent 清除指定事件的所有监听器
func (m *Manager) ClearEvent(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.listeners, event)
	m.rebuildDispatchLocked(event)
}

// Count returns listener count Count 返回已注册监听器总数
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, entries := range m.listeners {
		count += len(entries)
	}
	return count
}

// CountForEvent returns event listener count CountForEvent 返回指定事件的监听器数量
func (m *Manager) CountForEvent(event Event) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.listeners[event])
}

// GetListenerIDs returns listener ids GetListenerIDs 获取指定事件的监听器 ID 列表
func (m *Manager) GetListenerIDs(event Event) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := m.listeners[event]
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.config.ID)
	}
	return ids
}

// GetAllEvents returns registered events GetAllEvents 获取所有已注册监听器的事件
func (m *Manager) GetAllEvents() []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()

	events := make([]Event, 0, len(m.listeners))
	for event := range m.listeners {
		events = append(events, event)
	}
	return events
}

// HasListeners checks event listeners HasListeners 检查指定事件是否存在监听器
func (m *Manager) HasListeners(event Event) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.listeners[event]) > 0
}
