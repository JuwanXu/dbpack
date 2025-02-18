package misc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatTimeMillis(t *testing.T) {
	cases := map[string]struct {
		in  uint64
		out string
	}{
		"1": {4, "1970-01-01 00:00:00"},
		"2": {1844674, "1970-01-01 00:30:44"},
		"3": {18446744073709551615, "1969-12-31 23:59:59"},
		"4": {1653055405, "1970-01-20 03:10:55"},
		"5": {45408320589235234, "1730-04-30 13:42:42"},
		"6": {123456, "1970-01-01 00:02:03"},
	}

	for caseTitle, tc := range cases {
		t.Run(caseTitle, func(t *testing.T) {
			result := FormatTimeMillis(tc.in)
			assert.Equal(t, tc.out, result)
		})
	}
}

func TestFormatDate(t *testing.T) {
	cases := map[string]struct {
		in  uint64
		out string
	}{
		"1": {4, "1970-01-01"},
		"2": {1844674, "1970-01-01"},
		"3": {18446744073709551615, "1969-12-31"},
		"4": {1653055405, "1970-01-20"},
		"5": {45408320589235234, "1730-04-30"},
		"6": {123456, "1970-01-01"},
	}

	for caseTitle, tc := range cases {
		t.Run(caseTitle, func(t *testing.T) {
			result := FormatDate(tc.in)
			assert.Equal(t, tc.out, result)
		})
	}
}
