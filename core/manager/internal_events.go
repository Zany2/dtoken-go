// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"time"

	"github.com/Zany2/dtoken-go/core/listener"
)

// triggerEvent triggers an event through the event manager. triggerEvent 通过事件管理器触发事件。
func (m *Manager) triggerEvent(event listener.Event, loginID, device, deviceID, token string, extra map[string]any) {
	// Skip when event manager is absent 事件管理器不存在时跳过。
	if m.eventManager == nil {
		return
	}

	// Build event payload 构建事件载荷
	eventData := &listener.EventData{
		Event:     event,
		AuthType:  m.config.AuthType,
		LoginID:   loginID,
		Device:    device,
		DeviceID:  deviceID,
		Token:     token,
		Extra:     cloneEventExtra(extra),
		Timestamp: time.Now().Unix(),
	}

	if m.config.AsyncEvent {
		// Dispatch event asynchronously 异步分发事件
		m.submitAsync("triggerEvent", func() {
			// Trigger event in async task 在异步任务中触发事件。
			m.eventManager.Trigger(eventData)
		})
		return
	}

	// Dispatch event synchronously 同步分发事件
	m.eventManager.Trigger(eventData)
}

// cloneEventExtra snapshots mutable built-in event payloads before async submission. cloneEventExtra 在异步提交前快照内置的可变事件载荷。
func cloneEventExtra(extra map[string]any) map[string]any {
	if extra == nil {
		return nil
	}

	cloned := make(map[string]any, len(extra))
	for key, value := range extra {
		if values, ok := value.([]string); ok {
			if values == nil {
				cloned[key] = []string(nil)
				continue
			}
			copied := make([]string, len(values))
			copy(copied, values)
			cloned[key] = copied
			continue
		}
		cloned[key] = value
	}
	return cloned
}

// triggerTerminalLifecycleEvents emits terminal events after account writes are unlocked. triggerTerminalLifecycleEvents 在账号写锁释放后触发终端生命周期事件。
func (m *Manager) triggerTerminalLifecycleEvents(loginID string, events []terminalLifecycleEvent) {
	for _, lifecycleEvent := range events {
		var event listener.Event
		switch lifecycleEvent.state {
		case TokenStateLogout:
			event = listener.EventLogout
		case TokenStateKickOut:
			event = listener.EventKickout
		case TokenStateReplaced:
			event = listener.EventReplace
		case TokenStateActiveTimeout:
			event = listener.EventActiveTimeout
		}
		if event == "" {
			continue
		}

		terminal := lifecycleEvent.terminal
		m.triggerEvent(event, loginID, terminal.Device, terminal.DeviceID, terminal.Token, nil)
	}
}
