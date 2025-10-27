package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	GitHubAPI     = "https://api.github.com/repos/MakotoArai-CN/MikaBooM/releases/latest"
	GitHubRepo    = "https://github.com/MakotoArai-CN/MikaBooM/releases"
	UpdateTimeout = 300 * time.Second
)

type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
	Body    string  `json:"body"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type Updater struct {
	CurrentVersion string
	OS             string
	Arch           string
}

func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		CurrentVersion: currentVersion,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
	}
}

// CheckUpdate 检查是否有新版本
func (u *Updater) CheckUpdate() (*Release, bool, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", GitHubAPI, nil)
	if err != nil {
		return nil, false, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "MikaBooM-Updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("请求GitHub API失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("GitHub API返回错误: %d %s", resp.StatusCode, resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, fmt.Errorf("解析GitHub响应失败: %w", err)
	}

	// 比较版本
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(u.CurrentVersion, "v")

	if latestVersion == currentVersion {
		return &release, false, nil
	}

	// 简单的版本比较
	if compareVersions(latestVersion, currentVersion) > 0 {
		return &release, true, nil
	}

	return &release, false, nil
}

// CheckUpdateSilent 静默检查更新（用于启动时检查）
func (u *Updater) CheckUpdateSilent() (*Release, bool, error) {
	client := &http.Client{
		Timeout: 10 * time.Second, // 启动检查使用较短超时
	}

	req, err := http.NewRequest("GET", GitHubAPI, nil)
	if err != nil {
		return nil, false, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "MikaBooM-Updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("status: %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, false, err
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(u.CurrentVersion, "v")

	if latestVersion == currentVersion {
		return &release, false, nil
	}

	if compareVersions(latestVersion, currentVersion) > 0 {
		return &release, true, nil
	}

	return &release, false, nil
}

// PerformUpdate 执行更新
func (u *Updater) PerformUpdate(release *Release) error {
	// 找到匹配的资源
	asset := u.findMatchingAsset(release.Assets)
	if asset == nil {
		return fmt.Errorf("未找到适配 %s/%s 的发布文件", u.OS, u.Arch)
	}

	color.Green("✓ 找到更新文件: %s (%.2f MB)", asset.Name, float64(asset.Size)/1024/1024)
	fmt.Println()

	// 下载文件
	color.Cyan("📥 正在下载更新...")
	tempFile, err := u.downloadAsset(asset)
	if err != nil {
		return fmt.Errorf("下载更新失败: %w", err)
	}
	defer os.Remove(tempFile)

	color.Green("✓ 下载完成")
	fmt.Println()

	// 解压并替换
	color.Cyan("📦 正在安装更新...")
	if err := u.extractAndReplace(tempFile); err != nil {
		return fmt.Errorf("安装更新失败: %w", err)
	}

	color.Green("✓ 更新安装完成")
	fmt.Println()

	return nil
}

// findMatchingAsset 查找匹配当前系统的资源
func (u *Updater) findMatchingAsset(assets []Asset) *Asset {
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		osName := strings.ToLower(u.OS)
		archName := strings.ToLower(u.Arch)

		// 检查文件名是否包含操作系统和架构
		if strings.Contains(name, osName) && strings.Contains(name, archName) && strings.HasSuffix(name, ".tar.gz") {
			return &asset
		}
	}

	return nil
}

// downloadAsset 下载资源文件
func (u *Updater) downloadAsset(asset *Asset) (string, error) {
	client := &http.Client{
		Timeout: UpdateTimeout,
	}

	resp, err := client.Get(asset.BrowserDownloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: %d %s", resp.StatusCode, resp.Status)
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "mikaboom-update-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// 显示下载进度
	size := asset.Size
	downloaded := int64(0)
	lastPercent := 0

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := tempFile.Write(buf[:n])
			if writeErr != nil {
				return "", writeErr
			}
			downloaded += int64(n)

			// 更新进度
			percent := int(float64(downloaded) / float64(size) * 100)
			if percent != lastPercent && percent%10 == 0 {
				color.Cyan("  进度: %d%% (%.2f MB / %.2f MB)",
					percent,
					float64(downloaded)/1024/1024,
					float64(size)/1024/1024)
				lastPercent = percent
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	return tempFile.Name(), nil
}

// extractAndReplace 解压并替换可执行文件
func (u *Updater) extractAndReplace(tarGzPath string) error {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		exePath, _ = os.Executable()
	}

	// 打开 tar.gz 文件
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// 查找可执行文件
	var found bool
	binaryName := "MikaBooM"
	if u.OS == "windows" {
		binaryName = "MikaBooM.exe"
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "mikaboom-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 查找二进制文件
		if strings.Contains(header.Name, binaryName) {
			found = true

			// 提取到临时文件
			tempBinary := filepath.Join(tempDir, "mikaboom-new")
			if u.OS == "windows" {
				tempBinary += ".exe"
			}

			outFile, err := os.OpenFile(tempBinary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			// 备份当前文件
			backupPath := exePath + ".backup"
			if err := os.Rename(exePath, backupPath); err != nil {
				return fmt.Errorf("备份当前文件失败: %w", err)
			}

			// 替换文件
			if err := os.Rename(tempBinary, exePath); err != nil {
				// 恢复备份
				os.Rename(backupPath, exePath)
				return fmt.Errorf("替换文件失败: %w", err)
			}

			// 设置执行权限（Unix系统）
			if u.OS != "windows" {
				os.Chmod(exePath, 0755)
			}

			// 删除备份
			os.Remove(backupPath)

			break
		}
	}

	if !found {
		return fmt.Errorf("压缩包中未找到可执行文件")
	}

	return nil
}

// Restart 重启程序
func (u *Updater) Restart() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	args := os.Args[1:]
	// 移除 -update 参数
	newArgs := make([]string, 0)
	for _, arg := range args {
		if arg != "-update" {
			newArgs = append(newArgs, arg)
		}
	}

	color.Cyan("🔄 正在重启程序...")
	time.Sleep(1 * time.Second)

	// 启动新进程
	attr := &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}

	_, err = os.StartProcess(exePath, append([]string{exePath}, newArgs...), attr)
	return err
}

// ShowUpdateInfo 显示更新信息
func ShowUpdateInfo(release *Release, currentVersion string) {
	color.Yellow("╔═══════════════════════════════════════════════╗")
	color.Yellow("║           📦 发现新版本                        ║")
	color.Yellow("╚═══════════════════════════════════════════════╝")
	fmt.Println()
	color.Cyan("当前版本: v%s", currentVersion)
	color.Green("最新版本: %s", release.TagName)
	fmt.Println()
	if release.Body != "" {
		color.Yellow("更新说明:")
		// 简化显示，只显示前10行
		lines := strings.Split(release.Body, "\n")
		for i, line := range lines {
			if i >= 10 {
				color.HiBlack("  ... (更多内容请访问 GitHub)")
				break
			}
			if line != "" {
				fmt.Println("  " + line)
			}
		}
		fmt.Println()
	}
}

// ShowUpdateNotice 显示启动时的更新提示
func ShowUpdateNotice(release *Release, currentVersion string) {
	color.Yellow("┌───────────────────────────────────────────────┐")
	color.Yellow("│  📦 发现新版本: %s", release.TagName)
	color.Yellow("│  当前版本: v%s", currentVersion)
	color.Yellow("│  运行 'MikaBooM -update' 进行更新", )
	color.Yellow("└───────────────────────────────────────────────┘")
	fmt.Println()
}

// compareVersions 比较版本号
// 返回: 1 表示 v1 > v2, -1 表示 v1 < v2, 0 表示相等
func compareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int

		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &n2)
		}

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	return 0
}