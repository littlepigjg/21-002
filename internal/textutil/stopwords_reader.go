package textutil

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// LoadStopwordsFromReader 从 io.Reader 逐行读取停用词并加入集合。
// 空行与以 # 开头的注释行会被忽略。
func (s *StopwordSet) LoadStopwordsFromReader(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		s.load([]string{line})
	}
	return scanner.Err()
}

// LoadStopwordsFromFile 从指定文件加载停用词。
func (s *StopwordSet) LoadStopwordsFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.LoadStopwordsFromReader(f)
}

// Add 向集合添加单个停用词。
func (s *StopwordSet) Add(word string) {
	s.load([]string{word})
}

// Words 返回停用词集合的副本。
func (s *StopwordSet) Words() []string {
	out := make([]string, 0, len(s.words))
	for w := range s.words {
		out = append(out, w)
	}
	return out
}

// Merge 将另一个停用词集合合并到当前集合。
func (s *StopwordSet) Merge(other *StopwordSet) {
	if other == nil {
		return
	}
	for w := range other.words {
		s.words[w] = struct{}{}
	}
}
