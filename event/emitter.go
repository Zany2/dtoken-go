package event

// Emitter publishes domain events through the owning authentication instance. Emitter 通过所属认证实例发布领域事件。
type Emitter interface {
	Emit(Event, string, string, string, string, map[string]any)
}

// Emit forwards an event when an emitter is configured. Emit 在配置发布器时转发事件。
func Emit(emitter Emitter, event Event, loginID, device, deviceID, token string, extra map[string]any) {
	if emitter != nil {
		emitter.Emit(event, loginID, device, deviceID, token, extra)
	}
}
