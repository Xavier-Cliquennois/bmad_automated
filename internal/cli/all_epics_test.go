package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEpicFilter_Empty(t *testing.T) {
	got, err := parseEpicFilter("")
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = parseEpicFilter("   ")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestParseEpicFilter_SingleNumber(t *testing.T) {
	got, err := parseEpicFilter("9")
	require.NoError(t, err)
	assert.Equal(t, map[int]struct{}{9: {}}, got)
}

func TestParseEpicFilter_Range(t *testing.T) {
	got, err := parseEpicFilter("9-13")
	require.NoError(t, err)
	assert.Equal(t, map[int]struct{}{9: {}, 10: {}, 11: {}, 12: {}, 13: {}}, got)
}

func TestParseEpicFilter_List(t *testing.T) {
	got, err := parseEpicFilter("9,11,13")
	require.NoError(t, err)
	assert.Equal(t, map[int]struct{}{9: {}, 11: {}, 13: {}}, got)
}

func TestParseEpicFilter_RangeAndList(t *testing.T) {
	got, err := parseEpicFilter("9-11,15")
	require.NoError(t, err)
	assert.Equal(t, map[int]struct{}{9: {}, 10: {}, 11: {}, 15: {}}, got)
}

func TestParseEpicFilter_ToleratesWhitespace(t *testing.T) {
	got, err := parseEpicFilter(" 9 - 11 , 15 ")
	require.NoError(t, err)
	assert.Equal(t, map[int]struct{}{9: {}, 10: {}, 11: {}, 15: {}}, got)
}

func TestParseEpicFilter_InvalidRange(t *testing.T) {
	_, err := parseEpicFilter("13-9")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start > end")
}

func TestParseEpicFilter_InvalidToken(t *testing.T) {
	_, err := parseEpicFilter("foo")
	require.Error(t, err)

	_, err = parseEpicFilter("9-bar")
	require.Error(t, err)
}

func TestFilterEpics_KeepsRequestedAndPreservesOrder(t *testing.T) {
	epicIDs := []string{"3", "9", "10", "11", "12", "13", "14"}
	want := map[int]struct{}{9: {}, 10: {}, 11: {}, 12: {}, 13: {}}

	kept, missing := filterEpics(epicIDs, want)
	assert.Equal(t, []string{"9", "10", "11", "12", "13"}, kept)
	assert.Empty(t, missing)
}

func TestFilterEpics_ReportsMissing(t *testing.T) {
	epicIDs := []string{"3", "9", "10"}
	want := map[int]struct{}{9: {}, 10: {}, 11: {}, 13: {}}

	kept, missing := filterEpics(epicIDs, want)
	assert.Equal(t, []string{"9", "10"}, kept)
	assert.Equal(t, []int{11, 13}, missing)
}

func TestFilterEpics_NoOverlap(t *testing.T) {
	epicIDs := []string{"3", "4"}
	want := map[int]struct{}{9: {}, 10: {}}

	kept, missing := filterEpics(epicIDs, want)
	assert.Empty(t, kept)
	assert.Equal(t, []int{9, 10}, missing)
}
