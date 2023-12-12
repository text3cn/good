package kit

import (
	"fmt"
	"os"
	"strings"
)

// 创建目录
func MkDir(dir string, mode os.FileMode) {
	var err error
	if _, err = os.Stat(dir); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(dir, mode); err != nil {
			fmt.Println("创建目录失败：" + dir)
		}
	}
}

// 字符串替换，在 subject 中将 old 替换成 new
func StrReplace(old, new, subject string, count ...int) string {
	num := -1
	if len(count) > 0 {
		num = count[0]
	}
	// -1 代表替换全部，0 代表不做替换，1 代表只替换一次
	return strings.Replace(subject, old, new, num)
}

// 把字符串重复指定次数
func StrRepeat(input string, multiplier int) string {
	return strings.Repeat(input, multiplier)
}

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
