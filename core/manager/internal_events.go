package manager

import (
	"time"

	"github.com/Zany2/dtoken-go/core/listener"
)

// triggerEvent triggers an event through the event manager. triggerEvent 通过事件管理器触发事件。
func (m *Manager) triggerEvent(event listener.Event, loginID, device, deviceId, token string, extra map[string]any) {
	if m.eventManager == nil {
		return
	}

	// Build event payload 构建事件载荷
	eventData := &listener.EventData{
		Event:     event,
		AuthType:  m.config.AuthType,
		LoginID:   loginID,
		Device:    device,
		DeviceId:  deviceId,
		Token:     token,
		Extra:     extra,
		Timestamp: time.Now().Unix(),
	}

	if m.config.AsyncEvent {
		// Dispatch event asynchronously 异步分发事件
		m.submitAsync("triggerEvent", func() {
			m.eventManager.Trigger(eventData)
		})
		return
	}

	// Dispatch event synchronously 同步分发事件
	m.eventManager.Trigger(eventData)
}
