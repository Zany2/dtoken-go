// @Author daixk 2025/12/22 15:56:00
package listener

import (
	"fmt"
	"sync"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
)

// EventData defines triggered event data EventData 定义触发事件的数据
type EventData struct {
	Event     Event          // Event stores event type Event 存储事件类型。
	AuthType  string         // AuthType stores auth system type AuthType 存储认证体系类型。
	LoginID   string         // LoginID stores user login id LoginID 存储用户登录 ID。
	Device    string         // Device stores device name Device 存储设备标识。
	DeviceId  string         // DeviceId stores device id DeviceId 存储设备 ID。
	Token     string         // Token stores auth token Token 存储认证 Token。
	Extra     map[string]any // Extra stores custom data Extra 存储额外自定义数据。
	Timestamp int64          // Timestamp stores event unix time Timestamp 存储事件触发时间戳。
}

// String returns event data string String 返回事件数据字符串表示
func (e *EventData) String() string {
	if e == nil {
		return "Event<nil>"
	}
	return fmt.Sprintf("Event{type=%s,AuthType=%s, loginID=%s, device=%s, deviceId=%s, timestamp=%d}",
		e.Event, e.AuthType, e.LoginID, e.Device, e.DeviceId, e.Timestamp)
}

// Listener defines event listener interface Listener 定义事件监听器接口
type Listener interface {
	// OnEvent handles triggered event OnEvent 处理被触发的事件。
	OnEvent(data *EventData)
}

// ListenerFunc defines listener function adapter ListenerFunc 定义监听器函数适配器
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

// ListenerConfig defines listener config ListenerConfig 定义监听器配置
type ListenerConfig struct {
	Async    bool   // Async controls async execution Async 控制是否异步执行。
	Priority int    // Priority stores listener priority Priority 存储监听器优先级。
	ID       string // ID stores listener unique id ID 存储监听器唯一标识。
}

type listenerEntry struct {
	listener Listener
	config   ListenerConfig
}

// EventFilter defines event filter function EventFilter 定义事件过滤器函数
type EventFilter func(data *EventData) bool

// EventStats defines event statistics EventStats 定义事件统计信息
type EventStats struct {
	TotalTriggered int64               // TotalTriggered stores total count TotalTriggered 存储事件触发总数。
	EventCounts    map[Event]int64     // EventCounts stores count by event EventCounts 存储按事件分类的计数。
	LastTriggered  map[Event]time.Time // LastTriggered stores last trigger time LastTriggered 存储最后触发时间。
}

// Manager defines event listener manager Manager 定义事件监听管理器
type Manager struct {
	mu              sync.RWMutex
	listeners       map[Event][]listenerEntry
	panicHandler    func(event Event, data *EventData, recovered any)
	listenerCounter int
	enabledEvents   map[Event]bool // enabledEvents stores enabled event map enabledEvents 存储启用事件集合。
	asyncWaitGroup  sync.WaitGroup // asyncWaitGroup waits async listeners asyncWaitGroup 等待异步监听器完成。
	filters         []EventFilter  // filters stores global filters filters 存储全局事件过滤器。
	stats           *EventStats    // stats stores event stats stats 存储事件统计。
	enableStats     bool           // enableStats controls stats collection enableStats 控制是否收集统计。
	logger          adapter.Log    // logger stores log adapter logger 存储日志适配器。
}

// NewManager creates event manager NewManager 创建新的事件管理器
func NewManager(loggers ...adapter.Log) *Manager {
	var logger adapter.Log

	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	} else {
		logger = adapter.NewNopLogger()
	}

	m := &Manager{
		listeners:     make(map[Event][]listenerEntry),
		enabledEvents: nil, // enabledEvents nil means all enabled enabledEvents 为 nil 表示启用所有事件。
		filters:       make([]EventFilter, 0),
		stats: &EventStats{
			EventCounts:   make(map[Event]int64),
			LastTriggered: make(map[Event]time.Time),
		},
		enableStats: false, // enableStats false means stats disabled enableStats 为 false 表示默认不统计。
		logger:      logger,
	}

	// panicHandler binds initialized logger panicHandler 绑定已初始化的 logger。
	m.panicHandler = func(event Event, data *EventData, recovered any) {
		logger.Errorf(
			"listener.Manager: listener panic recovered, event=%s, panic=%v",
			event, recovered,
		)
	}

	return m
}

// SetPanicHandler sets panic handler SetPanicHandler 设置自定义 panic 处理器
func (m *Manager) SetPanicHandler(handler func(event Event, data *EventData, recovered any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panicHandler = handler
}

// AddFilter adds global filter AddFilter 添加全局事件过滤器
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

// EnableStats sets stats switch EnableStats 设置事件统计开关
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

// EnableEvent enables selected events EnableEvent 启用指定事件
func (m *Manager) EnableEvent(events ...Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(events) == 0 {
		m.enabledEvents = nil
		return
	}

	m.enabledEvents = make(map[Event]bool)
	for _, event := range events {
		m.enabledEvents[event] = true
	}
}

// DisableEvent disables selected events DisableEvent 禁用指定事件
func (m *Manager) DisableEvent(events ...Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.enabledEvents == nil {
		m.enabledEvents = make(map[Event]bool)
		// Add built-in and existing events 加入内置事件和当前已存在的事件。
		for _, event := range KnownEvents {
			m.enabledEvents[event] = true
		}
		for event := range m.listeners {
			m.enabledEvents[event] = true
		}
	}

	for _, event := range events {
		delete(m.enabledEvents, event)
	}
}

// IsEventEnabled checks event enable state IsEventEnabled 检查事件是否启用
func (m *Manager) IsEventEnabled(event Event) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.isEventEnabledLocked(event)
}

// Register registers listener with default config Register 使用默认配置注册监听器
func (m *Manager) Register(event Event, listener Listener) string {
	return m.RegisterWithConfig(event, listener, ListenerConfig{
		Async:    true,
		Priority: 0,
	})
}

// RegisterWithConfig registers listener with config RegisterWithConfig 使用自定义配置注册监听器
func (m *Manager) RegisterWithConfig(event Event, listener Listener, config ListenerConfig) string {
	if listener == nil {
		return ""
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate unique ID if not provided 自动生成唯一监听器 ID。
	if config.ID == "" {
		m.listenerCounter++
		config.ID = fmt.Sprintf("listener_%d", m.listenerCounter)
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

// RegisterFunc registers function listener RegisterFunc 注册函数监听器
func (m *Manager) RegisterFunc(event Event, handler func(data *EventData)) string {
	if handler == nil {
		return ""
	}
	return m.Register(event, ListenerFunc(handler))
}

// RegisterFuncWithConfig registers function listener with config RegisterFuncWithConfig 使用配置注册函数监听器
func (m *Manager) RegisterFuncWithConfig(event Event, handler func(data *EventData), config ListenerConfig) string {
	if handler == nil {
		return ""
	}
	return m.RegisterWithConfig(event, ListenerFunc(handler), config)
}

// Unregister removes listener by id Unregister 根据 ID 移除监听器
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

// sortListeners sorts listeners by priority sortListeners 按优先级降序排序监听器
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

// Trigger dispatches event to listeners Trigger 将事件分发给已注册监听器
func (m *Manager) Trigger(data *EventData) {
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
		if filter != nil && !filter(data) {
			return
		}
	}

	if enableStats {
		m.recordStats(data.Event)
	}

	extraInfo := ""
	if len(data.Extra) > 0 {
		extraInfo = fmt.Sprintf(", extra=%+v", data.Extra)
	}
	if logger != nil {
		logger.Infof(
			"listener.Manager.Trigger: event triggered, event=%s, authType=%s, loginID=%s, device=%s, deviceId=%s, token=%s, timestamp=%d, listeners=%d%s",
			data.Event,
			data.AuthType,
			data.LoginID,
			data.Device,
			data.DeviceId,
			data.Token,
			data.Timestamp,
			len(listenersToCall),
			extraInfo,
		)
	}

	for _, entry := range listenersToCall {
		if entry.config.Async {
			m.asyncWaitGroup.Add(1)
			go m.safeCall(entry.listener, data, &m.asyncWaitGroup)
		} else {
			m.safeCall(entry.listener, data, nil)
		}
	}
}

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
	if listeners, ok := m.listeners[EventAll]; ok {
		listenersToCall = append(listenersToCall, listeners...)
	}

	filters := append([]EventFilter(nil), m.filters...)
	return listenersToCall, filters, m.enableStats, m.logger, true
}

func (m *Manager) recordStats(event Event) {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.stats.TotalTriggered++
	m.stats.EventCounts[event]++
	m.stats.LastTriggered[event] = now
}

func (m *Manager) isEventEnabledLocked(event Event) bool {
	if m.enabledEvents == nil {
		return true
	}
	return m.enabledEvents[event] || m.enabledEvents[EventAll]
}

// TriggerAsync triggers event asynchronously TriggerAsync 异步触发事件并立即返回
func (m *Manager) TriggerAsync(data *EventData) {
	if data == nil {
		return
	}
	go m.Trigger(data)
}

// TriggerSync triggers event synchronously TriggerSync 同步触发事件并等待完成
func (m *Manager) TriggerSync(data *EventData) {
	m.Trigger(data)
	m.Wait()
}

// safeCall executes listener safely safeCall 安全执行监听器并恢复 panic
func (m *Manager) safeCall(listener Listener, data *EventData, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if listener == nil || data == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			m.mu.RLock()
			handler := m.panicHandler
			m.mu.RUnlock()

			if handler != nil {
				handler(data.Event, data, r)
			}
		}
	}()

	listener.OnEvent(data)
}

// Wait waits async listeners Wait 等待所有异步监听器完成
func (m *Manager) Wait() {
	m.asyncWaitGroup.Wait()
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
