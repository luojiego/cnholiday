package cnholiday

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

//go:embed data/*.json
var embeddedData embed.FS

// HolidayData 节假日数据结构
type HolidayData struct {
	Holidays   map[string]string `json:"holidays"`   // 法定节假日
	Workdays   map[string]string `json:"workdays"`   // 调休工作日
	InLieuDays map[string]string `json:"inLieuDays"` // 补休日
}

// Config 配置选项
type Config struct {
	// LocalDataDir 本地数据文件目录路径
	// 本地文件命名格式: {year}.json，例如: 2026.json
	LocalDataDir string
	// DisableRemote 禁用远程 CDN 获取，仅使用本地文件
	DisableRemote bool
	// CDNBaseURL 自定义 CDN 基础 URL
	CDNBaseURL string
}

// Checker 节假日检查器
type Checker struct {
	mu     sync.RWMutex
	cache  map[int]*HolidayData // 按年份缓存
	config Config
}

// NewChecker 创建新的检查器
func NewChecker() *Checker {
	return &Checker{
		cache: make(map[int]*HolidayData),
		config: Config{
			CDNBaseURL: "https://cdn.jsdelivr.net/npm/chinese-days/dist/years",
		},
	}
}

// NewCheckerWithConfig 使用自定义配置创建检查器
func NewCheckerWithConfig(config Config) *Checker {
	if config.CDNBaseURL == "" {
		config.CDNBaseURL = "https://cdn.jsdelivr.net/npm/chinese-days/dist/years"
	}
	return &Checker{
		cache:  make(map[int]*HolidayData),
		config: config,
	}
}

// LoadYear 加载指定年份的节假日数据
// 加载优先级：
// 1. 远程 CDN（如果未禁用）
// 2. 用户配置的本地目录（如果配置了 LocalDataDir）
// 3. 库内置的嵌入数据（如果网络和本地都失败，自动使用）
func (c *Checker) LoadYear(year int) error {
	var lastErr error

	// 1. 尝试从远程 CDN 获取（如果未禁用）
	if !c.config.DisableRemote {
		if err := c.loadYearFromRemote(year); err == nil {
			return nil // 成功从远程加载
		} else {
			lastErr = fmt.Errorf("远程加载失败: %w", err)
		}
	}

	// 2. 尝试从本地文件加载（如果配置了本地目录）
	if c.config.LocalDataDir != "" {
		if err := c.loadYearFromLocal(year); err == nil {
			return nil // 成功从本地加载
		} else {
			if lastErr != nil {
				lastErr = fmt.Errorf("%v; 本地加载失败: %w", lastErr, err)
			} else {
				lastErr = fmt.Errorf("本地加载失败: %w", err)
			}
		}
	}

	// 3. 尝试从嵌入的文件系统加载（库内置数据）
	if err := c.loadYearFromEmbedded(year); err == nil {
		return nil // 成功从嵌入数据加载
	} else {
		if lastErr != nil {
			lastErr = fmt.Errorf("%v; 嵌入数据加载失败: %w", lastErr, err)
		} else {
			lastErr = fmt.Errorf("嵌入数据加载失败: %w", err)
		}
	}

	// 4. 所有方式都失败
	if lastErr != nil {
		return fmt.Errorf("无法加载 %d 年的节假日数据: %w", year, lastErr)
	}

	return fmt.Errorf("无法加载 %d 年的节假日数据: 未配置数据源", year)
}

// loadYearFromRemote 从远程 CDN 加载数据
func (c *Checker) loadYearFromRemote(year int) error {
	url := fmt.Sprintf("%s/%d.json", c.config.CDNBaseURL, year)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 状态码 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var data HolidayData
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	c.mu.Lock()
	c.cache[year] = &data
	c.mu.Unlock()

	return nil
}

// loadYearFromLocal 从本地文件加载数据
func (c *Checker) loadYearFromLocal(year int) error {
	filename := filepath.Join(c.config.LocalDataDir, fmt.Sprintf("%d.json", year))

	data, err := os.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("文件不存在: %s", filename)
		}
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var holidayData HolidayData
	if err := json.Unmarshal(data, &holidayData); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	c.mu.Lock()
	c.cache[year] = &holidayData
	c.mu.Unlock()

	return nil
}

// loadYearFromEmbedded 从嵌入的文件系统加载数据
func (c *Checker) loadYearFromEmbedded(year int) error {
	filename := fmt.Sprintf("data/%d.json", year)

	data, err := embeddedData.ReadFile(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("嵌入文件中不存在: %s", filename)
		}
		return fmt.Errorf("读取嵌入文件失败: %w", err)
	}

	var holidayData HolidayData
	if err := json.Unmarshal(data, &holidayData); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	c.mu.Lock()
	c.cache[year] = &holidayData
	c.mu.Unlock()

	return nil
}

// LoadYearFromJSON 从JSON字节数据加载节假日数据
func (c *Checker) LoadYearFromJSON(year int, jsonData []byte) error {
	var data HolidayData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to parse holiday data: %w", err)
	}

	c.mu.Lock()
	c.cache[year] = &data
	c.mu.Unlock()

	return nil
}

// ensureYearLoaded 确保年份数据已加载
func (c *Checker) ensureYearLoaded(year int) error {
	c.mu.RLock()
	_, exists := c.cache[year]
	c.mu.RUnlock()

	if !exists {
		if err := c.LoadYear(year); err != nil {
			return fmt.Errorf("加载 %d 年数据失败: %w", year, err)
		}
	}
	return nil
}

// SetLocalDataDir 设置本地数据目录
func (c *Checker) SetLocalDataDir(dir string) {
	c.mu.Lock()
	c.config.LocalDataDir = dir
	c.mu.Unlock()
}

// SetDisableRemote 设置是否禁用远程获取
func (c *Checker) SetDisableRemote(disable bool) {
	c.mu.Lock()
	c.config.DisableRemote = disable
	c.mu.Unlock()
}

// IsYearLoaded 检查指定年份的数据是否已加载
func (c *Checker) IsYearLoaded(year int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.cache[year]
	return exists
}

// ClearCache 清空缓存
func (c *Checker) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[int]*HolidayData)
	c.mu.Unlock()
}

// ClearYear 清除指定年份的缓存
func (c *Checker) ClearYear(year int) {
	c.mu.Lock()
	delete(c.cache, year)
	c.mu.Unlock()
}

// IsHoliday 判断指定日期是否是节假日(休息日)
// 返回: isHoliday, holidayName, error
func (c *Checker) IsHoliday(date time.Time) (bool, string, error) {
	year := date.Year()
	if err := c.ensureYearLoaded(year); err != nil {
		return false, "", err
	}

	dateStr := date.Format("2006-01-02")

	c.mu.RLock()
	data := c.cache[year]
	c.mu.RUnlock()

	// 1. 检查是否在调休工作日列表中(周末变工作日)
	if name, exists := data.Workdays[dateStr]; exists {
		return false, name, nil // 是调休工作日,不是假日
	}

	// 2. 检查是否在法定节假日列表中
	if name, exists := data.Holidays[dateStr]; exists {
		return true, name, nil // 是法定节假日
	}

	// 3. 检查是否是周末
	weekday := date.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return true, "周末", nil
	}

	// 4. 工作日
	return false, "", nil
}

// IsWorkday 判断指定日期是否是工作日
func (c *Checker) IsWorkday(date time.Time) (bool, error) {
	isHoliday, _, err := c.IsHoliday(date)
	if err != nil {
		return false, err
	}
	return !isHoliday, nil
}

// GetHolidayInfo 获取节假日详细信息
func (c *Checker) GetHolidayInfo(date time.Time) (*HolidayInfo, error) {
	year := date.Year()
	if err := c.ensureYearLoaded(year); err != nil {
		return nil, err
	}

	dateStr := date.Format("2006-01-02")
	weekday := date.Weekday()

	c.mu.RLock()
	data := c.cache[year]
	c.mu.RUnlock()

	info := &HolidayInfo{
		Date:    date,
		Weekday: weekday,
	}

	// 检查调休工作日
	if name, exists := data.Workdays[dateStr]; exists {
		info.IsWorkday = true
		info.IsAdjustedWorkday = true
		info.HolidayName = name
		return info, nil
	}

	// 检查法定节假日
	if name, exists := data.Holidays[dateStr]; exists {
		info.IsHoliday = true
		info.HolidayName = name

		// 检查是否是补休日
		if _, isInLieu := data.InLieuDays[dateStr]; isInLieu {
			info.IsInLieuDay = true
		}
		return info, nil
	}

	// 检查周末
	if weekday == time.Saturday || weekday == time.Sunday {
		info.IsHoliday = true
		info.IsWeekend = true
		return info, nil
	}

	// 普通工作日
	info.IsWorkday = true
	return info, nil
}

// HolidayInfo 节假日详细信息
type HolidayInfo struct {
	Date              time.Time
	Weekday           time.Weekday
	IsWorkday         bool   // 是否是工作日
	IsHoliday         bool   // 是否是节假日
	IsWeekend         bool   // 是否是周末
	IsAdjustedWorkday bool   // 是否是调休工作日
	IsInLieuDay       bool   // 是否是补休日
	HolidayName       string // 节假日名称
}

// String 格式化输出节假日信息
func (h *HolidayInfo) String() string {
	if h.IsAdjustedWorkday {
		return fmt.Sprintf("%s (调休工作日 - %s)", h.Date.Format("2006-01-02"), h.HolidayName)
	}
	if h.IsInLieuDay {
		return fmt.Sprintf("%s (补休 - %s)", h.Date.Format("2006-01-02"), h.HolidayName)
	}
	if h.IsHoliday {
		if h.IsWeekend {
			return fmt.Sprintf("%s (周末)", h.Date.Format("2006-01-02"))
		}
		return fmt.Sprintf("%s (节假日 - %s)", h.Date.Format("2006-01-02"), h.HolidayName)
	}
	return fmt.Sprintf("%s (工作日)", h.Date.Format("2006-01-02"))
}

// 全局默认检查器
var defaultChecker = NewChecker()

// IsHoliday 使用默认检查器判断是否是节假日
func IsHoliday(date time.Time) (bool, string, error) {
	return defaultChecker.IsHoliday(date)
}

// IsWorkday 使用默认检查器判断是否是工作日
func IsWorkday(date time.Time) (bool, error) {
	return defaultChecker.IsWorkday(date)
}

// GetHolidayInfo 使用默认检查器获取节假日信息
func GetHolidayInfo(date time.Time) (*HolidayInfo, error) {
	return defaultChecker.GetHolidayInfo(date)
}
