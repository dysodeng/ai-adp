package valueobject

const defaultLimit = 20

type Pagination struct {
	page  int
	limit int
}

func NewPagination(page, limit int) Pagination {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	return Pagination{page: page, limit: limit}
}

func (p Pagination) Offset() int { return (p.page - 1) * p.limit }
func (p Pagination) Limit() int  { return p.limit }
func (p Pagination) Page() int   { return p.page }
