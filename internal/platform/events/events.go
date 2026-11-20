// Package events — اتوبوس رویدادهای دامنه‌ای درون‌حافظه‌ای.
//
// طبق سند بخش ۴: ارتباط ماژول‌ها از طریق سرویس‌ها، Interfaceهای داخلی
// یا رویدادهای دامنه‌ای انجام می‌شود. این اتوبوس نسخه ساده همان الگو است؛
// اگر در آینده ماژولی (مثل History یا Notifications) باید مستقل شود،
// کافی است همین اتوبوس به صف پیام واقعی تبدیل شود.
package events

import (
	"log/slog"
	"sync"
	"time"
)

type Event struct {
	Name    string // مثل "assets.moved" یا "locations.created"
	At      time.Time
	Payload any
}

type subscriber struct {
	id string
	ch chan Event
}

type Bus struct {
	mu     sync.RWMutex
	subs   []subscriber
	logger *slog.Logger
}

func NewBus(logger *slog.Logger) *Bus {
	return &Bus{logger: logger}
}

// Subscribe یک کانال رویداد برمی‌گرداند. شنونده باید همیشه مصرف کند،
// وگرنه بافر پر می‌شود و رویدادها برای آن شنونده حذف می‌شوند.
func (b *Bus) Subscribe(id string, buffer int) <-chan Event {
	if buffer <= 0 {
		buffer = 256
	}
	sub := subscriber{id: id, ch: make(chan Event, buffer)}
	b.mu.Lock()
	b.subs = append(b.subs, sub)
	b.mu.Unlock()
	return sub.ch
}

// Publish رویداد را به همه شنونده‌ها می‌فرستد؛ غیرمسدودکننده است.
func (b *Bus) Publish(name string, payload any) {
	ev := Event{Name: name, At: time.Now(), Payload: payload}

	b.mu.RLock()
	subs := make([]subscriber, len(b.subs))
	copy(subs, b.subs)
	b.mu.RUnlock()

	for _, sub := range subs {
		select {
		case sub.ch <- ev:
		default:
			b.logger.Warn("event dropped for slow subscriber",
				"subscriber", sub.id, "event", name)
		}
	}
}
