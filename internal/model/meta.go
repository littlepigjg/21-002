package model

// Service 元信息常量。
const (
	ServiceName    = "summarizer"
	ServiceVersion = "1.0.0"
	ServiceDesc    = "文章自动摘要与关键词提取系统"
)

// ServiceInfo 描述服务元信息。
type ServiceInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Desc    string `json:"desc"`
}

// BuildServiceInfo 返回服务元信息。
func BuildServiceInfo() ServiceInfo {
	return ServiceInfo{
		Name:    ServiceName,
		Version: ServiceVersion,
		Desc:    ServiceDesc,
	}
}
