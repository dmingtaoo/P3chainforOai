package main

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
)

func JudgeOperatingSystem() string {
	switch os := runtime.GOOS; os {
	case "darwin":
		return "macOS"
	case "linux":
		return "linux"
	case "windows":
		return "windows"
	default:
		return "unknown"
	}
}

func main() {
	pidCommand := ""
	switch JudgeOperatingSystem() {
	case "macOS":
		pidCommand += " -pid "
	case "linux":
		pidCommand += " -p "
	}

	root := ".." + string(os.PathSeparator) + "auto" // 指定目录的路径

	dperDir := []string{}

	// 打开目录
	dir, err := os.Open(root)
	if err != nil {
		fmt.Printf("无法打开目录：%v\n", err)
		return
	}
	defer dir.Close()

	// 读取目录中的文件和子目录
	fileInfos, err := dir.Readdir(0)
	if err != nil {
		fmt.Printf("读取目录时发生错误：%v\n", err)
		return
	}

	// 遍历文件和子目录
	for _, fileInfo := range fileInfos {
		// 如果是文件，则打印文件名
		if fileInfo.IsDir() {
			dperDir = append(dperDir, fileInfo.Name())
			//fmt.Println(fileInfo.Name())
		}
	}

	portSet := make([]string, 0)

	for _, dper := range dperDir {
		pidFile := root + string(os.PathSeparator) + dper + string(os.PathSeparator) + "log" + string(os.PathSeparator) + "log" + string(os.PathSeparator) + "pid.log"

		context, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Println(err)
		}

		// 创建正则表达式模式
		re := regexp.MustCompile(`\d+$`)
		// 在字符串中查找匹配项
		port := re.FindString(string(context))

		portSet = append(portSet, port)
	}

	startScriptCmd := "bash -c \"top"
	for _, portStr := range portSet {
		startScriptCmd += pidCommand + portStr
	}
	startScriptCmd += "\"\n"

	fmt.Println(startScriptCmd)

	file, _ := os.OpenFile("./top.sh", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

	file.WriteString(startScriptCmd)
	file.Close()
}
