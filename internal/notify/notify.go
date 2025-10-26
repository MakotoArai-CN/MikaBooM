package notify

import (
	"fmt"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

type Notifier struct {
	enabled         bool
	cooldown        int
	lastNotify      time.Time
	lastCPUNotify   time.Time
	lastMemNotify   time.Time
	mu              sync.Mutex
}

func NewNotifier(enabled bool, cooldown int) *Notifier {
	now := time.Now().Add(-time.Duration(cooldown) * time.Second)
	return &Notifier{
		enabled:       enabled,
		cooldown:      cooldown,
		lastNotify:    now,
		lastCPUNotify: now,
		lastMemNotify: now,
	}
}

// ============ CPU 相关通知 ============

func (n *Notifier) NotifyCPUWorkStart(threshold int) {
	if !n.enabled {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if time.Since(n.lastCPUNotify) < time.Duration(n.cooldown)*time.Second {
		return
	}

	title := "Resource Monitor - CPU计算开始"
	message := fmt.Sprintf("其他程序CPU占用低于阈值 %d%%\n开始CPU负载调整计算", threshold)
	
	err := beeep.Notify(title, message, "")
	if err != nil {
		// 忽略通知错误
		return
	}

	n.lastCPUNotify = time.Now()
}

func (n *Notifier) NotifyCPUWorkStop() {
	if !n.enabled {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if time.Since(n.lastCPUNotify) < time.Duration(n.cooldown)*time.Second {
		return
	}

	title := "Resource Monitor - CPU计算停止"
	message := "其他程序CPU占用达到阈值\n已停止CPU负载调整计算"
	
	err := beeep.Notify(title, message, "")
	if err != nil {
		// 忽略通知错误
		return
	}

	n.lastCPUNotify = time.Now()
}

// ============ 内存 相关通知 ============

func (n *Notifier) NotifyMemWorkStart(threshold int) {
	if !n.enabled {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if time.Since(n.lastMemNotify) < time.Duration(n.cooldown)*time.Second {
		return
	}

	title := "Resource Monitor - 内存计算开始"
	message := fmt.Sprintf("其他程序内存占用低于阈值 %d%%\n开始内存负载调整计算", threshold)
	
	err := beeep.Notify(title, message, "")
	if err != nil {
		// 忽略通知错误
		return
	}

	n.lastMemNotify = time.Now()
}

func (n *Notifier) NotifyMemWorkStop() {
	if !n.enabled {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if time.Since(n.lastMemNotify) < time.Duration(n.cooldown)*time.Second {
		return
	}

	title := "Resource Monitor - 内存计算停止"
	message := "其他程序内存占用达到阈值\n已停止内存负载调整计算"
	
	err := beeep.Notify(title, message, "")
	if err != nil {
		// 忽略通知错误
		return
	}

	n.lastMemNotify = time.Now()
}

// ============ 通用通知 ============

func (n *Notifier) NotifyError(errorMsg string) {
	if !n.enabled {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if time.Since(n.lastNotify) < time.Duration(n.cooldown)*time.Second {
		return
	}

	title := "Resource Monitor - 错误"
	
	err := beeep.Notify(title, errorMsg, "")
	if err != nil {
		// 忽略通知错误
		return
	}

	n.lastNotify = time.Now()
}

func (n *Notifier) NotifyInfo(message string) {
	if !n.enabled {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if time.Since(n.lastNotify) < time.Duration(n.cooldown)*time.Second {
		return
	}

	title := "Resource Monitor"
	
	err := beeep.Notify(title, message, "")
	if err != nil {
		// 忽略通知错误
		return
	}

	n.lastNotify = time.Now()
}

func (n *Notifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled
}

func (n *Notifier) IsEnabled() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.enabled
}

func (n *Notifier) SetCooldown(cooldown int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cooldown = cooldown
}

func (n *Notifier) GetCooldown() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.cooldown
}