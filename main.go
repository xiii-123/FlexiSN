package main

import (
	_ "main/cmd"
	"main/run"
)

func main() {
	// 启动整个程序，所有逻辑在run包中执行
	run.Start()

	//	file, err := os.Open("hello.txt")
	//	defer file.Close()
	//	bufFile := bufio.NewReader(file)
	//	buf := make([]byte, 1024)
	//	n, err := bufFile.Read(buf)
	//	if err != nil {
	//		return
	//	}
	//	os.Stdout.Write(buf[:n])
}
