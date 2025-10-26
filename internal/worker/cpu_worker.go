package worker

import (
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type CPUWorker struct {
	threshold  int
	running    atomic.Bool
	usage      atomic.Value // float64
	workers    int
	stopChan   chan struct{}
	wg         sync.WaitGroup
	intensity  atomic.Int32 // 工作强度 0-100
	
	// 性能统计
	lastAdjustTime time.Time
	adjustMutex    sync.Mutex
}

func NewCPUWorker(threshold int) *CPUWorker {
	w := &CPUWorker{
		threshold:      threshold,
		workers:        runtime.NumCPU(),
		stopChan:       make(chan struct{}),
		lastAdjustTime: time.Now(),
	}
	w.usage.Store(0.0)
	w.intensity.Store(30) // 初始强度30%
	return w
}

func (w *CPUWorker) Start() {
	if w.running.Load() {
		return
	}

	w.running.Store(true)
	w.stopChan = make(chan struct{})

	// 为每个CPU核心启动一个工作协程
	for i := 0; i < w.workers; i++ {
		w.wg.Add(1)
		go w.work(i)
	}
}

func (w *CPUWorker) Stop() {
	if !w.running.Load() {
		return
	}

	w.running.Store(false)
	close(w.stopChan)
	w.wg.Wait()
	w.usage.Store(0.0)
}

func (w *CPUWorker) work(id int) {
	defer w.wg.Done()

	// 绑定到特定的OS线程
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		select {
		case <-w.stopChan:
			return
		default:
			intensity := w.intensity.Load()
			
			// 根据强度调整工作时间和休息时间
			// 使用微秒级精度
			workDuration := time.Duration(intensity) * time.Microsecond * 100
			sleepDuration := time.Duration(100-intensity) * time.Microsecond * 100

			// 执行计算密集型任务
			start := time.Now()
			for time.Since(start) < workDuration {
				// 1. 数学计算：计算圆周率（使用 Leibniz 公式）
				piResult := 0.0
				for i := 0; i < 1000; i++ {
					piResult += math.Pow(-1, float64(i)) / (2*float64(i) + 1)
				}
				_ = piResult * 4
				
				// 2. 三角函数计算
				for i := 0; i < 100; i++ {
					angle := float64(i) * 0.1
					_ = math.Sin(angle) * math.Cos(angle) * math.Tan(angle)
				}
				
				// 3. 矩阵运算
				matrix := make([][]float64, 10)
				for i := range matrix {
					matrix[i] = make([]float64, 10)
					for j := range matrix[i] {
						matrix[i][j] = float64(i*j) * math.Sin(float64(i+j))
					}
				}
				
				// 4. 对数和指数运算
				for i := 1; i < 100; i++ {
					_ = math.Log(float64(i)) * math.Exp(float64(i%10))
				}
				
				// 5. 平方根和幂运算
				for i := 1; i < 100; i++ {
					_ = math.Sqrt(float64(i)) * math.Pow(float64(i), 1.5)
				}
			}

			// 休息
			if sleepDuration > 0 {
				time.Sleep(sleepDuration)
			}
		}
	}
}

func (w *CPUWorker) AdjustLoad(currentWorkerUsage, targetWorkerUsage float64) {
	if !w.running.Load() {
		return
	}

	w.adjustMutex.Lock()
	defer w.adjustMutex.Unlock()

	if time.Since(w.lastAdjustTime) < 500*time.Millisecond {
		return
	}
	w.lastAdjustTime = time.Now()

	diff := targetWorkerUsage - currentWorkerUsage
	currentIntensity := w.intensity.Load()
	newIntensity := currentIntensity

	// 根据差值调整强度（更精细的控制）
	if diff > 20 {
		newIntensity += 10
	} else if diff > 10 {
		newIntensity += 5
	} else if diff > 5 {
		newIntensity += 3
	} else if diff > 2 {
		newIntensity += 2
	} else if diff > 0.5 {
		newIntensity += 1
	} else if diff < -20 {
		newIntensity -= 10
	} else if diff < -10 {
		newIntensity -= 5
	} else if diff < -5 {
		newIntensity -= 3
	} else if diff < -2 {
		newIntensity -= 2
	} else if diff < -0.5 {
		newIntensity -= 1
	}

	if newIntensity < 0 {
		newIntensity = 0
	} else if newIntensity > 100 {
		newIntensity = 100
	}

	w.intensity.Store(newIntensity)
}

func (w *CPUWorker) GetUsage() float64 {
	if !w.running.Load() {
		return 0
	}
	
	// 根据工作强度估算CPU占用
	// 这是一个粗略估算，实际占用会受系统调度影响
	intensity := float64(w.intensity.Load())
	estimatedUsage := (intensity / 100.0) * float64(w.workers) * 100.0 / float64(runtime.NumCPU())
	
	// 限制在合理范围内
	if estimatedUsage > 100 {
		estimatedUsage = 100
	}
	
	return estimatedUsage
}

func (w *CPUWorker) IsRunning() bool {
	return w.running.Load()
}

// GetIntensity 获取当前工作强度
func (w *CPUWorker) GetIntensity() int32 {
	return w.intensity.Load()
}

// GetWorkerCount 获取工作线程数量
func (w *CPUWorker) GetWorkerCount() int {
	return w.workers
}

// SetIntensity 设置工作强度
func (w *CPUWorker) SetIntensity(intensity int32) {
	if intensity < 0 {
		intensity = 0
	} else if intensity > 100 {
		intensity = 100
	}
	w.intensity.Store(intensity)
}

// ResetIntensity 重置工作强度到默认值
func (w *CPUWorker) ResetIntensity() {
	w.intensity.Store(30)
}