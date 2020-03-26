package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	//test1()
	//test2()
	test3()
}

// chan 是否会一直存在的测试
func test1() {
	ch := make(chan string)
	go func() {
		ch <- "hello"
	}()
	for i := 1; i <= 10; i++ {
		hello := <-ch
		fmt.Println(hello)
	}
}

// io.Copy 函数的测试
func test2() {
	// 会让我一直输入
	io.Copy(os.Stdout, os.Stdin)
	fmt.Println("done!")
}

// map 遍历值测试
func test3() {
	mmp := make(map[string]int)
	mmp["hello"] = 1
	mmp["world"] = 2
	for m := range mmp {
		fmt.Println(m)
	}
}