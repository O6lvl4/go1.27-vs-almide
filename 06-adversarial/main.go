// almide の fan に仕掛けた意地悪テストを、そのまま Go にぶつける。
//
// 意地悪1: エラー発生時、スリープ中の兄弟を止められるか
// 意地悪2: race の敗者はどうなるか
// 意地悪3: 終わらないタスクを抱えたら Wait はどうなるか
package main

import (
	"context"
	"fmt"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

// ── 意地悪1: err + スリープ中の兄弟 ──────────────────────────────
// Go のキャンセルは context 経由の協調式。time.Sleep は ctx を見ない。
func test1() {
	fmt.Println("━━ 意地悪1: 50ms で err、兄弟は 5 秒スリープ")
	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	wg.Go(func() { // boom: 50ms で「エラー」→ cancel
		time.Sleep(50 * time.Millisecond)
		fmt.Printf("  boom: err @ %v\n", time.Since(start).Round(time.Millisecond))
		cancel()
	})
	wg.Go(func() { // ctx を無視する素朴な兄弟 (almide の env.sleep_ms と同じ立場)
		time.Sleep(5 * time.Second)
		fmt.Printf("  SLOW COMPLETED ANYWAY @ %v (ctx 無視版)\n", time.Since(start).Round(time.Millisecond))
	})
	wg.Go(func() { // ctx を見るように書いた兄弟だけが救われる
		select {
		case <-time.After(5 * time.Second):
			fmt.Println("  unreachable")
		case <-ctx.Done():
			fmt.Printf("  ctx 対応版: cancelled @ %v\n", time.Since(start).Round(time.Millisecond))
		}
	})
	wg.Wait()
	fmt.Printf("  Wait() 完了 @ %v — 全員の完走を待つのは almide と同じ\n\n", time.Since(start).Round(time.Millisecond))
}

// ── 意地悪2: race の敗者 ─────────────────────────────────────────
// select は本物のレース (10ms の方が勝つ)。ただし敗者は放置される。
func raceOnce() string {
	res := make(chan string) // unbuffered: 敗者は送信で永遠にブロック
	go func() {
		time.Sleep(2 * time.Second)
		res <- "slow" // 誰も受信しない
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		res <- "fast"
	}()
	return <-res
}

func test2() {
	fmt.Println("━━ 意地悪2: race([slow 2s, fast 10ms])")
	start := time.Now()
	fmt.Printf("  winner: %s @ %v — select は本物のレース\n", raceOnce(), time.Since(start).Round(time.Millisecond))

	time.Sleep(2500 * time.Millisecond) // 敗者が送信ブロックに到達するのを待つ
	fmt.Println("  勝者確定の 2.5 秒後、goroutineleak プロファイルを見ると:")
	pprof.Lookup("goroutineleak").WriteTo(os.Stdout, 1)
	fmt.Println("  → 敗者はキャンセルされず「リーク」として生きている。1.27 はそれを観測できる")
	fmt.Println()
}

// ── 意地悪3: 終わらないタスクを抱えた Wait ───────────────────────
func test3() {
	fmt.Println("━━ 意地悪3: cancel の効かない不死身タスクを Wait する")
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { // ctx を見ないループ: 殺す手段はない (almide と同じ)
		for {
			time.Sleep(100 * time.Millisecond)
		}
	})
	cancel()
	_ = ctx

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		fmt.Println("  unreachable")
	case <-time.After(2 * time.Second):
		fmt.Println("  Wait() は 2 秒待っても返らない — almide の fan なら、ここで永久ハング")
	}
	fmt.Println("  ただし Go は Wait を諦めて先へ進める(=リークとして抱えたまま生きる)自由がある")
	fmt.Println("  なお sleep ループは「ブロック」ではないので goroutineleak にも映らない死角")
}

func main() {
	test1()
	test2()
	test3()
}
