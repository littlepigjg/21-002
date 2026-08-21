package service

import (
	"context"

	"summarizer/internal/pipeline"
	"summarizer/internal/textutil"
)

// BuildCleanPipeline 构造一个文本清洗流水线：
// 规范化 -> 全角转半角 -> 空白折叠。
func BuildCleanPipeline() *pipeline.Pipeline {
	return pipeline.New(
		pipeline.NewFuncStage("normalize", func(ctx context.Context, in interface{}) (interface{}, error) {
			s, _ := in.(string)
			return textutil.Normalize(s), nil
		}),
		pipeline.NewFuncStage("fullwidth_to_halfwidth", func(ctx context.Context, in interface{}) (interface{}, error) {
			s, _ := in.(string)
			return textutil.FullWidthToHalfWidth(s), nil
		}),
		pipeline.NewFuncStage("compact_space", func(ctx context.Context, in interface{}) (interface{}, error) {
			s, _ := in.(string)
			return textutil.CompactSpace(s), nil
		}),
	)
}

// CleanText 使用清洗流水线处理文本。
func CleanText(ctx context.Context, text string) (string, error) {
	out, err := BuildCleanPipeline().Run(ctx, text)
	if err != nil {
		return "", err
	}
	s, _ := out.(string)
	return s, nil
}
