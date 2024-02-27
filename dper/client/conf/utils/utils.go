package utils

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

func CopyDir(src string, dest string) {
	// 创建目标目录
	if err := os.Mkdir(dest, 0755); err != nil {
		panic(err)
	}
	// 1.读取源目录下的所有配置文件
	files, err := ioutil.ReadDir(src)
	if err != nil {
		errInfo := fmt.Sprintf("文件夹复制失败，无法读取源文件目录(%s),err:%v\n", src, err)
		panic(errInfo)
	}
	// 2.将源目录下的所有配置文件统一拷贝到目标节点目录下
	for _, file := range files {
		srcPath := filepath.Join(src, file.Name())
		destPath := filepath.Join(dest, file.Name())
		// 如果要拷贝的是一个文件夹,则需要递归拷贝
		if file.IsDir() {
			// 递归复制源文件夹下的所有文件
			CopyDir(srcPath, destPath)
		} else { // 拷贝的就是一个文件
			if err := CopyFile(srcPath, destPath); err != nil {
				errInfo := fmt.Sprintf("无法完成从源目录(%s)到目的目录(%s)的文件拷贝，err:%v\n", srcPath, destPath, err)
				panic(errInfo)
			}
		}
	}
}

func CopyFile(src string, dest string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		errInfo := fmt.Sprintf("无法打开指定路径(%s)上的源文件,err:%v\n", src, err)
		panic(errInfo)
	}
	defer srcFile.Close()

	// 创建目标文件
	destFile, err := os.Create(dest)
	if err != nil {
		errInfo := fmt.Sprintf("无法在指定目标路径(%s)上创建文件,err:%v\n", dest, err)
		panic(errInfo)
	}
	defer destFile.Close()

	// 拷贝文件内容
	if _, err := io.Copy(destFile, srcFile); err != nil {
		errInfo := fmt.Sprintf("文件拷贝失败,err:%v\n", err)
		panic(errInfo)
	}
	// 获取源文件的元数据并设置给目标文件
	srcInfo, err := os.Stat(src)
	if err != nil {
		errInfo := fmt.Sprintf("无法获取源文件(%s)状态,err:%v\n", src, err)
		panic(errInfo)
	}
	if err := os.Chmod(dest, srcInfo.Mode()); err != nil {
		errInfo := fmt.Sprintf("无法更改文件(%s)权限,err:%v\n", dest, err)
		panic(errInfo)
	}
	return nil
}

func AppendContext(filePath string, context string) error {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		errInfo := fmt.Sprintf("无法打开指定路径(%s)上的源文件,err:%v\n", filePath, err)
		panic(errInfo)
	}
	defer file.Close()
	_, err = file.WriteString(context)
	if err != nil {
		errInfo := fmt.Sprintf("无法向文件(%s)追加新内容(%s),err:%v\n:", filePath, context, err)
		panic(errInfo)
	}
	return nil
}

func MkFile(path string, content string) {
	var file *os.File
	var err error
	if file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0777); err != nil {
		panic(err)
	}
	file.WriteString(content)
	file.Close()
}

func MkDir(path string) {
	if err := os.Mkdir(path, 0755); err != nil {
		panic(err)
	}
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

func GetParentDirectory(dirctory string) string {
	return substr(dirctory, 0, strings.LastIndex(dirctory, string(os.PathSeparator)))
}

func GetCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

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

// 跳转到指定文件目录下，运行指定文件名的文件，且添加命令行参数
func RunExecFile(dir string, execFile string, arg string, wg *sync.WaitGroup) {
	os.Chdir(dir)
	cmd := exec.Command(execFile, arg)
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		return
	}
	cmd.Wait()
	defer wg.Done()
}

func RemoveDirContents(dirPath string) error {
	// 打开文件夹
	dir, err := os.Open(dirPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	// 读取文件夹中的文件和子文件夹
	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	// 递归删除文件和子文件夹
	for _, fileInfo := range fileInfos {
		filePath := filepath.Join(dirPath, fileInfo.Name())

		// 如果是子文件夹，则递归删除
		if fileInfo.IsDir() {
			err := RemoveDirContents(filePath)
			if err != nil {
				return err
			}
		} else {
			// 如果是文件，则删除
			err := os.Remove(filePath)
			if err != nil {
				return err
			}
		}
	}

	// 删除空文件夹
	err = os.Remove(dirPath)
	if err != nil {
		return err
	}

	return nil
}
