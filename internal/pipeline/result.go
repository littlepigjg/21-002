package pipeline

// Result 描述一次流水线执行的输出与经过的阶段。
type Result struct {
	Output interface{}
	Stages []string
}

// NewResult 构造流水线执行结果。
func NewResult(output interface{}, stages []string) Result {
	return Result{Output: output, Stages: stages}
}
