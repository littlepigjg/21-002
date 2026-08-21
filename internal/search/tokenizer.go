package search

// Tokenize 将文本分词并去除停用词，返回查询词列表。
func (idx *Index) Tokenize(text string) []string {
	return tokenizeQuery(text, idx.stopwords)
}

// ExpandQuery 为查询词补充相邻字符二元组变体，提升 CJK 查询的召回率。
func ExpandQuery(tokens []string) []string {
	set := make(map[string]struct{}, len(tokens)*2)
	for _, t := range tokens {
		set[t] = struct{}{}
		rs := []rune(t)
		for i := 0; i+1 < len(rs); i++ {
			set[string(rs[i])+string(rs[i+1])] = struct{}{}
		}
	}

	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	return out
}
