// Go 1.27: ジェネリックメソッド
// メソッド宣言が独自の型パラメータを持てるようになった。
// これまで package スコープの関数 (Map[T, U any](s *Stack[T], ...)) として
// 書くしかなかったものを、型の名前空間に置ける。
package main

import (
	"fmt"
	"strings"
)

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

// メソッド自身が型パラメータ U を宣言する（Go 1.26 まではコンパイルエラー）
func (s *Stack[T]) Map[U any](f func(T) U) *Stack[U] {
	out := &Stack[U]{}
	for _, v := range s.items {
		out.Push(f(v))
	}
	return out
}

func (s *Stack[T]) Fold[A any](init A, f func(A, T) A) A {
	acc := init
	for _, v := range s.items {
		acc = f(acc, v)
	}
	return acc
}

func main() {
	s := &Stack[int]{}
	s.Push(1)
	s.Push(2)
	s.Push(3)

	// 型引数は関数リテラルから推論される
	labels := s.Map(func(v int) string { return fmt.Sprintf("<%d>", v) })
	fmt.Println(labels.items)

	// 明示的なインスタンス化も可能
	joined := labels.Fold[string]("", func(acc, v string) string {
		return strings.TrimPrefix(acc+"-"+v, "-")
	})
	fmt.Println(joined)

	// 制限: interface のメソッドは型パラメータを宣言できない。
	// ジェネリックメソッドで interface を実装することもできない。
}
