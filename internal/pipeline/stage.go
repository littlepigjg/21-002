package pipeline

import "context"

// FuncStage 将普通函数包装为 Stage，便于快速组装流水线。
type FuncStage struct {
	name string
	fn   func(ctx context.Context, input interface{}) (interface{}, error)
}

// NewFuncStage 构造一个由函数驱动的处理阶段。
func NewFuncStage(name string, fn func(ctx context.Context, input interface{}) (interface{}, error)) *FuncStage {
	return &FuncStage{name: name, fn: fn}
}

// Name 返回阶段名称。
func (s *FuncStage) Name() string { return s.name }

// Run 执行阶段处理函数。
func (s *FuncStage) Run(ctx context.Context, input interface{}) (interface{}, error) {
	return s.fn(ctx, input)
}
