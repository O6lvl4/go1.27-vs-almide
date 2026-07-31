# Go 1.27 × almide

[Miyazaki.go Go v1.27rc Sneak Peek #5](https://miyazaki-go.connpass.com/) (2026-07-31) 向けの比較ノート。
Go 1.27 の新機能を、自作言語 [almide](https://github.com/almide/almide) では同じ問題をどう解いているかと並べて眺める。

**全サンプルは実際に動かして検証済み**: `go1.27rc2 darwin/arm64` / `almide 0.41.0`。

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

Go 1.27 でメソッド宣言が独自の型パラメータを持てるようになった。

```go
func (s *Stack[T]) Map[U any](f func(T) U) *Stack[U]
```

これまで package スコープの関数として書くしかなかったものを、型の名前空間に置ける。
制限: interface のメソッドは型パラメータを宣言できず、ジェネリックメソッドで interface を実装することもできない。

almide には「メソッド」という概念自体がほぼない。関数の第 1 引数の型が合えば `s.map_each(f)` と呼べる（UFCS）ので、
**型の名前空間にジェネリック関数を置きたいという動機自体が発生しない**。stdlib の `list.map[A, B]` が `xs.map(f)` と呼べるのも同じ仕組み。

```almd
fn map_each[T, U](s: Stack[T], f: fn(T) -> U) -> Stack[U] =
  { items: list.map(s.items, f) }
```

なお almide の慣習メソッド `fn Type.method` はジェネリック型には未対応（SPEC §5.5）。
Go が 8 年かけてメソッドに型パラメータを持ち込んだ問題を、almide は「メソッドを特別扱いしない」ことで回避している、という対比。

## 02 — encoding/json/v2

Go 1.27 で `encoding/json/v2` が正式に stdlib 入り。既存の `encoding/json` も内部実装が v2 になった（`GOEXPERIMENT=nojsonv2` で戻せる）。
v2 のデフォルトは厳格: 重複キー拒否・不正 UTF-8 拒否・フィールド名は exact match。緩めたい場合は `Options` でオプトイン。

rc2 で実測して分かったこと:

- **実験版 json/v2 にあった `format` タグ（`format:DateOnly` 等）は 1.27 正式版では未サポート**。`unsupported format tag option` エラーになる。
- 重複キー `{"a":1,"a":2}` の扱いは三者三様:
  - Go v1: 黙って後勝ち
  - Go v2: エラー（`jsontext.AllowDuplicateNames(true)` でオプトイン）
  - almide: **両方保持**（`json.stringify` すると `{"a":1,"a":2}` がそのまま出てくる）

almide は struct タグの代わりに `Codec` プロトコルを型に付けると `encode`/`decode` が導出される。
`decode` は最初から `Result` を返すので、フィールド欠けは型レベルで失敗する（`missing field 'held'`）。

## 03 — uuid

Go 1.27 で `uuid` が stdlib 入り（RFC 9562）。`New` / `NewV4` / `NewV7` / `Parse`、乱数源は CSPRNG。
V7 はタイムスタンプ先頭なので生成順にソートでき、`slices.SortFunc(ids, uuid.UUID.Compare)` がそのまま書ける。

almide には UUID がない（stdlib にも姉妹パッケージにもない）ので、`random` + ビット演算で V4 を自作した。
本質的な差は乱数源: almide の `random` は seed 指定も CSPRNG もない最小 API（4 関数）で、
「stdlib に UUID を入れる」なら Go 同様まず乱数の品質保証が要る、という話につながる。

## 04 — stdlib 小ネタ

- `strings.CutLast` / `bytes.CutLast`: `Cut` (1.18) の最後の区切り版。almide にはないが `string.last_index_of` + `slice` で 5 行。
- `math/rand/v2` の `Rand.N`: **これ自体がジェネリックメソッドの言語変更で初めて書けるようになった stdlib API**。`func (r *Rand) N[Int intType](n Int) Int`。
- `net/url` の `URL.Clone`: Go では手書きコピーしがちだった複製が公式 API に。almide は record が immutable なので `{ ...u, query: "tab=2" }` — **clone は言語がタダでくれる**。

## 05 — 並行性: 観測 vs 構造

Go 1.27 は `goroutineleak` プロファイルを追加した。実行可能な goroutine から到達不能なままブロックしている goroutine を検出する。
実測でもチャネル待ちで放置した goroutine 3 つがきっちり報告される。

almide の並行性は `fan` ブロックだけ。スコープを抜ける時点で全タスクは完了済みか（どれかが err なら）キャンセル済みで、
**「野良タスク」が構造上作れないのでリーク検出器が要らない**。

```almd
let (a, b, c) = fan {
  work("A", 30)
  work("B", 10)
  work("C", 20)
}
```

Go は「リークは起きるもの」として観測手段を整備し、almide は「リークを構文で封じる」。
20 年近い後方互換を背負う言語と、2026 年に設計された言語の対比がいちばん出るところ。

## まとめ

| 観点 | Go 1.27 | almide 0.41 |
|---|---|---|
| ジェネリクスの置き場 | メソッドにも型パラメータ（8 年越し） | UFCS で最初からメソッド不要 |
| JSON の厳格さ | v2 で opt-out 方式に転換 | Codec + Result で最初から厳格 |
| バッテリー | uuid など「みんな使うもの」を取り込み続ける | 965 関数 / 41 モジュール + git 依存 |
| 並行性の安全 | リークを観測するツールを足す | リークを構造で封じる |

Go の 1.27 は「後方互換を守りながら、厳格さと観測可能性をオプトインで足していく」リリース。
almide は LLM 生成コード前提で「最初から厳格・構造的」に振っている。
どちらも「デフォルトを間違えると 10 年引きずる」ことの教材。
