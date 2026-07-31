# Go 1.27 × almide

[Miyazaki.go Go v1.27rc Sneak Peek #5](https://miyazaki-go.connpass.com/) (2026-07-31) 向けの比較ノート。
Go 1.27 の新機能を、自作言語 [almide](https://github.com/almide/almide) では同じ問題をどう解いているかと並べて眺める。

**全サンプルは実際に動かして検証済み**: `go1.27rc2 darwin/arm64` / `almide 0.41.0`。
以下のコードはすべて各ディレクトリの `main.go` / `main.almd` からの抜粋。

## 実行方法

```bash
# Go 側 (go.mod が go 1.27rc2 を指しているので、Go 1.21+ ならツールチェーンが自動取得される)
go run ./01-generic-methods

# almide 側 (https://github.com/almide/almide)
almide run 01-generic-methods/main.almd

# 全部まとめて
./demo.sh
```

## テーマ

| # | Go 1.27 | almide |
|---|---------|--------|
| [01](01-generic-methods/) | ジェネリックメソッド（言語変更の目玉） | UFCS + ジェネリック自由関数 |
| [02](02-json-v2/) | `encoding/json/v2` 正式入り | `Codec` プロトコル + `json` モジュール |
| [03](03-uuid/) | `uuid` パッケージが stdlib 入り | 存在しないので 20 行で自作 |
| [04](04-stdlib-bits/) | `strings.CutLast` / `Rand.N` / `URL.Clone` | `last_index_of` + slice / `random.int` / record spread |
| [05](05-concurrency/) | `goroutineleak` プロファイル | `fan` 構造化並行 |

## 01 — ジェネリックメソッド

### Go 1.27

メソッド宣言が独自の型パラメータを持てるようになった。これまで package スコープの関数として書くしかなかったものを、型の名前空間に置ける。

```go
type Stack[T any] struct{ items []T }

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

// メソッド自身が型パラメータ U を宣言する（Go 1.26 まではコンパイルエラー）
func (s *Stack[T]) Map[U any](f func(T) U) *Stack[U] {
	out := &Stack[U]{}
	for _, v := range s.items {
		out.Push(f(v))
	}
	return out
}

// 型引数は関数リテラルから推論される
labels := s.Map(func(v int) string { return fmt.Sprintf("<%d>", v) })

// 制限: interface のメソッドは型パラメータを宣言できない。
// ジェネリックメソッドで interface を実装することもできない。
```

### almide

「メソッド」という概念自体がほぼない。関数の第 1 引数の型が合えば `s.map_each(f)` と呼べる（UFCS）ので、**型の名前空間にジェネリック関数を置きたいという動機自体が発生しない**。stdlib の `list.map[A, B]` が `xs.map(f)` と呼べるのも同じ仕組み。

```almd
type Stack[T] = { items: List[T] }

fn stack_new[T]() -> Stack[T] = { items: [] }

fn push[T](s: Stack[T], v: T) -> Stack[T] =
  { items: list.insert(s.items, list.len(s.items), v) }

// Go 1.27 の func (s *Stack[T]) Map[U any](f func(T) U) に相当
fn map_each[T, U](s: Stack[T], f: fn(T) -> U) -> Stack[U] =
  { items: list.map(s.items, f) }

// 明示的な型引数は Go と同じく [] で渡し、UFCS でメソッド風に呼ぶ
let s = stack_new[Int]().push(1).push(2).push(3)
let labels = s.map_each((v) => "<${int.to_string(v)}>")
```

なお almide の慣習メソッド `fn Type.method` はジェネリック型には未対応（SPEC §5.5）。
Go が 8 年かけてメソッドに型パラメータを持ち込んだ問題を、almide は「メソッドを特別扱いしない」ことで回避している、という対比。

## 02 — encoding/json/v2

### Go 1.27

`encoding/json/v2` が正式に stdlib 入り。既存の `encoding/json` も内部実装が v2 になった（`GOEXPERIMENT=nojsonv2` で戻せる）。
v2 のデフォルトは厳格: 重複キー拒否・不正 UTF-8 拒否・フィールド名は exact match。緩めたい場合は `Options` でオプトイン。

```go
import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

type Event struct {
	Name  string    `json:"name"`
	Held  time.Time `json:"held"`           // v2 でも time.Time は RFC 3339
	Venue string    `json:"venue,omitzero"` // omitempty より直感的な omitzero
}

// Options を可変長引数で渡す。整形は jsontext.WithIndent
out, _ := json.Marshal(ev, jsontext.WithIndent("  "))

// v1 は黙って後勝ちにしていた重複キーを、v2 はエラーにする
err := json.Unmarshal([]byte(`{"a":1,"a":2}`), &m)
// => jsontext: duplicate object member name "a"

// v1 の緩い挙動が必要な場合はオプトインで戻せる
json.Unmarshal([]byte(`{"a":1,"a":2}`), &m, jsontext.AllowDuplicateNames(true))
json.Unmarshal([]byte(`{"NAME":"x"}`), &ev, json.MatchCaseInsensitiveNames(true))
```

rc2 で実測して分かったこと: **実験版 json/v2 にあった `format` タグ（`format:DateOnly` 等）は 1.27 正式版では未サポート**。`unsupported format tag option` エラーになる。

### almide

struct タグの代わりに `Codec` プロトコルを型に付けると `encode`/`decode` が導出される。
`decode` は最初から `Result` を返すので、フィールド欠けは型レベルで失敗する。

```almd
import json

type Event: Codec = {
  name: String,
  held: String,
  venue: String,
}

println(json.stringify_pretty(ev.encode()))

// Go v2 がエラーにする重複キーを、almide は両方保持する
match json.parse("{\"a\": 1, \"a\": 2}") {
  ok(v) => println("accepted: ${json.stringify(v)}"),  // => accepted: {"a":1,"a":2}
  err(e) => println("rejected: ${e}"),
}

// decode の厳格さ: フィールドが欠けた JSON は Result の err になる
match Event.decode(v) {
  ok(e) => println("decoded: ${e.name}"),
  err(msg) => println("decode rejected: ${msg}"),  // => missing field 'held'
}
```

重複キー `{"a":1,"a":2}` の扱いは三者三様:

- Go v1: 黙って後勝ち
- Go v2: エラー（`jsontext.AllowDuplicateNames(true)` でオプトイン）
- almide: **両方保持**（`json.stringify` すると `{"a":1,"a":2}` がそのまま出てくる）

## 03 — uuid

### Go 1.27

`uuid` が stdlib 入り（RFC 9562）。乱数源は CSPRNG。google/uuid 等のサードパーティ依存が要らなくなる。

```go
import "uuid"

u := uuid.New() // ランダムベース (V4 相当)

// V7 はタイムスタンプ先頭なので生成順にソート可能。DB の主キー向き
ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
slices.SortFunc(ids, uuid.UUID.Compare)

// Parse / MustParse、encoding.TextMarshaler 実装も揃っている
p, err := uuid.Parse(u.String())
```

### almide

UUID がない（stdlib にも姉妹パッケージにもない）ので、`random` + ビット演算で V4 を自作した。

```almd
import random

fn to_hex2(n: Int) -> String = string.pad_start(int.to_hex(n), 2, "0")

effect fn uuid_v4() -> String = {
  var bs: List[Int] = []
  for _ in 0..<16 {
    bs.push(random.int(0, 255))
  }
  // version 4 (上位ニブルを 0100 に)、variant 10xx
  let b6 = int.bor(int.band(list.get_or(bs, 6, 0), 15), 64)
  let b8 = int.bor(int.band(list.get_or(bs, 8, 0), 63), 128)
  let fixed = list.set(list.set(bs, 6, b6), 8, b8)
  let hx = list.join(list.map(fixed, (b) => to_hex2(b)), "")
  string.slice(hx, 0, 8) + "-" + string.slice(hx, 8, 12) + "-" +
    string.slice(hx, 12, 16) + "-" + string.slice(hx, 16, 20) + "-" +
    string.slice(hx, 20, 32)
}

// effect fn の呼び出しは暗黙に Result になる(効果は失敗しうる)ので ?? で剥がす
println(uuid_v4() ?? "")
// => 435d3f13-6fd3-4cc5-a4af-489277a1c0a5
```

本質的な差は乱数源: almide の `random` は seed 指定も CSPRNG もない最小 API（4 関数）で、
「stdlib に UUID を入れる」なら Go 同様まず乱数の品質保証が要る、という話につながる。

## 04 — stdlib 小ネタ

### Go 1.27

```go
// Cut (1.18) の最後の区切り版。拡張子や末尾セグメントの分離が一発
dir, file, _ := strings.CutLast("archive/2026/photo.tar.gz", "/")
base, ext, _ := strings.CutLast(file, ".")

// Rand.N: これ自体がジェネリックメソッドの言語変更で初めて書けるようになった stdlib API
// func (r *Rand) N[Int intType](n Int) Int
r := rand.New(rand.NewPCG(1, 2))
fmt.Println(r.N(10), r.N(int64(100)), r.N(uint8(50)))

// URL.Clone: これまで手書きコピーしがちだった URL の複製が公式 API に
v := u.Clone()
v.RawQuery = "tab=2"
```

### almide

```almd
// CutLast はないが last_index_of + slice で 5 行
fn cut_last(s: String, sep: String) -> (String, String, Bool) =
  match string.last_index_of(s, sep) {
    some(i) => (string.slice(s, 0, i), string.drop(s, i + string.len(sep)), true),
    none => (s, "", false),
  }

// Rand.N 相当は random.int(min, max)。ただし Int のみ・seed なし
println(int.to_string(random.int(0, 9)))

// record は immutable なので clone は spread 構文がタダでくれる
let u = Url { host: "miyazaki-go.connpass.com", path: "/event/12345/", query: "tab=1" }
let v = { ...u, query: "tab=2" }  // u はそのまま
```

## 05 — 並行性: 観測 vs 構造

### Go 1.27

`goroutineleak` プロファイルが追加された。実行可能な goroutine から到達不能なままブロックしている goroutine を検出する。

```go
func leak() {
	ch := make(chan int) // 誰も送信しないチャネル
	go func() { <-ch }() // この goroutine は永遠に回収されない
}

func main() {
	for range 3 {
		leak()
	}
	time.Sleep(100 * time.Millisecond)
	pprof.Lookup("goroutineleak").WriteTo(os.Stdout, 1)
}
```

実測出力（3 つのリークがきっちり報告される）:

```
goroutineleak profile: total 3
3 @ 0x1029abce8 0x102947080 0x102946c04 0x1029ff374 0x1029b1de4
#	0x1029ff373	main.leak.func1+0x23	.../05-concurrency/main.go:15
```

### almide

並行性は `fan` ブロックだけ。SPEC には「どれかが err なら兄弟はキャンセル」とあり、
**「野良タスク」が構造上作れないのでリーク検出器が要らない** — というのが建前。実測は後述。

```almd
import env

effect fn work(name: String, ms: Int) -> Result[String, String] = {
  env.sleep_ms(ms)
  ok("${name} done (${int.to_string(ms)}ms)")
}

effect fn main() -> Unit = {
  // ネイティブでは実 OS スレッドで並行実行、結果はタプルで揃って返る
  let (a, b, c) = fan {
    work("A", 30)
    work("B", 10)
    work("C", 20)
  }
  println(a)  // => A done (30ms)
}
```

### 意地悪テスト: 「キャンセル」は本物か

SPEC の「first error cancels all siblings」を almide 0.41.0 の実バイナリで攻撃してみた結果:

| 実験 | 期待（SPEC） | 実測 |
|---|---|---|
| 1 秒スリープ ×3 を `fan` | 並行実行 | ✅ 約 1 秒で完了。**並行は本物**（実 OS スレッド） |
| 50ms で err + 5 秒スリープの兄弟 | 兄弟はキャンセル | ❌ **5 秒完走**。スリープ後の `println` まで実行された。fan は「全員の決着を待ってから最初の err を返す」 |
| 即 err + 終わらない兄弟（sleep 入り無限ループ） | キャンセルして err | ❌ **fan ごと永久ブロック**（30 秒監視でもハング） |
| `fan.race([slow(2s), fast(10ms)])` | 先に終わった fast が勝つ | ❌ **slow が勝者**（2 秒待つ）。race は「先着」ではなく**リスト先頭の結果を返す決定論**（cross-target 契約 C-005 に明記済み — SPEC の prose と矛盾） |
| race の敗者は生き残るか（勝者確定後 3 秒監視） | — | ✅ 敗者の痕跡なし。**タスクがスコープから漏れることはなかった** |

なので正確な結論はこう:

- 「野良タスクが作れない」は**真** — タスクはスコープの外に漏れない。Go 的な「リーク」は起きない。
- 「err なら兄弟をキャンセル」は**現状は未実装** — キャンセルではなく join（完走待ち）。
- その帰結として、**終わらないタスクは fan ごと道連れにする**。Go で goroutine リークになる事故は、almide では「ハング」に変換される — そしてハングの検出器はまだない。

Go は「リークは起きるもの」として観測手段を整備し、almide は「リークを構造で禁じた」結果、同じ事故が別の形で現れる。
20 年近い後方互換を背負う言語と、2026 年に設計された言語（の、仕様と実装のギャップ）の対比がいちばん出るところ。

## まとめ

| 観点 | Go 1.27 | almide 0.41 |
|---|---|---|
| ジェネリクスの置き場 | メソッドにも型パラメータ（8 年越し） | UFCS で最初からメソッド不要 |
| JSON の厳格さ | v2 で opt-out 方式に転換 | Codec + Result で最初から厳格 |
| バッテリー | uuid など「みんな使うもの」を取り込み続ける | 965 関数 / 41 モジュール + git 依存 |
| 並行性の安全 | リークを観測するツールを足す | リークを構造で禁じる（ただしキャンセルは未実装でハングに変換） |

Go の 1.27 は「後方互換を守りながら、厳格さと観測可能性をオプトインで足していく」リリース。
almide は LLM 生成コード前提で「最初から厳格・構造的」に振っている。
どちらも「デフォルトを間違えると 10 年引きずる」ことの教材。
