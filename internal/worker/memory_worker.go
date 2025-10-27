package worker

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
)

type MemoryWorker struct {
	threshold      int
	running        atomic.Bool
	usage          atomic.Value
	allocatedMem   [][]byte
	mu             sync.Mutex
	stopChan       chan struct{}
	targetSize     atomic.Int64
	totalMemory    int64
	lastAdjustTime time.Time
	adjustMutex    sync.Mutex
}

func NewMemoryWorker(threshold int) *MemoryWorker {
	// 动态获取系统总内存
	totalMem := int64(16 * 1024 * 1024 * 1024) // 默认值 16GB
	if v, err := mem.VirtualMemory(); err == nil {
		totalMem = int64(v.Total)
	}

	w := &MemoryWorker{
		threshold:      threshold,
		allocatedMem:   make([][]byte, 0),
		stopChan:       make(chan struct{}),
		totalMemory:    totalMem,
		lastAdjustTime: time.Now(),
	}
	w.usage.Store(0.0)
	w.targetSize.Store(0)
	return w
}

func (w *MemoryWorker) Start() {
	if w.running.Load() {
		return
	}

	w.running.Store(true)
	w.stopChan = make(chan struct{})

	go w.work()
}

func (w *MemoryWorker) Stop() {
	if !w.running.Load() {
		return
	}

	w.running.Store(false)
	close(w.stopChan)

	time.Sleep(100 * time.Millisecond)

	w.mu.Lock()
	w.allocatedMem = make([][]byte, 0)
	w.mu.Unlock()

	w.usage.Store(0.0)
	w.targetSize.Store(0)
}

func (w *MemoryWorker) work() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			targetSize := w.targetSize.Load()
			currentSize := w.getCurrentAllocatedSize()

			diff := targetSize - currentSize

			if diff > 0 {
				w.allocateMemory(diff)
			} else if diff < 0 {
				w.freeMemory(-diff)
			}
		}
	}
}

func (w *MemoryWorker) allocateMemory(size int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	chunkSize := int64(10 * 1024 * 1024) // 10MB 每块
	chunks := size / chunkSize

	for i := int64(0); i < chunks; i++ {
		chunk := make([]byte, chunkSize)
		// 写入数据以确保真实分配
		for j := 0; j < len(chunk); j += 4096 {
			chunk[j] = byte(j % 256)
		}
		w.allocatedMem = append(w.allocatedMem, chunk)
	}

	remainder := size % chunkSize
	if remainder > 0 {
		chunk := make([]byte, remainder)
		for j := 0; j < len(chunk); j += 4096 {
			chunk[j] = byte(j % 256)
		}
		w.allocatedMem = append(w.allocatedMem, chunk)
	}
}

func (w *MemoryWorker) freeMemory(size int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	freed := int64(0)
	newMem := make([][]byte, 0)

	for i := len(w.allocatedMem) - 1; i >= 0; i-- {
		if freed >= size {
			newMem = append([][]byte{w.allocatedMem[i]}, newMem...)
		} else {
			freed += int64(len(w.allocatedMem[i]))
		}
	}

	w.allocatedMem = newMem
}

func (w *MemoryWorker) getCurrentAllocatedSize() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()

	total := int64(0)
	for _, chunk := range w.allocatedMem {
		total += int64(len(chunk))
	}

	return total
}

func (w *MemoryWorker) AdjustLoad(currentWorkerUsage, targetWorkerUsage float64) {
	if !w.running.Load() {
		return
	}

	w.adjustMutex.Lock()
	defer w.adjustMutex.Unlock()

	if time.Since(w.lastAdjustTime) < 2*time.Second {
		return
	}

	w.lastAdjustTime = time.Now()

	totalMemory := w.totalMemory
	if totalMemory == 0 {
		// 重新获取系统内存
		if v, err := mem.VirtualMemory(); err == nil {
			totalMemory = int64(v.Total)
			w.totalMemory = totalMemory
		} else {
			totalMemory = int64(16 * 1024 * 1024 * 1024) // 默认值
		}
	}

	targetBytes := int64(float64(totalMemory) * targetWorkerUsage / 100.0)

	maxAdjust := int64(512 * 1024 * 1024) // 每次最大调整 512MB
	currentBytes := w.getCurrentAllocatedSize()
	diff := targetBytes - currentBytes

	if diff > maxAdjust {
		targetBytes = currentBytes + maxAdjust
	} else if diff < -maxAdjust {
		targetBytes = currentBytes - maxAdjust
	}

	if targetBytes < 0 {
		targetBytes = 0
	}

	maxTarget := int64(float64(totalMemory) * 0.8) // 最多使用80%系统内存
	if targetBytes > maxTarget {
		targetBytes = maxTarget
	}

	w.targetSize.Store(targetBytes)
}

func (w *MemoryWorker) GetUsage() float64 {
	if !w.running.Load() {
		return 0
	}

	allocatedSize := w.getCurrentAllocatedSize()

	if w.totalMemory == 0 {
		return 0
	}

	usage := float64(allocatedSize) / float64(w.totalMemory) * 100.0

	if usage > 100 {
		usage = 100
	}

	return usage
}

func (w *MemoryWorker) IsRunning() bool {
	return w.running.Load()
}

func (w *MemoryWorker) GetAllocatedSize() int64 {
	return w.getCurrentAllocatedSize()
}

func (w *MemoryWorker) GetTargetSize() int64 {
	return w.targetSize.Load()
}

func (w *MemoryWorker) SetTotalMemory(total uint64) {
	w.totalMemory = int64(total)
}

func (w *MemoryWorker) GetTotalMemory() int64 {
	return w.totalMemory
}

func (w *MemoryWorker) SetTargetSize(size int64) {
	if size < 0 {
		size = 0
	}

	maxTarget := int64(float64(w.totalMemory) * 0.8)
	if size > maxTarget {
		size = maxTarget
	}

	w.targetSize.Store(size)
}

func (w *MemoryWorker) GetChunkCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.allocatedMem)
}

func (w *MemoryWorker) ClearMemory() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.allocatedMem = make([][]byte, 0)
	w.targetSize.Store(0)
}