package monitor

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

type CPUMonitor struct {
	mu          sync.RWMutex
	lastUsage   float64
	updateTime  time.Time
}

func NewCPUMonitor() *CPUMonitor {
	return &CPUMonitor{
		lastUsage:  0,
		updateTime: time.Now(),
	}
}

func (m *CPUMonitor) GetUsage() (float64, error) {
	// 获取CPU使用率（所有核心的平均值）
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, nil
	}

	usage := percentages[0]

	m.mu.Lock()
	m.lastUsage = usage
	m.updateTime = time.Now()
	m.mu.Unlock()

	return usage, nil
}

func (m *CPUMonitor) GetCachedUsage() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastUsage
}

func (m *CPUMonitor) GetCPUCount() (int, error) {
	count, err := cpu.Counts(true)
	return count, err
}

func (m *CPUMonitor) GetCPUInfo() ([]cpu.InfoStat, error) {
	return cpu.Info()
}

func (m *CPUMonitor) GetPerCoreUsage() ([]float64, error) {
	// 获取每个核心的使用率
	percentages, err := cpu.Percent(time.Second, true)
	if err != nil {
		return nil, err
	}
	return percentages, nil
}