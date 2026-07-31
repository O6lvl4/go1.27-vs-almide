// Go 1.27: goroutine リークを検出する新プロファイル "goroutineleak"。
// 実行可能な goroutine から到達不能なままブロックしている goroutine を報告する。
// 「リークは起きるもの」として観測手段を整備するのが Go 流。
package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"time"
)

func leak() {
	ch := make(chan int) // 誰も送信しないチャネル
	go func() { <-ch }() // この goroutine は永遠に回収されない
}

func main() {
	for range 3 {
		leak()
	}
	time.Sleep(100 * time.Millisecond) // ブロック状態が観測されるのを待つ

	fmt.Println("=== goroutineleak profile ===")
	pprof.Lookup("goroutineleak").WriteTo(os.Stdout, 1)
}
