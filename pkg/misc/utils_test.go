/*
 * Copyright 2022 CECTC, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package misc

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cectc/dbpack/pkg/constant"
	"github.com/cectc/dbpack/pkg/errors"
)

var (
	sDate0     = "0000-00-00"
	sDateTime0 = "0000-00-00 00:00:00"
)

func TestRegisterTLSConfig(t *testing.T) {
	err := RegisterTLSConfig("custom", &tls.Config{
		ServerName:         "localhost",
		InsecureSkipVerify: true,
	})
	assert.Equal(t, nil, err)

	cfg1 := GetTLSConfigClone("custom")
	assert.Equal(t, "localhost", cfg1.ServerName)
	assert.Equal(t, true, cfg1.InsecureSkipVerify)

	DeregisterTLSConfig("custom")
	cfg2 := GetTLSConfigClone("custom")
	assert.Nil(t, cfg2)
}

func TestReadBool(t *testing.T) {
	testCases := []struct {
		in          string
		expectValue bool
		expectValid bool
	}{
		{
			in:          "1",
			expectValue: true,
			expectValid: true,
		},
		{
			in:          "true",
			expectValue: true,
			expectValid: true,
		},
		{
			in:          "TRUE",
			expectValue: true,
			expectValid: true,
		},
		{
			in:          "True",
			expectValue: true,
			expectValid: true,
		},
		{
			in:          "0",
			expectValue: false,
			expectValid: true,
		},
		{
			in:          "false",
			expectValue: false,
			expectValid: true,
		},
		{
			in:          "FALSE",
			expectValue: false,
			expectValid: true,
		},
		{
			in:          "False",
			expectValue: false,
			expectValid: true,
		},
		{
			in:          "zero",
			expectValue: false,
			expectValid: false,
		},
	}

	for _, c := range testCases {
		t.Run(c.in, func(t *testing.T) {
			value, valid := ReadBool(c.in)
			assert.Equal(t, c.expectValue, value)
			assert.Equal(t, c.expectValid, valid)
		})
	}
}

func TestLengthEncodedInteger(t *testing.T) {
	var integerTests = []struct {
		num     uint64
		encoded []byte
	}{
		{0x0000000000000000, []byte{0x00}},
		{0x0000000000000012, []byte{0x12}},
		{0x00000000000000fa, []byte{0xfa}},
		{0x0000000000000100, []byte{0xfc, 0x00, 0x01}},
		{0x0000000000001234, []byte{0xfc, 0x34, 0x12}},
		{0x000000000000ffff, []byte{0xfc, 0xff, 0xff}},
		{0x0000000000010000, []byte{0xfd, 0x00, 0x00, 0x01}},
		{0x0000000000123456, []byte{0xfd, 0x56, 0x34, 0x12}},
		{0x0000000000ffffff, []byte{0xfd, 0xff, 0xff, 0xff}},
		{0x0000000001000000, []byte{0xfe, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}},
		{0x123456789abcdef0, []byte{0xfe, 0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12}},
		{0xffffffffffffffff, []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	}

	for _, tst := range integerTests {
		num, isNull, numLen := ReadLengthEncodedInteger(tst.encoded)
		if isNull {
			t.Errorf("%x: expected %d, got NULL", tst.encoded, tst.num)
		}
		if num != tst.num {
			t.Errorf("%x: expected %d, got %d", tst.encoded, tst.num, num)
		}
		if numLen != len(tst.encoded) {
			t.Errorf("%x: expected size %d, got %d", tst.encoded, len(tst.encoded), numLen)
		}
		encoded := AppendLengthEncodedInteger(nil, num)
		if !bytes.Equal(encoded, tst.encoded) {
			t.Errorf("%v: expected %x, got %x", num, tst.encoded, encoded)
		}
	}
}

func TestFormatBinaryDateTime(t *testing.T) {
	rawDate := [11]byte{}
	binary.LittleEndian.PutUint16(rawDate[:2], 1978)   // years
	rawDate[2] = 12                                    // months
	rawDate[3] = 30                                    // days
	rawDate[4] = 15                                    // hours
	rawDate[5] = 46                                    // minutes
	rawDate[6] = 23                                    // seconds
	binary.LittleEndian.PutUint32(rawDate[7:], 987654) // microseconds
	expect := func(expected string, inlen, outlen uint8) {
		actual, _ := FormatBinaryDateTime(rawDate[:inlen], outlen)
		bytes, ok := actual.([]byte)
		if !ok {
			t.Errorf("formatBinaryDateTime must return []byte, was %T", actual)
		}
		if string(bytes) != expected {
			t.Errorf(
				"expected %q, got %q for length in %d, out %d",
				expected, actual, inlen, outlen,
			)
		}
	}
	expect("0000-00-00", 0, 10)
	expect("0000-00-00 00:00:00", 0, 19)
	expect("1978-12-30", 4, 10)
	expect("1978-12-30 15:46:23", 7, 19)
	expect("1978-12-30 15:46:23.987654", 11, 26)
}

func TestFormatBinaryTime(t *testing.T) {
	expect := func(expected string, src []byte, outlen uint8) {
		actual, _ := FormatBinaryTime(src, outlen)
		bytes, ok := actual.([]byte)
		if !ok {
			t.Errorf("formatBinaryDateTime must return []byte, was %T", actual)
		}
		if string(bytes) != expected {
			t.Errorf(
				"expected %q, got %q for src=%q and outlen=%d",
				expected, actual, src, outlen)
		}
	}

	// binary format:
	// sign (0: positive, 1: negative), days(4), hours, minutes, seconds, micro(4)

	// Zeros
	expect("00:00:00", []byte{}, 8)
	expect("00:00:00.0", []byte{}, 10)
	expect("00:00:00.000000", []byte{}, 15)

	// Without micro(4)
	expect("12:34:56", []byte{0, 0, 0, 0, 0, 12, 34, 56}, 8)
	expect("-12:34:56", []byte{1, 0, 0, 0, 0, 12, 34, 56}, 8)
	expect("12:34:56.00", []byte{0, 0, 0, 0, 0, 12, 34, 56}, 11)
	expect("24:34:56", []byte{0, 1, 0, 0, 0, 0, 34, 56}, 8)
	expect("-99:34:56", []byte{1, 4, 0, 0, 0, 3, 34, 56}, 8)
	expect("103079215103:34:56", []byte{0, 255, 255, 255, 255, 23, 34, 56}, 8)

	// With micro(4)
	expect("12:34:56.00", []byte{0, 0, 0, 0, 0, 12, 34, 56, 99, 0, 0, 0}, 11)
	expect("12:34:56.000099", []byte{0, 0, 0, 0, 0, 12, 34, 56, 99, 0, 0, 0}, 15)
}

func TestEscapeBackslash(t *testing.T) {
	expect := func(expected, value string) {
		actual := string(escapeBytesBackslash([]byte{}, []byte(value)))
		if actual != expected {
			t.Errorf(
				"expected %s, got %s",
				expected, actual,
			)
		}

		actual = string(escapeStringBackslash([]byte{}, value))
		if actual != expected {
			t.Errorf(
				"expected %s, got %s",
				expected, actual,
			)
		}
	}

	expect("foo\\0bar", "foo\x00bar")
	expect("foo\\nbar", "foo\nbar")
	expect("foo\\rbar", "foo\rbar")
	expect("foo\\Zbar", "foo\x1abar")
	expect("foo\\\"bar", "foo\"bar")
	expect("foo\\\\bar", "foo\\bar")
	expect("foo\\'bar", "foo'bar")
}

func TestEscapeQuotes(t *testing.T) {
	expect := func(expected, value string) {
		actual := string(escapeBytesQuotes([]byte{}, []byte(value)))
		if actual != expected {
			t.Errorf(
				"expected %s, got %s",
				expected, actual,
			)
		}

		actual = string(escapeStringQuotes([]byte{}, value))
		if actual != expected {
			t.Errorf(
				"expected %s, got %s",
				expected, actual,
			)
		}
	}

	expect("foo\x00bar", "foo\x00bar") // not affected
	expect("foo\nbar", "foo\nbar")     // not affected
	expect("foo\rbar", "foo\rbar")     // not affected
	expect("foo\x1abar", "foo\x1abar") // not affected
	expect("foo''bar", "foo'bar")      // affected
	expect("foo\"bar", "foo\"bar")     // not affected
}

func TestAtomicBool(t *testing.T) {
	var ab atomicBool
	if ab.IsSet() {
		t.Fatal("Expected value to be false")
	}

	ab.Set(true)
	if ab.value != 1 {
		t.Fatal("Set(true) did not set value to 1")
	}
	if !ab.IsSet() {
		t.Fatal("Expected value to be true")
	}

	ab.Set(true)
	if !ab.IsSet() {
		t.Fatal("Expected value to be true")
	}

	ab.Set(false)
	if ab.value != 0 {
		t.Fatal("Set(false) did not set value to 0")
	}
	if ab.IsSet() {
		t.Fatal("Expected value to be false")
	}

	ab.Set(false)
	if ab.IsSet() {
		t.Fatal("Expected value to be false")
	}
	if ab.TrySet(false) {
		t.Fatal("Expected TrySet(false) to fail")
	}
	if !ab.TrySet(true) {
		t.Fatal("Expected TrySet(true) to succeed")
	}
	if !ab.IsSet() {
		t.Fatal("Expected value to be true")
	}

	ab.Set(true)
	if !ab.IsSet() {
		t.Fatal("Expected value to be true")
	}
	if ab.TrySet(true) {
		t.Fatal("Expected TrySet(true) to fail")
	}
	if !ab.TrySet(false) {
		t.Fatal("Expected TrySet(false) to succeed")
	}
	if ab.IsSet() {
		t.Fatal("Expected value to be false")
	}

	ab._noCopy.Lock() // we've "tested" it ¯\_(ツ)_/¯
}

func TestAtomicError(t *testing.T) {
	var ae atomicError
	if ae.Value() != nil {
		t.Fatal("Expected value to be nil")
	}

	ae.Set(errors.ErrMalformedPkt)
	if v := ae.Value(); v != errors.ErrMalformedPkt {
		if v == nil {
			t.Fatal("Value is still nil")
		}
		t.Fatal("Error did not match")
	}
	ae.Set(errors.ErrPktSync)
	if ae.Value() == errors.ErrMalformedPkt {
		t.Fatal("Error still matches old error")
	}
	if v := ae.Value(); v != errors.ErrPktSync {
		t.Fatal("Error did not match")
	}
}

func TestIsolationLevelMapping(t *testing.T) {
	data := []struct {
		level    driver.IsolationLevel
		expected string
	}{
		{
			level:    driver.IsolationLevel(sql.LevelReadCommitted),
			expected: "READ COMMITTED",
		},
		{
			level:    driver.IsolationLevel(sql.LevelRepeatableRead),
			expected: "REPEATABLE READ",
		},
		{
			level:    driver.IsolationLevel(sql.LevelReadUncommitted),
			expected: "READ UNCOMMITTED",
		},
		{
			level:    driver.IsolationLevel(sql.LevelSerializable),
			expected: "SERIALIZABLE",
		},
	}

	for i, td := range data {
		if actual, err := mapIsolationLevel(td.level); actual != td.expected || err != nil {
			t.Fatal(i, td.expected, actual, err)
		}
	}

	// check unsupported mapping
	expectedErr := "mysql: unsupported isolation level: 7"
	actual, err := mapIsolationLevel(driver.IsolationLevel(sql.LevelLinearizable))
	if actual != "" || err == nil {
		t.Fatal("Expected error on unsupported isolation level")
	}
	if err.Error() != expectedErr {
		t.Fatalf("Expected error to be %q, got %q", expectedErr, err)
	}
}

func TestAppendDateTime(t *testing.T) {
	tests := []struct {
		t   time.Time
		str string
	}{
		{
			t:   time.Date(1234, 5, 6, 0, 0, 0, 0, time.UTC),
			str: "1234-05-06",
		},
		{
			t:   time.Date(4567, 12, 31, 12, 0, 0, 0, time.UTC),
			str: "4567-12-31 12:00:00",
		},
		{
			t:   time.Date(2020, 5, 30, 12, 34, 0, 0, time.UTC),
			str: "2020-05-30 12:34:00",
		},
		{
			t:   time.Date(2020, 5, 30, 12, 34, 56, 0, time.UTC),
			str: "2020-05-30 12:34:56",
		},
		{
			t:   time.Date(2020, 5, 30, 22, 33, 44, 123000000, time.UTC),
			str: "2020-05-30 22:33:44.123",
		},
		{
			t:   time.Date(2020, 5, 30, 22, 33, 44, 123456000, time.UTC),
			str: "2020-05-30 22:33:44.123456",
		},
		{
			t:   time.Date(2020, 5, 30, 22, 33, 44, 123456789, time.UTC),
			str: "2020-05-30 22:33:44.123456789",
		},
		{
			t:   time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC),
			str: "9999-12-31 23:59:59.999999999",
		},
		{
			t:   time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
			str: "0001-01-01",
		},
	}
	for _, v := range tests {
		buf := make([]byte, 0, 32)
		buf, _ = AppendDateTime(buf, v.t)
		if str := string(buf); str != v.str {
			t.Errorf("appendDateTime(%v), have: %s, want: %s", v.t, str, v.str)
		}
	}

	// year out of range
	{
		v := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
		buf := make([]byte, 0, 32)
		_, err := AppendDateTime(buf, v)
		if err == nil {
			t.Error("want an error")
			return
		}
	}
	{
		v := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		buf := make([]byte, 0, 32)
		_, err := AppendDateTime(buf, v)
		if err == nil {
			t.Error("want an error")
			return
		}
	}
}

func TestParseDateTime(t *testing.T) {
	cases := []struct {
		name string
		str  string
	}{
		{
			name: "parse date",
			str:  "2020-05-13",
		},
		{
			name: "parse null date",
			str:  sDate0,
		},
		{
			name: "parse datetime",
			str:  "2020-05-13 21:30:45",
		},
		{
			name: "parse null datetime",
			str:  sDateTime0,
		},
		{
			name: "parse datetime nanosec 1-digit",
			str:  "2020-05-25 23:22:01.1",
		},
		{
			name: "parse datetime nanosec 2-digits",
			str:  "2020-05-25 23:22:01.15",
		},
		{
			name: "parse datetime nanosec 3-digits",
			str:  "2020-05-25 23:22:01.159",
		},
		{
			name: "parse datetime nanosec 4-digits",
			str:  "2020-05-25 23:22:01.1594",
		},
		{
			name: "parse datetime nanosec 5-digits",
			str:  "2020-05-25 23:22:01.15949",
		},
		{
			name: "parse datetime nanosec 6-digits",
			str:  "2020-05-25 23:22:01.159491",
		},
	}

	for _, loc := range []*time.Location{
		time.UTC,
		time.FixedZone("test", 8*60*60),
	} {
		for _, cc := range cases {
			t.Run(cc.name+"-"+loc.String(), func(t *testing.T) {
				var want time.Time
				if cc.str != sDate0 && cc.str != sDateTime0 {
					var err error
					want, err = time.ParseInLocation(constant.TimeFormat[:len(cc.str)], cc.str, loc)
					if err != nil {
						t.Fatal(err)
					}
				}
				got, err := ParseDateTime([]byte(cc.str), loc)
				if err != nil {
					t.Fatal(err)
				}

				if !want.Equal(got) {
					t.Fatalf("want: %v, but got %v", want, got)
				}
			})
		}
	}
}

func TestParseDateTimeFail(t *testing.T) {
	cases := []struct {
		name    string
		str     string
		wantErr string
	}{
		{
			name:    "parse invalid time",
			str:     "hello",
			wantErr: "invalid time bytes: hello",
		},
		{
			name:    "parse year",
			str:     "000!-00-00 00:00:00.000000",
			wantErr: "not [0-9]",
		},
		{
			name:    "parse month",
			str:     "0000-!0-00 00:00:00.000000",
			wantErr: "not [0-9]",
		},
		{
			name:    `parse "-" after parsed year`,
			str:     "0000:00-00 00:00:00.000000",
			wantErr: "bad value for field: `:`",
		},
		{
			name:    `parse "-" after parsed month`,
			str:     "0000-00:00 00:00:00.000000",
			wantErr: "bad value for field: `:`",
		},
		{
			name:    `parse " " after parsed date`,
			str:     "0000-00-00+00:00:00.000000",
			wantErr: "bad value for field: `+`",
		},
		{
			name:    `parse ":" after parsed date`,
			str:     "0000-00-00 00-00:00.000000",
			wantErr: "bad value for field: `-`",
		},
		{
			name:    `parse ":" after parsed hour`,
			str:     "0000-00-00 00:00-00.000000",
			wantErr: "bad value for field: `-`",
		},
		{
			name:    `parse "." after parsed sec`,
			str:     "0000-00-00 00:00:00?000000",
			wantErr: "bad value for field: `?`",
		},
	}

	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			got, err := ParseDateTime([]byte(cc.str), time.UTC)
			if err == nil {
				t.Fatal("want error")
			}
			if cc.wantErr != err.Error() {
				t.Fatalf("want `%s`, but got `%s`", cc.wantErr, err)
			}
			if !got.IsZero() {
				t.Fatal("want zero time")
			}
		})
	}
}
