//go:build !js && !appengine

package runewidth

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sort"
	"testing"
	"unicode/utf8"
)

var _ sort.Interface = (*table)(nil) // ensure that type "table" does implement sort.Interface

func init() {
	os.Setenv("RUNEWIDTH_EASTASIAN", "")
	handleEnv()
}

func (t table) Len() int {
	return len(t)
}

func (t table) Less(i, j int) bool {
	return t[i].first < t[j].first
}

func (t *table) Swap(i, j int) {
	(*t)[i], (*t)[j] = (*t)[j], (*t)[i]
}

type tableInfo struct {
	tbl     table
	name    string
	wantN   int
	wantSHA string
}

var tables = []tableInfo{
	{private, "private", 137468, "a4a641206dc8c5de80bd9f03515a54a706a5a4904c7684dc6a33d65c967a51b2"},
	{nonprint, "nonprint", 2143, "288904683eb225e7c4c0bd3ee481b53e8dace404ec31d443afdbc4d13729fe95"},
	{combining, "combining", 555, "bf1cafd5aa2c3734b07a609ffd4d981cd3184e322a1b261431ff746031305cb4"},
	{doublewidth, "doublewidth", 182521, "88f214dc0a0c31eb2bc083d1e4b3ad58f720634c6708be8b61f10446a8967b37"},
	{ambiguous, "ambiguous", 138739, "d05e339a10f296de6547ff3d6c5aee32f627f6555477afebd4a3b7e3cf74c9e3"},
	{emoji, "emoji", 3535, "9ec17351601d49c535658de8d129c1d0ccda2e620669fc39a2faaee7dedcef6d"},
	{narrow, "narrow", 111, "fa897699c5e3cd9141c638d539331b0bdd508b874e22996c5e929767d455fc5a"},
	{neutral, "neutral", 28382, "1cbccfec7db52c7bd0e6c97c26229278a221b68afc0ca7830f1ba7e86c9b6dbc"},
}

func TestTableChecksums(t *testing.T) {
	for _, ti := range tables {
		gotN := 0
		buf := make([]byte, utf8.MaxRune+1)
		for r := rune(0); r <= utf8.MaxRune; r++ {
			if inTable(r, ti.tbl) {
				gotN++
				buf[r] = 1
			}
		}
		gotSHA := fmt.Sprintf("%x", sha256.Sum256(buf))
		if gotN != ti.wantN || gotSHA != ti.wantSHA {
			t.Errorf("\ntable = %s\nn = %d want %d\nsha256 %s\nwant   %s",
				ti.name, gotN, ti.wantN, gotSHA, ti.wantSHA)
		}
	}
}

func TestRuneWidthChecksums(t *testing.T) {
	var testcases = []struct {
		name           string
		eastAsianWidth bool
		wantSHA        string
	}{
		{"ea-no", false, "a98d2a32d1b3407a3037636a279a73c3d549f6a9fbc8e92bee91dd991acdf0e1"},
		{"ea-yes", true, "cac3940e576bfd67d8312b762ddee862caf388d30a137359a8d9b07ba09166de"},
	}

	for _, testcase := range testcases {
		c := NewCondition()
		c.EastAsianWidth = testcase.eastAsianWidth
		buf := make([]byte, utf8.MaxRune+1)
		for r := rune(0); r <= utf8.MaxRune; r++ {
			buf[r] = byte(c.RuneWidth(r))
		}
		gotSHA := fmt.Sprintf("%x", sha256.Sum256(buf))
		if gotSHA != testcase.wantSHA {
			t.Errorf("\nTestRuneWidthChecksums = %s\nsha256 %s\nwant   %s",
				testcase.name, gotSHA, testcase.wantSHA)
		}

		// Test with LUT
		c.CreateLUT()
		for r := rune(0); r <= utf8.MaxRune; r++ {
			buf[r] = byte(c.RuneWidth(r))
		}
		gotSHA = fmt.Sprintf("%x", sha256.Sum256(buf))
		if gotSHA != testcase.wantSHA {
			t.Errorf("\nTestRuneWidthChecksums = %s\nsha256 %s\nwant   %s",
				testcase.name, gotSHA, testcase.wantSHA)
		}
	}
}

func TestDefaultLUT(t *testing.T) {
	var testcases = []struct {
		name           string
		eastAsianWidth bool
		wantSHA        string
	}{
		{"ea-no", false, "a98d2a32d1b3407a3037636a279a73c3d549f6a9fbc8e92bee91dd991acdf0e1"},
		{"ea-yes", true, "cac3940e576bfd67d8312b762ddee862caf388d30a137359a8d9b07ba09166de"},
	}

	old := os.Getenv("RUNEWIDTH_EASTASIAN")
	defer os.Setenv("RUNEWIDTH_EASTASIAN", old)

	CreateLUT()
	for _, testcase := range testcases {
		c := DefaultCondition

		if testcase.eastAsianWidth {
			os.Setenv("RUNEWIDTH_EASTASIAN", "1")
		} else {
			os.Setenv("RUNEWIDTH_EASTASIAN", "0")
		}
		handleEnv()

		buf := make([]byte, utf8.MaxRune+1)
		for r := rune(0); r <= utf8.MaxRune; r++ {
			buf[r] = byte(c.RuneWidth(r))
		}
		gotSHA := fmt.Sprintf("%x", sha256.Sum256(buf))
		if gotSHA != testcase.wantSHA {
			t.Errorf("\nTestRuneWidthChecksums = %s,\nsha256 %s\nwant   %s",
				testcase.name, gotSHA, testcase.wantSHA)
		}
	}
	// Remove for other tests.
	DefaultCondition.combinedLut = nil
}

func checkInterval(first, last rune) bool {
	return first >= 0 && first <= utf8.MaxRune &&
		last >= 0 && last <= utf8.MaxRune &&
		first <= last
}

func isCompact(t *testing.T, ti *tableInfo) bool {
	tbl := ti.tbl
	for i := range tbl {
		e := tbl[i]
		if !checkInterval(e.first, e.last) { // sanity check
			t.Errorf("table invalid: table = %s index = %d %v", ti.name, i, e)
			return false
		}
		if i+1 < len(tbl) && e.last+1 >= tbl[i+1].first { // can be combined into one entry
			t.Errorf("table not compact: table = %s index = %d %v %v", ti.name, i, e, tbl[i+1])
			return false
		}
	}
	return true
}

func TestSorted(t *testing.T) {
	for _, ti := range tables {
		if !sort.IsSorted(&ti.tbl) {
			t.Errorf("table not sorted: %s", ti.name)
		}
		if !isCompact(t, &ti) {
			t.Errorf("table not compact: %s", ti.name)
		}
	}
}

var runewidthtests = []struct {
	in     rune
	out    int
	eaout  int
	nseout int
}{
	{'世', 2, 2, 2},
	{'界', 2, 2, 2},
	{'ｾ', 1, 1, 1},
	{'ｶ', 1, 1, 1},
	{'ｲ', 1, 1, 1},
	{'☆', 1, 2, 2}, // double width in ambiguous
	{'☺', 1, 1, 2},
	{'☻', 1, 1, 2},
	{'♥', 1, 2, 2},
	{'♦', 1, 1, 2},
	{'♣', 1, 2, 2},
	{'♠', 1, 2, 2},
	{'♂', 1, 2, 2},
	{'♀', 1, 2, 2},
	{'♪', 1, 2, 2},
	{'♫', 1, 1, 2},
	{'☼', 1, 1, 2},
	{'↕', 1, 2, 2},
	{'‼', 1, 1, 2},
	{'↔', 1, 2, 2},
	{'\x00', 0, 0, 0},
	{'\x01', 0, 0, 0},
	{'\u0300', 0, 0, 0},
	{'\u2028', 0, 0, 0},
	{'\u2029', 0, 0, 0},
	{'a', 1, 1, 1}, // ASCII classified as "na" (narrow)
	{'⟦', 1, 1, 1}, // non-ASCII classified as "na" (narrow)
	{'👁', 1, 1, 2},
}

func TestRuneWidth(t *testing.T) {
	c := NewCondition()
	c.EastAsianWidth = false
	for _, tt := range runewidthtests {
		if out := c.RuneWidth(tt.in); out != tt.out {
			t.Errorf("RuneWidth(%q) = %d, want %d (EastAsianWidth=false)", tt.in, out, tt.out)
		}
	}
	c.EastAsianWidth = true
	for _, tt := range runewidthtests {
		if out := c.RuneWidth(tt.in); out != tt.eaout {
			t.Errorf("RuneWidth(%q) = %d, want %d (EastAsianWidth=true)", tt.in, out, tt.eaout)
		}
	}
	c.StrictEmojiNeutral = false
	for _, tt := range runewidthtests {
		if out := c.RuneWidth(tt.in); out != tt.nseout {
			t.Errorf("RuneWidth(%q) = %d, want %d (StrictEmojiNeutral=false)", tt.in, out, tt.eaout)
		}
	}
}

var isambiguouswidthtests = []struct {
	in  rune
	out bool
}{
	{'世', false},
	{'■', true},
	{'界', false},
	{'○', true},
	{'㈱', false},
	{'①', true},
	{'②', true},
	{'③', true},
	{'④', true},
	{'⑤', true},
	{'⑥', true},
	{'⑦', true},
	{'⑧', true},
	{'⑨', true},
	{'⑩', true},
	{'⑪', true},
	{'⑫', true},
	{'⑬', true},
	{'⑭', true},
	{'⑮', true},
	{'⑯', true},
	{'⑰', true},
	{'⑱', true},
	{'⑲', true},
	{'⑳', true},
	{'☆', true},
}

func TestIsAmbiguousWidth(t *testing.T) {
	for _, tt := range isambiguouswidthtests {
		if out := IsAmbiguousWidth(tt.in); out != tt.out {
			t.Errorf("IsAmbiguousWidth(%q) = %v, want %v", tt.in, out, tt.out)
		}
	}
}

var isneutralwidthtests = []struct {
	in  rune
	out bool
}{
	{'→', false},
	{'┊', false},
	{'┈', false},
	{'～', false},
	{'└', false},
	{'⣀', true},
	{'⣀', true},
}

func TestIsNeutralWidth(t *testing.T) {
	for _, tt := range isneutralwidthtests {
		if out := IsNeutralWidth(tt.in); out != tt.out {
			t.Errorf("IsNeutralWidth(%q) = %v, want %v", tt.in, out, tt.out)
		}
	}
}

func TestEnv(t *testing.T) {
	old := os.Getenv("RUNEWIDTH_EASTASIAN")
	defer os.Setenv("RUNEWIDTH_EASTASIAN", old)

	os.Setenv("RUNEWIDTH_EASTASIAN", "0")
	handleEnv()

	if w := RuneWidth('│'); w != 1 {
		t.Errorf("RuneWidth('│') = %d, want %d", w, 1)
	}
}
