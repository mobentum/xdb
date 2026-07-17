package xdb

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPage_Offset_Default(t *testing.T) {
	p := Page{}
	assert.Equal(t, 0, p.offset())
}

func TestPage_Offset_FirstPage(t *testing.T) {
	p := Page{Number: 1, Size: 20}
	assert.Equal(t, 0, p.offset())
}

func TestPage_Offset_SecondPage(t *testing.T) {
	p := Page{Number: 2, Size: 20}
	assert.Equal(t, 20, p.offset())
}

func TestPage_Offset_ThirdPage(t *testing.T) {
	p := Page{Number: 3, Size: 50}
	assert.Equal(t, 100, p.offset())
}

func TestPage_Offset_ZeroPage(t *testing.T) {
	p := Page{Number: 0, Size: 20}
	assert.Equal(t, 0, p.offset())
}

func TestPage_Offset_NegativePage(t *testing.T) {
	p := Page{Number: -5, Size: 20}
	assert.Equal(t, 0, p.offset())
}

func TestPage_Size_Default(t *testing.T) {
	p := Page{}
	assert.Equal(t, 20, p.size())
}

func TestPage_Size_Custom(t *testing.T) {
	p := Page{Size: 50}
	assert.Equal(t, 50, p.size())
}

func TestPage_Size_Zero(t *testing.T) {
	p := Page{Size: 0}
	assert.Equal(t, 20, p.size())
}

func TestPage_Size_Negative(t *testing.T) {
	p := Page{Size: -10}
	assert.Equal(t, 20, p.size())
}

func TestPageResult_Structure(t *testing.T) {
	result := &PageResult[string]{
		Items:      []string{"a", "b", "c"},
		Total:      100,
		Page:       1,
		PageSize:   20,
		TotalPages: 5,
	}

	assert.Len(t, result.Items, 3)
	assert.Equal(t, 100, result.Total)
	assert.Equal(t, 5, result.TotalPages)
}

func TestPageResult_EmptyItems(t *testing.T) {
	result := &PageResult[int]{
		Items:      []int{},
		Total:      0,
		Page:       1,
		PageSize:   20,
		TotalPages: 0,
	}

	assert.Empty(t, result.Items)
	assert.Equal(t, 0, result.TotalPages)
}

func TestPage_LargePage(t *testing.T) {
	p := Page{Number: 1000, Size: 100}
	assert.Equal(t, 99900, p.offset())
}

func TestPage_OneItemPerPage(t *testing.T) {
	p := Page{Number: 5, Size: 1}
	assert.Equal(t, 4, p.offset())
}

func TestPageResult_TotalPagesCalculation(t *testing.T) {
	tests := []struct {
		name          string
		total         int
		pageSize      int
		expectedPages int
	}{
		{"exact pages", 100, 20, 5},
		{"remainder", 101, 20, 6},
		{"one page", 10, 20, 1},
		{"zero total", 0, 20, 0},
		{"large remainder", 99, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &PageResult[string]{
				Total:      tt.total,
				PageSize:   tt.pageSize,
				TotalPages: (tt.total + tt.pageSize - 1) / tt.pageSize,
			}
			assert.Equal(t, tt.expectedPages, result.TotalPages)
		})
	}
}

func TestPaginateWithCount_ReflectHelper(t *testing.T) {
	type testItem struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}

	typ := reflect.TypeFor[testItem]()
	assert.Equal(t, 2, typ.NumField())
	assert.Equal(t, "ID", typ.Field(0).Name)
	assert.Equal(t, "Name", typ.Field(1).Name)
}

func TestPage_Calculation_Examples(t *testing.T) {
	tests := []struct {
		page   int
		size   int
		offset int
		name   string
	}{
		{1, 10, 0, "first page"},
		{2, 10, 10, "second page"},
		{5, 20, 80, "fifth page with size 20"},
		{10, 50, 450, "tenth page with size 50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Page{Number: tt.page, Size: tt.size}
			assert.Equal(t, tt.offset, p.offset())
		})
	}
}
