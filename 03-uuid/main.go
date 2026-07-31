// Go 1.27: 標準ライブラリに uuid パッケージが追加された (RFC 9562)。
// google/uuid 等のサードパーティ依存なしで UUID を生成・解析できる。
package main

import (
	"fmt"
	"slices"
	"uuid"
)

func main() {
	// New はランダムベース (V4 相当) の UUID
	u := uuid.New()
	fmt.Println("New:   ", u)

	// V7 はタイムスタンプ先頭なので生成順にソート可能。DB の主キー向き
	ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	slices.SortFunc(ids, uuid.UUID.Compare)
	for _, id := range ids {
		fmt.Println("V7:    ", id)
	}

	// Parse / MustParse、encoding.TextMarshaler 実装も揃っている
	p, err := uuid.Parse(u.String())
	fmt.Println("Parse: ", p, err == nil)
	fmt.Println("Nil:   ", uuid.Nil())
}
