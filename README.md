# cnholiday - 中国节假日查询库

一个简单易用的 Go 语言中国节假日查询库，支持查询法定节假日、调休工作日、周末等信息。

## 特性

- 🎯 **自动获取数据**：自动从 CDN 获取最新的节假日数据
- 💾 **本地文件支持**：支持从本地 JSON 文件加载数据作为 fallback
- ⚡ **高性能缓存**：内置缓存机制，提高查询效率
- 🔒 **并发安全**：使用读写锁保证并发安全
- 📅 **功能丰富**：支持判断节假日、工作日、调休日、补休日等
- 🛠️ **灵活配置**：支持自定义 CDN 地址和本地数据目录

## 安装

```bash
go get github.com/luojiego/cnholiday
```

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "time"
    "github.com/luojiego/cnholiday"
)

func main() {
    // 使用默认检查器
    date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
    
    isHoliday, name, err := cnholiday.IsHoliday(date)
    if err != nil {
        fmt.Printf("查询失败: %v\n", err)
        return
    }
    
    if isHoliday {
        fmt.Printf("%s 是节假日: %s\n", date.Format("2006-01-02"), name)
    } else {
        fmt.Printf("%s 是工作日\n", date.Format("2006-01-02"))
    }
}
```

### 使用自定义配置

```go
package main

import (
    "fmt"
    "time"
    "github.com/luojiego/cnholiday"
)

func main() {
    // 创建自定义配置的检查器
    checker := cnholiday.NewCheckerWithConfig(cnholiday.Config{
        LocalDataDir: "./data", // 本地数据目录
        DisableRemote: false,   // 是否禁用远程获取
    })
    
    date := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
    
    info, err := checker.GetHolidayInfo(date)
    if err != nil {
        fmt.Printf("查询失败: %v\n", err)
        return
    }
    
    fmt.Println(info.String())
}
```

### 预加载数据

```go
package main

import (
    "fmt"
    "github.com/luojiego/cnholiday"
)

func main() {
    checker := cnholiday.NewChecker()
    
    // 预加载多个年份的数据
    years := []int{2024, 2025, 2026}
    for _, year := range years {
        if err := checker.LoadYear(year); err != nil {
            fmt.Printf("加载 %d 年数据失败: %v\n", year, err)
        } else {
            fmt.Printf("成功加载 %d 年数据\n", year)
        }
    }
}
```

## API 文档

### 类型定义

#### Config

配置选项结构体。

```go
type Config struct {
    LocalDataDir  string // 本地数据文件目录路径
    DisableRemote bool   // 禁用远程 CDN 获取
    CDNBaseURL    string // 自定义 CDN 基础 URL
}
```

#### HolidayData

节假日数据结构体。

```go
type HolidayData struct {
    Holidays   map[string]string // 法定节假日
    Workdays   map[string]string // 调休工作日
    InLieuDays map[string]string // 补休日
}
```

#### HolidayInfo

节假日详细信息结构体。

```go
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
```

### Checker 方法

#### NewChecker

创建新的检查器实例。

```go
func NewChecker() *Checker
```

#### NewCheckerWithConfig

使用自定义配置创建检查器实例。

```go
func NewCheckerWithConfig(config Config) *Checker
```

#### LoadYear

加载指定年份的节假日数据。优先从 CDN 获取，失败则尝试从本地文件加载。

```go
func (c *Checker) LoadYear(year int) error
```

**错误处理：**

- 如果远程和本地都加载失败，返回详细的错误信息
- 错误信息包含具体的失败原因

#### LoadYearFromJSON

从 JSON 字节数据加载节假日数据。

```go
func (c *Checker) LoadYearFromJSON(year int, jsonData []byte) error
```

#### IsHoliday

判断指定日期是否是节假日（休息日）。

```go
func (c *Checker) IsHoliday(date time.Time) (isHoliday bool, holidayName string, err error)
```

**返回值：**

- `isHoliday`: 是否是节假日
- `holidayName`: 节假日名称（如果是节假日）
- `err`: 错误信息

#### IsWorkday

判断指定日期是否是工作日。

```go
func (c *Checker) IsWorkday(date time.Time) (bool, error)
```

#### GetHolidayInfo

获取指定日期的详细节假日信息。

```go
func (c *Checker) GetHolidayInfo(date time.Time) (*HolidayInfo, error)
```

#### SetLocalDataDir

设置本地数据目录。

```go
func (c *Checker) SetLocalDataDir(dir string)
```

#### SetDisableRemote

设置是否禁用远程获取。

```go
func (c *Checker) SetDisableRemote(disable bool)
```

#### IsYearLoaded

检查指定年份的数据是否已加载到缓存。

```go
func (c *Checker) IsYearLoaded(year int) bool
```

#### ClearCache

清空所有缓存的数据。

```go
func (c *Checker) ClearCache()
```

#### ClearYear

清除指定年份的缓存数据。

```go
func (c *Checker) ClearYear(year int)
```

### 全局函数

库提供了使用默认检查器的全局函数，方便快速使用：

```go
func IsHoliday(date time.Time) (bool, string, error)
func IsWorkday(date time.Time) (bool, error)
func GetHolidayInfo(date time.Time) (*HolidayInfo, error)
```

## 数据格式

### 本地 JSON 文件格式

本地数据文件应命名为 `{year}.json`，例如 `2026.json`，格式如下：

```json
{
  "holidays": {
    "2026-01-01": "元旦",
    "2026-01-02": "元旦",
    "2026-01-03": "元旦",
    "2026-01-28": "春节",
    "2026-01-29": "春节",
    "2026-01-30": "春节",
    "2026-01-31": "春节",
    "2026-02-01": "春节",
    "2026-02-02": "春节",
    "2026-02-03": "春节"
  },
  "workdays": {
    "2026-01-24": "春节",
    "2026-02-07": "春节"
  },
  "inLieuDays": {
    "2026-01-29": "春节",
    "2026-01-30": "春节",
    "2026-01-31": "春节",
    "2026-02-02": "春节",
    "2026-02-03": "春节"
  }
}
```

**字段说明：**

- `holidays`: 法定节假日和休息日，键为日期（YYYY-MM-DD），值为节日名称
- `workdays`: 调休工作日（周末变工作日），键为日期，值为对应的节日名称
- `inLieuDays`: 补休日（工作日变休息日），键为日期，值为节日名称

## 数据获取策略

库使用以下策略获取节假日数据：

1. **优先远程获取**：首先尝试从配置的 CDN 地址获取数据（默认使用 jsdelivr CDN）
2. **本地 fallback**：如果远程获取失败，尝试从配置的本地目录加载 JSON 文件
3. **错误返回**：如果两种方式都失败，返回详细的错误信息，包含失败原因

### 配置示例

```go
// 仅使用本地文件
checker := cnholiday.NewCheckerWithConfig(cnholiday.Config{
    LocalDataDir: "./data",
    DisableRemote: true,
})

// 使用自定义 CDN
checker := cnholiday.NewCheckerWithConfig(cnholiday.Config{
    CDNBaseURL: "https://your-cdn.com/holidays",
    LocalDataDir: "./data", // fallback
})

// 运行时动态设置
checker := cnholiday.NewChecker()
checker.SetLocalDataDir("./data")
checker.SetDisableRemote(false)
```

## 判断逻辑

判断某日期是否为节假日的逻辑顺序：

1. **调休工作日检查**：如果日期在 `workdays` 中，则为工作日（即使是周末）
2. **法定节假日检查**：如果日期在 `holidays` 中，则为节假日
3. **周末检查**：如果是周六或周日，则为节假日
4. **默认**：其他情况为工作日

## 错误处理

库会返回以下类型的错误：

- **数据加载失败**：无法从远程或本地加载指定年份的数据
- **网络错误**：远程请求失败或返回非 200 状态码
- **文件错误**：本地文件不存在或无法读取
- **解析错误**：JSON 数据格式错误

**示例：**

```go
info, err := checker.GetHolidayInfo(date)
if err != nil {
    // 详细的错误信息，包含失败原因
    log.Printf("查询失败: %v", err)
    return
}
```

## 最佳实践

1. **预加载数据**：在应用启动时预加载常用年份的数据，避免首次查询时的延迟
2. **本地备份**：准备本地 JSON 文件作为备份，防止网络问题导致服务不可用
3. **缓存清理**：如果数据更新，使用 `ClearYear` 或 `ClearCache` 清理缓存
4. **错误处理**：妥善处理可能的错误，避免影响业务逻辑
5. **并发使用**：Checker 是并发安全的，可以在多个 goroutine 中共享使用

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 数据来源

节假日数据通过 npm 包（jsdelivr CDN）获取。默认从 `https://cdn.jsdelivr.net/npm/chinese-days/dist/years` 获取数据。
