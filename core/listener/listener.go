// @Author daixk 2025/12/22 15:56:00
package listener

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// EventData defines triggered event data EventData 定义触发事件的数据。
type EventData struct {
	Event     Event          // Event stores event type Event 存储事件类型。
	AuthType  string         // AuthType stores auth system type AuthType 存储认证体系类型。
	LoginID   string         // LoginID stores user login id LoginID 存储用户登录 ID。
	Device    string         // Device stores device name Device 存储设备标识。
	DeviceID  string         // DeviceID stores device id DeviceID 存储设备 ID。
	Token     string         // Token stores auth token Token 存储认证 Token。
	Extra     map[string]any // Extra stores custom data Extra 存储额外自定义数据。
	Timestamp int64          // Timestamp stores event unix time Timestamp 存储事件触发时间戳。
}

// String returns event data string String 返回事件数据字符串表示。
func (e *EventData) String() string {
	if e == nil {
		return "Event<nil>"
	}
	return fmt.Sprintf("Event{type=%s,AuthType=%s, loginID=%s, device=%s, deviceId=%s, timestamp=%d}",
		e.Event, e.AuthType, e.LoginID, e.Device, e.DeviceID, e.Timestamp)
}

// cloneEventData copies event fields and the top-level Extra map. cloneEventData 复制事件字段和顶层 Extra 映射。
func cloneEventData(data *EventData) *EventData {
	if data == nil {
		return nil
	}

	cloned := *data
	if data.Extra != nil {
		cloned.Extra = make(map[string]any, len(data.Extra))
		for key, value := range data.Extra {
			cloned.Extra[key] = value
		}
	}
	return &cloned
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
	Async    bool   // Async controls async execution; asynchronous completion order is not guaranteed. Async 控制是否异步执行，异步完成顺序不受保证。
	Priority int    // Priority stores dispatch priority; higher values are visited first. Priority 存储分发优先级，数值越高越先遍历。
	ID       string // ID stores listener unique id ID 存储监听器唯一标识。
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
	TotalTriggered int64               // TotalTriggered stores total count TotalTriggered 存储事件触发总数。
	EventCounts    map[Event]int64     // EventCounts stores count by event EventCounts 存储按事件分类的计数。
	LastTriggered  map[Event]time.Time // LastTriggered stores last trigger time LastTriggered 存储最后触发时间。
}

// Manager defines event listener manager Manager 定义事件监听管理器。
type Manager struct {
	mu              sync.RWMutex
	listeners       map[Event][]listenerEntry
	panicHandler    func(event Event, data *EventData, recovered any)
	listenerCounter int
	eventStates     map[Event]bool // eventStates stores explicit event enable overrides. eventStates 存储事件启用状态覆盖值。
	defaultEnabled  bool           // defaultEnabled applies when no event override matches. defaultEnabled 用于没有匹配覆盖值的事件。
	asyncMu         sync.Mutex     // asyncMu protects asynchronous task accounting. asyncMu 保护异步任务计数。
	asyncCond       *sync.Cond     // asyncCond wakes lifecycle waiters when tasks finish. asyncCond 在任务完成时唤醒生命周期等待方。
	asyncTasks      int            // asyncTasks counts accepted asynchronous tasks. asyncTasks 统计已接收的异步任务。
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
		eventStates:    nil,
		defaultEnabled: true,
		filters:        make([]EventFilter, 0),
		stats: &EventStats{
			EventCounts:   make(map[Event]int64),
			LastTriggered: make(map[Event]time.Time),
		},
		enableStats: false, // enableStats false means stats disabled enableStats 为 false 表示默认不统计。
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
	m.filters = append(m.filters, filter)
}

// ClearFilters clears all filters ClearFilters 清除所有事件过滤器
func (m *Manager) ClearFilters() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.filters = make([]EventFilter, 0)
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

	if m.listeners[event] == nil {
		m.listeners[event] = make([]listenerEntry, 0)
	}

	entry := listenerEntry{
		listener: listener,
		config:   config,
	}

	m.listeners[event] = append(m.listeners[event], entry)

	// Sort by priority 排序监听器优先级。
	m.sortListeners(event)

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
				m.listeners[event] = append(entries[:i], entries[i+1:]...)
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

// Trigger dispatches an event on the current goroutine; asynchronous listeners are started but not awaited. Trigger 在当前 goroutine 分发事件，异步监听器启动后不等待其完成。
func (m *Manager) Trigger(data *EventData) {
	m.trigger(cloneEventData(data), nil)
}

// trigger dispatches one owned event snapshot and optionally tracks this dispatch's listeners. trigger 分发一个独立事件快照，并按需跟踪本次分发的监听器。
func (m *Manager) trigger(data *EventData, localWaitGroup *sync.WaitGroup) {
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
		if entry.config.Async {
			m.startAsync(localWaitGroup, func() {
				m.safeCall(listener, listenerData)
			})
		} else {
			m.safeCall(listener, listenerData)
		}
	}
}

// snapshot copies listeners and runtime settings for triggering snapshot 复制触发事件所需的监听器和运行时设置。
func (m *Manager) snapshot(event Event) ([]listenerEntry, []EventFilter, bool, adapter.Log, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isEventEnabledLocked(event) {
		return nil, nil, false, m.logger, false
	}

	listenersToCall := make([]listenerEntry, 0, len(m.listeners[event])+len(m.listeners[EventAll]))
	if listeners, ok := m.listeners[event]; ok {
		listenersToCall = append(listenersToCall, listeners...)
	}
	if event != EventAll {
		if listeners, ok := m.listeners[EventAll]; ok {
			listenersToCall = append(listenersToCall, listeners...)
		}
	}
	sort.SliceStable(listenersToCall, func(i, j int) bool {
		return listenersToCall[i].config.Priority > listenersToCall[j].config.Priority
	})

	filters := append([]EventFilter(nil), m.filters...)
	return listenersToCall, filters, m.enableStats, m.logger, true
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

// TriggerAsync snapshots and dispatches an event asynchronously, and the dispatch is tracked by Wait. TriggerAsync 快照并异步分发事件，整个分发过程会被 Wait 跟踪。
func (m *Manager) TriggerAsync(data *EventData) {
	snapshot := cloneEventData(data)
	if snapshot == nil {
		return
	}

	m.startAsync(nil, func() {
		m.trigger(snapshot, nil)
	})
}

// TriggerSync dispatches an event and waits for asynchronous listeners started directly by this dispatch. TriggerSync 分发事件并等待本次分发直接启动的异步监听器完成。
func (m *Manager) TriggerSync(data *EventData) {
	snapshot := cloneEventData(data)
	if snapshot == nil {
		return
	}

	var localWaitGroup sync.WaitGroup
	m.trigger(snapshot, &localWaitGroup)
	localWaitGroup.Wait()
}

// startAsync registers and starts one asynchronous task. startAsync 登记并启动一个异步任务。
func (m *Manager) startAsync(localWaitGroup *sync.WaitGroup, task func()) {
	if localWaitGroup != nil {
		localWaitGroup.Add(1)
	}

	m.asyncMu.Lock()
	m.asyncTasks++
	m.asyncMu.Unlock()

	go func() {
		if localWaitGroup != nil {
			defer localWaitGroup.Done()
		}
		defer m.finishAsync()
		task()
	}()
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
}

// ClearEvent clears event listeners ClearEvent 清除指定事件的所有监听器
func (m *Manager) ClearEvent(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.listeners, event)
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
