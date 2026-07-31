// Go 1.27: 標準ライブラリの小さな追加をまとめて。
// - strings.CutLast / bytes.CutLast
// - math/rand/v2 の Rand.N — これ自体が「ジェネリックメソッド」の言語変更で
//   初めて書けるようになった stdlib API という点がおもしろい
// - net/url の URL.Clone
package main

import (
	"fmt"
	"math/rand/v2"
	"net/url"
	"strings"
)

func main() {
	// Cut (1.18) の最後の区切り版。拡張子や末尾セグメントの分離が一発
	dir, file, _ := strings.CutLast("archive/2026/photo.tar.gz", "/")
	base, ext, _ := strings.CutLast(file, ".")
	fmt.Println(dir, "|", base, "|", ext)

	// Rand.N: トップレベルの rand.N と同じことが Rand インスタンスでできる。
	// メソッドが型パラメータ [Int intType] を持つ = ジェネリックメソッドの実用例
	r := rand.New(rand.NewPCG(1, 2))
	fmt.Println(r.N(10), r.N(int64(100)), r.N(uint8(50)))

	// URL.Clone: これまで手書きコピーしがちだった URL の複製が公式 API に
	u, _ := url.Parse("https://miyazaki-go.connpass.com/event/12345/?tab=1")
	v := u.Clone()
	v.RawQuery = "tab=2"
	fmt.Println(u.String())
	fmt.Println(v.String())
}
