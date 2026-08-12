package mintegral

// PageRequest 描述以 1 为起始页码的列表请求分页参数。
type PageRequest struct {
	// Number 是从 1 开始的请求页码。
	Number int
	// Limit 是每页最多返回的条目数。
	Limit int
}

// PageInfo 描述服务端返回的当前页及列表总量信息。
type PageInfo struct {
	// Number 是服务端返回的当前页码。
	Number int
	// Limit 是服务端用于分页的每页条目数。
	Limit int
	// Total 是符合条件的条目总数。
	Total int
	// Returned 是当前页实际返回的条目数。
	Returned int
}

// Next 返回下一页请求；没有可安全继续读取的下一页时返回 false。
func (value PageInfo) Next() (PageRequest, bool) {
	if value.Number < 1 || value.Limit < 1 || value.Total < 1 || value.Returned < 1 {
		return PageRequest{}, false
	}
	totalPages := value.Total / value.Limit
	if value.Total%value.Limit != 0 {
		totalPages++
	}
	if value.Number >= totalPages {
		return PageRequest{}, false
	}
	return PageRequest{Number: value.Number + 1, Limit: value.Limit}, true
}
