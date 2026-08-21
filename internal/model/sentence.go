package model

// Sentence 表示从原文切分出的句子及其预处理结果。
type Sentence struct {
	// Index 是该句子在原文中的序号，从 0 开始。
	Index int
	// Text 是句子的原始文本。
	Text string
	// Tokens 是去除停用词后的分词结果。
	Tokens []string
}
