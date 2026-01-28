package cnholiday

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewChecker(t *testing.T) {
	checker := NewChecker()
	if checker == nil {
		t.Fatal("NewChecker() returned nil")
	}
	if checker.cache == nil {
		t.Error("cache not initialized")
	}
	if checker.config.CDNBaseURL == "" {
		t.Error("CDNBaseURL not set to default")
	}
}

func TestNewCheckerWithConfig(t *testing.T) {
	config := Config{
		LocalDataDir:  "./testdata",
		DisableRemote: true,
		CDNBaseURL:    "https://custom.cdn.com",
	}
	checker := NewCheckerWithConfig(config)
	if checker.config.LocalDataDir != "./testdata" {
		t.Error("LocalDataDir not set correctly")
	}
	if !checker.config.DisableRemote {
		t.Error("DisableRemote not set correctly")
	}
	if checker.config.CDNBaseURL != "https://custom.cdn.com" {
		t.Error("CDNBaseURL not set correctly")
	}
}

func TestLoadYearFromJSON(t *testing.T) {
	checker := NewChecker()

	jsonData := []byte(`{
		"holidays": {
			"2026-01-01": "元旦",
			"2026-10-01": "国庆节"
		},
		"workdays": {
			"2026-01-04": "元旦"
		},
		"inLieuDays": {}
	}`)

	err := checker.LoadYearFromJSON(2026, jsonData)
	if err != nil {
		t.Fatalf("LoadYearFromJSON failed: %v", err)
	}

	if !checker.IsYearLoaded(2026) {
		t.Error("Year 2026 should be loaded")
	}

	// 测试无效的 JSON
	invalidJSON := []byte(`{invalid json}`)
	err = checker.LoadYearFromJSON(2027, invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestIsHoliday(t *testing.T) {
	checker := NewChecker()

	// 加载测试数据
	jsonData := []byte(`{
		"holidays": {
			"2026-01-01": "元旦",
			"2026-10-01": "国庆节"
		},
		"workdays": {
			"2026-01-04": "元旦"
		},
		"inLieuDays": {}
	}`)

	err := checker.LoadYearFromJSON(2026, jsonData)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		date     string
		expected bool
		name     string
	}{
		{"2026-01-01", true, "元旦"},  // 法定节假日
		{"2026-10-01", true, "国庆节"}, // 法定节假日
		{"2026-01-04", false, "元旦"}, // 调休工作日（周日变工作日）
		{"2026-01-03", true, "周末"},  // 周六
		{"2026-01-05", false, ""},   // 普通工作日（周一）
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			date, _ := time.Parse("2006-01-02", tt.date)
			isHoliday, name, err := checker.IsHoliday(date)
			if err != nil {
				t.Fatalf("IsHoliday failed: %v", err)
			}
			if isHoliday != tt.expected {
				t.Errorf("IsHoliday(%s) = %v, want %v", tt.date, isHoliday, tt.expected)
			}
			if name != tt.name {
				t.Errorf("Holiday name = %s, want %s", name, tt.name)
			}
		})
	}
}

func TestIsWorkday(t *testing.T) {
	checker := NewChecker()

	jsonData := []byte(`{
		"holidays": {
			"2026-01-01": "元旦"
		},
		"workdays": {
			"2026-01-04": "元旦"
		},
		"inLieuDays": {}
	}`)

	err := checker.LoadYearFromJSON(2026, jsonData)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		date     string
		expected bool
	}{
		{"2026-01-01", false}, // 元旦，节假日
		{"2026-01-04", true},  // 调休工作日
		{"2026-01-05", true},  // 普通工作日
		{"2026-01-03", false}, // 周六
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			date, _ := time.Parse("2006-01-02", tt.date)
			isWorkday, err := checker.IsWorkday(date)
			if err != nil {
				t.Fatalf("IsWorkday failed: %v", err)
			}
			if isWorkday != tt.expected {
				t.Errorf("IsWorkday(%s) = %v, want %v", tt.date, isWorkday, tt.expected)
			}
		})
	}
}

func TestGetHolidayInfo(t *testing.T) {
	checker := NewChecker()

	jsonData := []byte(`{
		"holidays": {
			"2026-01-01": "元旦",
			"2026-01-02": "元旦"
		},
		"workdays": {
			"2026-01-04": "元旦"
		},
		"inLieuDays": {
			"2026-01-02": "元旦"
		}
	}`)

	err := checker.LoadYearFromJSON(2026, jsonData)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 测试法定节假日
	date1, _ := time.Parse("2006-01-02", "2026-01-01")
	info1, err := checker.GetHolidayInfo(date1)
	if err != nil {
		t.Fatalf("GetHolidayInfo failed: %v", err)
	}
	if !info1.IsHoliday || info1.HolidayName != "元旦" {
		t.Error("Expected holiday info for 2026-01-01")
	}

	// 测试补休日
	date2, _ := time.Parse("2006-01-02", "2026-01-02")
	info2, err := checker.GetHolidayInfo(date2)
	if err != nil {
		t.Fatalf("GetHolidayInfo failed: %v", err)
	}
	if !info2.IsInLieuDay {
		t.Error("Expected IsInLieuDay for 2026-01-02")
	}

	// 测试调休工作日
	date3, _ := time.Parse("2006-01-02", "2026-01-04")
	info3, err := checker.GetHolidayInfo(date3)
	if err != nil {
		t.Fatalf("GetHolidayInfo failed: %v", err)
	}
	if !info3.IsAdjustedWorkday || !info3.IsWorkday {
		t.Error("Expected adjusted workday for 2026-01-04")
	}

	// 测试周末
	date4, _ := time.Parse("2006-01-02", "2026-01-03")
	info4, err := checker.GetHolidayInfo(date4)
	if err != nil {
		t.Fatalf("GetHolidayInfo failed: %v", err)
	}
	if !info4.IsWeekend || !info4.IsHoliday {
		t.Error("Expected weekend for 2026-01-03")
	}
}

func TestLoadYearFromLocal(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 创建测试数据文件
	testData := []byte(`{
		"holidays": {
			"2026-01-01": "元旦"
		},
		"workdays": {},
		"inLieuDays": {}
	}`)

	testFile := filepath.Join(tmpDir, "2026.json")
	err := os.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 测试从本地加载
	checker := NewCheckerWithConfig(Config{
		LocalDataDir:  tmpDir,
		DisableRemote: true,
	})

	err = checker.LoadYear(2026)
	if err != nil {
		t.Fatalf("LoadYear from local failed: %v", err)
	}

	if !checker.IsYearLoaded(2026) {
		t.Error("Year 2026 should be loaded from local file")
	}
}

func TestLoadYearError(t *testing.T) {
	// 测试既没有远程也没有本地数据的情况
	checker := NewCheckerWithConfig(Config{
		LocalDataDir:  "/nonexistent",
		DisableRemote: true,
	})

	err := checker.LoadYear(2026)
	if err == nil {
		t.Error("Expected error when loading non-existent year")
	}
}

func TestCacheOperations(t *testing.T) {
	checker := NewChecker()

	jsonData := []byte(`{
		"holidays": {"2026-01-01": "元旦"},
		"workdays": {},
		"inLieuDays": {}
	}`)

	// 加载数据
	err := checker.LoadYearFromJSON(2026, jsonData)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if !checker.IsYearLoaded(2026) {
		t.Error("Year should be loaded")
	}

	// 清除指定年份
	checker.ClearYear(2026)
	if checker.IsYearLoaded(2026) {
		t.Error("Year should be cleared")
	}

	// 重新加载
	err = checker.LoadYearFromJSON(2026, jsonData)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	// 清空所有缓存
	checker.ClearCache()
	if checker.IsYearLoaded(2026) {
		t.Error("Cache should be cleared")
	}
}

func TestSetters(t *testing.T) {
	checker := NewChecker()

	// 测试 SetLocalDataDir
	checker.SetLocalDataDir("./testdata")
	if checker.config.LocalDataDir != "./testdata" {
		t.Error("SetLocalDataDir failed")
	}

	// 测试 SetDisableRemote
	checker.SetDisableRemote(true)
	if !checker.config.DisableRemote {
		t.Error("SetDisableRemote failed")
	}
}

func TestGlobalFunctions(t *testing.T) {
	// 测试全局函数
	// 注意：这些测试可能需要网络连接或预加载数据

	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)

	// 尝试使用全局函数（可能会失败如果没有网络）
	_, _, err := IsHoliday(date)
	// 我们不检查错误，因为可能没有网络
	_ = err

	_, err = IsWorkday(date)
	_ = err

	_, err = GetHolidayInfo(date)
	_ = err
}

func TestHolidayInfoString(t *testing.T) {
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)

	tests := []struct {
		info     HolidayInfo
		expected string
	}{
		{
			HolidayInfo{Date: date, IsWorkday: true},
			"2026-01-01 (工作日)",
		},
		{
			HolidayInfo{Date: date, IsHoliday: true, IsWeekend: true},
			"2026-01-01 (周末)",
		},
		{
			HolidayInfo{Date: date, IsHoliday: true, HolidayName: "元旦"},
			"2026-01-01 (节假日 - 元旦)",
		},
		{
			HolidayInfo{Date: date, IsWorkday: true, IsAdjustedWorkday: true, HolidayName: "春节"},
			"2026-01-01 (调休工作日 - 春节)",
		},
		{
			HolidayInfo{Date: date, IsHoliday: true, IsInLieuDay: true, HolidayName: "春节"},
			"2026-01-01 (补休 - 春节)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.info.String()
			if result != tt.expected {
				t.Errorf("String() = %s, want %s", result, tt.expected)
			}
		})
	}
}
