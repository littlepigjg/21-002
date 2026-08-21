package model

// Pagination 描述分页查询参数。
type Pagination struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// Normalize 将分页参数收敛到合法范围。
func (p Pagination) Normalize() Pagination {
	if p.Offset < 0 {
		p.Offset = 0
	}
	if p.Limit < 1 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	return p
}

// PageResult 描述一次分页查询的返回结构。
type PageResult struct {
	Items  interface{} `json:"items"`
	Total  int         `json:"total"`
	Offset int         `json:"offset"`
	Limit  int         `json:"limit"`
}
