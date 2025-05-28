package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/honeybbq/protoc-gen-go-zero-errors/errors"
)

const (
	// ANSI颜色代码
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

type ErrorInfo struct {
	Package     string `json:"package"`
	Function    string `json:"function"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Timestamp   int64  `json:"timestamp"`
	GoroutineID uint64 `json:"goroutine_id"`
	ProcessID   int    `json:"process_id"`
	Random      string `json:"random"`
	HumanTime   string `json:"human_time"`
	Raw         string `json:"raw"`
}

var (
	flagJSON    = flag.Bool("json", false, "输出JSON格式")
	flagNoColor = flag.Bool("no-color", false, "禁用颜色输出")
	flagHelp    = flag.Bool("h", false, "显示帮助信息")
	flagVersion = flag.Bool("version", false, "显示版本信息")
	flagBatch   = flag.Bool("batch", false, "批量模式，从stdin读取多个错误ID")
	flagVerbose = flag.Bool("v", false, "详细输出模式")
)

const version = "v1.0.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `%s错误ID解析工具%s %s%s%s

%s用法:%s
  %s./error-decoder [选项] <错误ID>%s
  %secho "错误ID" | ./error-decoder -batch%s

%s选项:%s
  %s-json%s        输出JSON格式
  %s-no-color%s    禁用颜色输出  
  %s-batch%s       批量模式，从stdin读取
  %s-v%s           详细输出模式
  %s-h%s           显示此帮助信息
  %s-version%s     显示版本信息

%s示例:%s
  %s# 解析单个错误ID%s
  %s./error-decoder "YXBpL3VzZXIvdjEuR2V0VXNlckB1c2VyX2xvZ2ljLmdvOjI1OjE2NDA5OTUyMDA="%s

  %s# JSON格式输出%s
  %s./error-decoder -json "错误ID"%s

  %s# 批量解析%s
  %secho -e "ID1\nID2\nID3" | ./error-decoder -batch%s

`,
			ColorBold+ColorCyan, ColorReset, ColorYellow, version, ColorReset,
			ColorBold, ColorReset,
			ColorGreen, ColorReset,
			ColorGreen, ColorReset,
			ColorBold, ColorReset,
			ColorYellow, ColorReset,
			ColorYellow, ColorReset,
			ColorYellow, ColorReset,
			ColorYellow, ColorReset,
			ColorYellow, ColorReset,
			ColorYellow, ColorReset,
			ColorBold, ColorReset,
			ColorCyan, ColorReset,
			ColorGreen, ColorReset,
			ColorCyan, ColorReset,
			ColorGreen, ColorReset,
			ColorCyan, ColorReset,
			ColorGreen, ColorReset,
		)
	}

	flag.Parse()

	if *flagHelp {
		flag.Usage()
		return
	}

	if *flagVersion {
		fmt.Printf("error-decoder %s\n", version)
		return
	}

	if *flagBatch {
		processBatch()
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "%s错误: 请提供错误ID%s\n", ColorRed, ColorReset)
		fmt.Fprintf(os.Stderr, "使用 -h 查看帮助信息\n")
		os.Exit(1)
	}

	errorID := args[0]
	processErrorID(errorID)
}

func processBatch() {
	fmt.Printf("%s🔍 批量解析模式 - 等待输入错误ID (每行一个，Ctrl+D结束)%s\n", ColorCyan, ColorReset)

	var line string
	count := 0
	for {
		n, err := fmt.Scanln(&line)
		if err != nil || n == 0 {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		count++
		fmt.Printf("\n%s=== 错误ID #%d ===%s\n", ColorYellow, count, ColorReset)
		processErrorID(line)
	}

	if count > 0 {
		fmt.Printf("\n%s✅ 总共处理了 %d 个错误ID%s\n", ColorGreen, count, ColorReset)
	} else {
		fmt.Printf("%s⚠️  没有收到任何错误ID%s\n", ColorYellow, ColorReset)
	}
}

func processErrorID(errorID string) {
	errorID = strings.TrimSpace(errorID)
	if errorID == "" {
		fmt.Printf("%s错误: 错误ID为空%s\n", ColorRed, ColorReset)
		return
	}

	info, err := parseErrorID(errorID)
	if err != nil {
		fmt.Printf("%s解析错误: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	if *flagJSON {
		outputJSON(info)
	} else {
		outputFormatted(info)
	}
}

func parseErrorID(errorID string) (*ErrorInfo, error) {
	// 使用我们的errors包解码
	debugInfo, err := errors.DecodeErrorID(errorID)
	if err != nil {
		return nil, fmt.Errorf("无法解码错误ID: %w", err)
	}

	raw := debugInfo["raw"].(string)

	// 解析格式: pkg.func@file:line:timestamp:gid:pid:random
	parts := strings.Split(raw, ":")
	if len(parts) < 6 {
		return nil, fmt.Errorf("错误ID格式不正确，期望至少6个部分，实际: %d", len(parts))
	}

	// 解析包名和函数名
	funcPart := parts[0] // pkg.func@file
	atIndex := strings.LastIndex(funcPart, "@")
	if atIndex == -1 {
		return nil, fmt.Errorf("无法找到@分隔符")
	}

	pkgFunc := funcPart[:atIndex]
	file := funcPart[atIndex+1:]

	// 分离包名和函数名
	lastDotIndex := strings.LastIndex(pkgFunc, ".")
	var pkg, function string
	if lastDotIndex != -1 {
		pkg = pkgFunc[:lastDotIndex]
		function = pkgFunc[lastDotIndex+1:]
	} else {
		pkg = "main"
		function = pkgFunc
	}

	// 解析数值
	line, _ := strconv.Atoi(parts[1])
	timestamp, _ := strconv.ParseInt(parts[2], 10, 64)
	goroutineID, _ := strconv.ParseUint(parts[3], 10, 64)
	processID, _ := strconv.Atoi(parts[4])
	random := parts[5]

	// 转换时间戳为人类可读格式
	humanTime := time.Unix(0, timestamp).Format("2006-01-02 15:04:05.000000000")

	return &ErrorInfo{
		Package:     pkg,
		Function:    function,
		File:        file,
		Line:        line,
		Timestamp:   timestamp,
		GoroutineID: goroutineID,
		ProcessID:   processID,
		Random:      random,
		HumanTime:   humanTime,
		Raw:         raw,
	}, nil
}

func outputJSON(info *ErrorInfo) {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Printf("%s生成JSON失败: %v%s\n", ColorRed, err, ColorReset)
		return
	}
	fmt.Println(string(data))
}

func outputFormatted(info *ErrorInfo) {
	// 选择颜色函数
	color := func(c, text string) string {
		if *flagNoColor {
			return text
		}
		return c + text + ColorReset
	}

	fmt.Printf("%s\n", color(ColorBold+ColorCyan, "🔍 错误ID解析结果"))
	fmt.Printf("%s\n", strings.Repeat("=", 50))

	fmt.Printf("%s %s\n",
		color(ColorBold, "📦 包名:"),
		color(ColorGreen, info.Package))

	fmt.Printf("%s %s\n",
		color(ColorBold, "🔧 函数:"),
		color(ColorYellow, info.Function))

	fmt.Printf("%s %s:%s\n",
		color(ColorBold, "📄 位置:"),
		color(ColorCyan, info.File),
		color(ColorRed, strconv.Itoa(info.Line)))

	fmt.Printf("%s %s\n",
		color(ColorBold, "⏰ 时间:"),
		color(ColorPurple, info.HumanTime))

	fmt.Printf("%s %s\n",
		color(ColorBold, "🧵 协程ID:"),
		color(ColorBlue, strconv.FormatUint(info.GoroutineID, 10)))

	fmt.Printf("%s %s\n",
		color(ColorBold, "🆔 进程ID:"),
		color(ColorBlue, strconv.Itoa(info.ProcessID)))

	fmt.Printf("%s %s\n",
		color(ColorBold, "🎲 随机值:"),
		color(ColorWhite, info.Random))

	if *flagVerbose {
		fmt.Printf("\n%s\n", color(ColorBold, "📋 详细信息:"))
		fmt.Printf("%s %d\n",
			color(ColorBold, "  • 纳秒时间戳:"),
			info.Timestamp)
		fmt.Printf("%s %s\n",
			color(ColorBold, "  • 原始数据:"),
			color(ColorWhite, info.Raw))
	}

	fmt.Printf("\n%s\n",
		color(ColorGreen+ColorBold, "✅ 解析完成!"))
}
