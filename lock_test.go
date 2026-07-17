package xdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLockBuilder_ForUpdate(t *testing.T) {
	lb := ForUpdate()
	assert.Equal(t, "FOR UPDATE", lb.String())
}

func TestLockBuilder_ForNoKeyUpdate(t *testing.T) {
	lb := ForNoKeyUpdate()
	assert.Equal(t, "FOR NO KEY UPDATE", lb.String())
}

func TestLockBuilder_ForShare(t *testing.T) {
	lb := ForShare()
	assert.Equal(t, "FOR SHARE", lb.String())
}

func TestLockBuilder_SkipLocked(t *testing.T) {
	lb := ForUpdate().SkipLocked()
	assert.Equal(t, "FOR UPDATE SKIP LOCKED", lb.String())
}

func TestLockBuilder_NoWait(t *testing.T) {
	lb := ForUpdate().NoWait()
	assert.Equal(t, "FOR UPDATE NOWAIT", lb.String())
}

func TestLockBuilder_Of_Single(t *testing.T) {
	lb := ForUpdate().Of("orders")
	assert.Equal(t, "FOR UPDATE OF orders", lb.String())
}

func TestLockBuilder_Of_Multiple(t *testing.T) {
	lb := ForUpdate().Of("orders", "line_items")
	assert.Equal(t, "FOR UPDATE OF orders, line_items", lb.String())
}

func TestLockBuilder_Of_SkipLocked(t *testing.T) {
	lb := ForUpdate().Of("orders").SkipLocked()
	assert.Equal(t, "FOR UPDATE OF orders SKIP LOCKED", lb.String())
}

func TestLockBuilder_AllStrengthsImplementStringer(t *testing.T) {
	assert.Equal(t, "FOR UPDATE", ForUpdate().String())
	assert.Equal(t, "FOR NO KEY UPDATE", ForNoKeyUpdate().String())
	assert.Equal(t, "FOR SHARE", ForShare().String())
}

func TestLockBuilder_Immutability(t *testing.T) {
	base := ForUpdate()
	withSkip := base.SkipLocked()
	withNoWait := base.NoWait()

	assert.Equal(t, "FOR UPDATE", base.String())
	assert.Equal(t, "FOR UPDATE SKIP LOCKED", withSkip.String())
	assert.Equal(t, "FOR UPDATE NOWAIT", withNoWait.String())
}

func TestSelectBuilder_Lock(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "orders"}}
	sb2 := sb.Lock(ForUpdate().SkipLocked())
	sql, _, err := sb2.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "FOR UPDATE SKIP LOCKED")
}

func TestSelectBuilder_Lock_Of(t *testing.T) {
	sb := SelectBuilder{data: selectData{columns: []string{"*"}, from: "orders"}}
	sb2 := sb.Lock(ForShare().Of("orders"))
	sql, _, err := sb2.ToSQL()
	assert.NoError(t, err)
	assert.Contains(t, sql, "FOR SHARE OF orders")
}

func TestFormatIdentList_Empty(t *testing.T) {
	assert.Equal(t, "", formatIdentList(nil))
	assert.Equal(t, "", formatIdentList([]string{}))
}

func TestFormatIdentList_Single(t *testing.T) {
	assert.Equal(t, "orders", formatIdentList([]string{"orders"}))
}

func TestFormatIdentList_Multiple(t *testing.T) {
	assert.Equal(t, "orders, items", formatIdentList([]string{"orders", "items"}))
}

func TestLockBuilder_StringerInterface(t *testing.T) {
	var s interface{} = ForUpdate()
	_, ok := s.(interface{ String() string })
	assert.True(t, ok, "LockBuilder should implement fmt.Stringer")
}
