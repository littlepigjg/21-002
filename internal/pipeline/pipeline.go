package pipeline

import "context"

// Stage 是流水线中的一个处理阶段。
type Stage interface {
	Name() string
	Run(ctx context.Context, input interface{}) (interface{}, error)
}

// Pipeline 按顺序执行一组处理阶段。
type Pipeline struct {
	stages []Stage
}

// New 构造一个包含指定阶段的流水线。
func New(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// Add 向流水线追加一个阶段，并返回自身以便链式调用。
func (p *Pipeline) Add(stage Stage) *Pipeline {
	p.stages = append(p.stages, stage)
	return p
}

// Run 依次执行所有阶段，任一阶段失败或上下文取消时立即返回。
func (p *Pipeline) Run(ctx context.Context, input interface{}) (interface{}, error) {
	var err error
	current := input

	for _, s := range p.stages {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		current, err = s.Run(ctx, current)
		if err != nil {
			return nil, err
		}
	}
	return current, nil
}

// Names 返回流水线中所有阶段的名称。
func (p *Pipeline) Names() []string {
	names := make([]string, len(p.stages))
	for i, s := range p.stages {
		names[i] = s.Name()
	}
	return names
}

// Len 返回流水线中的阶段数量。
func (p *Pipeline) Len() int {
	return len(p.stages)
}
