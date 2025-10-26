package worker

import (
	"sync"
	"sync/atomic"
	"time"
)

type MemoryWorker struct {
	threshold    int
	running      atomic.Bool
	usage        atomic.Value // float64
	allocatedMem [][]byte
	mu           sync.Mutex
	stopChan     chan struct{}
	targetSize   atomic.Int64 // 目标分配大小(字节)
	totalMemory  int64        // 系统总内存
	
	// 性能统计
	lastAdjustTime time.Time
	adjustMutex    sync.Mutex
}

func NewMemoryWorker(threshold int) *MemoryWorker {
	w := &MemoryWorker{
		threshold:      threshold,
		allocatedMem:   make([][]byte, 0),
		stopChan:       make(chan struct{}),
		totalMemory:    int64(16 * 1024 * 1024 * 1024), // 默认16GB
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
	
	// 等待工作协程结束
	time.Sleep(100 * time.Millisecond)
	
	// 释放所有分配的内存
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
				// 需要分配更多内存
				w.allocateMemory(diff)
			} else if diff < 0 {
				// 需要释放一些内存
				w.freeMemory(-diff)
			}
		}
	}
}

func (w *MemoryWorker) allocateMemory(size int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 分块分配，每块10MB
	chunkSize := int64(10 * 1024 * 1024)
	chunks := size / chunkSize
	
	for i := int64(0); i < chunks; i++ {
		chunk := make([]byte, chunkSize)
		// 写入数据确保真实分配
		for j := 0; j < len(chunk); j += 4096 {
			chunk[j] = byte(j % 256)
		}
		w.allocatedMem = append(w.allocatedMem, chunk)
	}

	// 分配剩余部分
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

	// 从后向前释放
	for i := len(w.allocatedMem) - 1; i >= 0; i-- {
		if freed >= size {
			// 保留剩余的块
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
		totalMemory = int64(16 * 1024 * 1024 * 1024)
	}

	// 计算目标分配大小
	targetBytes := int64(float64(totalMemory) * targetWorkerUsage / 100.0)
	
	// 限制最大调整量（每次最多调整512MB）
	maxAdjust := int64(512 * 1024 * 1024)
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

	maxTarget := int64(float64(totalMemory) * 0.8)
	if targetBytes > maxTarget {
		targetBytes = maxTarget
	}

	w.targetSize.Store(targetBytes)
}

func (w *MemoryWorker) GetUsage() float64 {
	if !w.running.Load() {
		return 0
	}
	
	// 根据已分配内存计算占用率
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

// GetAllocatedSize 获取已分配内存大小（字节）
func (w *MemoryWorker) GetAllocatedSize() int64 {
	return w.getCurrentAllocatedSize()
}

// GetTargetSize 获取目标分配大小（字节）
func (w *MemoryWorker) GetTargetSize() int64 {
	return w.targetSize.Load()
}

// SetTotalMemory 设置系统总内存
func (w *MemoryWorker) SetTotalMemory(total uint64) {
	w.totalMemory = int64(total)
}

// GetTotalMemory 获取系统总内存
func (w *MemoryWorker) GetTotalMemory() int64 {
	return w.totalMemory
}

// SetTargetSize 设置目标分配大小
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

// GetChunkCount 获取内存块数量
func (w *MemoryWorker) GetChunkCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.allocatedMem)
}

// ClearMemory 清空所有已分配内存
func (w *MemoryWorker) ClearMemory() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.allocatedMem = make([][]byte, 0)
	w.targetSize.Store(0)
}