package textutil

import "strings"

// StopwordSet 维护一份用于过滤无意义词汇的停用词集合。
// 该集合为只读，可在多个 goroutine 之间安全共享。
type StopwordSet struct {
	words map[string]struct{}
}

// NewStopwordSet 构造停用词集合，并加载内置的中英文停用词表。
func NewStopwordSet() *StopwordSet {
	s := &StopwordSet{words: make(map[string]struct{}, 512)}
	s.load(cnStopwords)
	s.load(enStopwords)
	return s
}

// load 将一批停用词写入集合，写入前统一做小写与去空白处理。
func (s *StopwordSet) load(list []string) {
	for _, w := range list {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" {
			s.words[w] = struct{}{}
		}
	}
}

// Contains 判断给定单词是否为停用词。
func (s *StopwordSet) Contains(word string) bool {
	_, ok := s.words[strings.ToLower(word)]
	return ok
}

// Filter 返回去除停用词后的 token 列表。该操作会复用输入切片底层数组，
// 调用方不应再依赖传入的 tokens 内容。
func (s *StopwordSet) Filter(tokens []string) []string {
	out := tokens[:0]
	for _, t := range tokens {
		if !s.Contains(t) {
			out = append(out, t)
		}
	}
	return out
}

// Size 返回当前停用词数量。
func (s *StopwordSet) Size() int {
	return len(s.words)
}
