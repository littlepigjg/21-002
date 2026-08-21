package service

// AnalyzeOptions 是分析过程的可调参数集合。
type AnalyzeOptions struct {
	MaxKeywords     int
	MaxSentences    int
	TextRankMaxIter int
	TextRankDamping float64
}

// DefaultAnalyzeOptions 返回默认分析参数。
func DefaultAnalyzeOptions() AnalyzeOptions {
	return AnalyzeOptions{
		MaxKeywords:     10,
		MaxSentences:    5,
		TextRankMaxIter: 30,
		TextRankDamping: 0.85,
	}
}

// Option 是修改 AnalyzeOptions 的函数式选项。
type Option func(*AnalyzeOptions)

// WithMaxKeywords 设置关键词数量上限。
func WithMaxKeywords(n int) Option {
	return func(o *AnalyzeOptions) { o.MaxKeywords = n }
}

// WithMaxSentences 设置摘要句子数量上限。
func WithMaxSentences(n int) Option {
	return func(o *AnalyzeOptions) { o.MaxSentences = n }
}

// WithTextRank 设置 TextRank 迭代次数与阻尼系数。
func WithTextRank(iter int, damping float64) Option {
	return func(o *AnalyzeOptions) {
		o.TextRankMaxIter = iter
		o.TextRankDamping = damping
	}
}

// ApplyOptions 将一组选项应用到默认参数上。
func ApplyOptions(opts ...Option) AnalyzeOptions {
	o := DefaultAnalyzeOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
