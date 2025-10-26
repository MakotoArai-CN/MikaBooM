package monitor

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
)

type MemoryMonitor struct {
	mu         sync.RWMutex
	lastUsage  float64
	updateTime time.Time
}

func NewMemoryMonitor() *MemoryMonitor {
	return &MemoryMonitor{
		lastUsage:  0,
		updateTime: time.Now(),
	}
}

func (m *MemoryMonitor) GetUsage() (float64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}

	usage := v.UsedPercent

	m.mu.Lock()
	m.lastUsage = usage
	m.updateTime = time.Now()
	m.mu.Unlock()

	return usage, nil
}

func (m *MemoryMonitor) GetCachedUsage() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastUsage
}

func (m *MemoryMonitor) GetMemoryInfo() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

func (m *MemoryMonitor) GetTotalMemory() (uint64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return v.Total, nil
}

func (m *MemoryMonitor) GetUsedMemory() (uint64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return v.Used, nil
}

func (m *MemoryMonitor) GetAvailableMemory() (uint64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return v.Available, nil
}

func (m *MemoryMonitor) GetFreeMemory() (uint64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return v.Free, nil
}