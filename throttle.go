package genelet

import (
	"net/http"
	"sync"
	"time"
)

const (
	loginThrottleWindow = 5 * time.Minute
	loginThrottleMax    = 5
)

type loginThrottleEntry struct {
	Count     int
	ExpiresAt time.Time
}

var loginThrottle = struct {
	sync.Mutex
	m map[string]loginThrottleEntry
}{m: make(map[string]loginThrottleEntry)}

func loginThrottleKey(base *Base, provider, login string) string {
	return base.RoleValue + "|" + provider + "|" + login + "|" + base.GetIP()
}

func loginThrottleAllowed(base *Base, provider, login string) bool {
	key := loginThrottleKey(base, provider, login)
	now := time.Now()
	loginThrottle.Lock()
	defer loginThrottle.Unlock()
	entry, ok := loginThrottle.m[key]
	if !ok || now.After(entry.ExpiresAt) {
		return true
	}
	return entry.Count < loginThrottleMax
}

func loginThrottleSuccess(base *Base, provider, login string) {
	key := loginThrottleKey(base, provider, login)
	loginThrottle.Lock()
	delete(loginThrottle.m, key)
	loginThrottle.Unlock()
}

func loginThrottleFailure(base *Base, provider, login string) error {
	key := loginThrottleKey(base, provider, login)
	now := time.Now()
	loginThrottle.Lock()
	defer loginThrottle.Unlock()
	entry, ok := loginThrottle.m[key]
	if !ok || now.After(entry.ExpiresAt) {
		entry = loginThrottleEntry{ExpiresAt: now.Add(loginThrottleWindow)}
	}
	entry.Count++
	loginThrottle.m[key] = entry
	if entry.Count >= loginThrottleMax {
		return Err(http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests))
	}
	return nil
}
