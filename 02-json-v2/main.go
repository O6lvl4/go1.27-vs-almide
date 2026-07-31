// Go 1.27: encoding/json/v2 が正式に標準ライブラリ入り。
// 従来の encoding/json も内部実装が v2 に置き換わった (GOEXPERIMENT=nojsonv2 で戻せる)。
// v2 はデフォルトが厳格: 重複キー拒否・不正 UTF-8 拒否・フィールド名は大文字小文字を区別。
package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"time"
)

type Event struct {
	Name  string    `json:"name"`
	Held  time.Time `json:"held"`           // v2 でも time.Time は RFC 3339。実験版にあった format タグは 1.27 正式版では未サポート
	Venue string    `json:"venue,omitzero"` // omitempty より直感的な omitzero
}

func main() {
	ev := Event{
		Name: "Miyazaki.go Go v1.27rc Sneak Peek #5",
		Held: time.Date(2026, 7, 31, 19, 0, 0, 0, time.Local),
	}

	// Options を可変長引数で渡す。整形は jsontext.WithIndent
	out, _ := json.Marshal(ev, jsontext.WithIndent("  "))
	fmt.Println(string(out))

	// v1 は黙って後勝ちにしていた重複キーを、v2 はエラーにする
	var m map[string]int
	err := json.Unmarshal([]byte(`{"a":1,"a":2}`), &m)
	fmt.Println("duplicate key:", err)

	// v1 の緩い挙動が必要な場合はオプトインで戻せる
	err = json.Unmarshal([]byte(`{"a":1,"a":2}`), &m, jsontext.AllowDuplicateNames(true))
	fmt.Println("allow duplicates:", m, err)

	// フィールド名マッチも v1 は case-insensitive、v2 は exact match がデフォルト
	var ev2 Event
	err = json.Unmarshal([]byte(`{"NAME":"x"}`), &ev2)
	fmt.Println("case mismatch ignored:", ev2.Name == "", err)
	_ = json.Unmarshal([]byte(`{"NAME":"x"}`), &ev2, json.MatchCaseInsensitiveNames(true))
	fmt.Println("case-insensitive opt-in:", ev2.Name)
}
