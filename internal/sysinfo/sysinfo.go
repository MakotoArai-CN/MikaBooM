package sysinfo

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemInfo struct {
	OS          string
	Platform    string
	Hostname    string
	CPUModel    string
	CPUCores    int
	TotalMemory uint64
}

func GetSystemInfo() *SystemInfo {
	info := &SystemInfo{}

	// 操作系统信息
	info.OS = runtime.GOOS
	
	// 主机信息
	hostInfo, err := host.Info()
	if err == nil {
		info.Platform = hostInfo.Platform + " " + hostInfo.PlatformVersion
		info.Hostname = hostInfo.Hostname
		info.OS = hostInfo.OS + " " + hostInfo.PlatformVersion
	}

	// CPU信息
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info.CPUModel = cpuInfo[0].ModelName
	}
	
	cores, err := cpu.Counts(true)
	if err == nil {
		info.CPUCores = cores
	}

	// 内存信息
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		info.TotalMemory = memInfo.Total
	}

	return info
}

func GetCPUModelName() string {
	cpuInfo, err := cpu.Info()
	if err != nil || len(cpuInfo) == 0 {
		return "Unknown"
	}
	return cpuInfo[0].ModelName
}

func GetTotalMemoryGB() float64 {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}
	return float64(memInfo.Total) / 1024 / 1024 / 1024
}
