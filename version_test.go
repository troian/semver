package semver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrictNewVersion(t *testing.T) {
	tests := []struct {
		version string
		err     bool
	}{
		{"1.2.3", false},
		{"1.2.3-alpha.01", true},
		{"1.2.3+test.01", false},
		{"1.2.3-alpha.-1", false},
		{"v1.2.3", true},
		{"1.0", true},
		{"v1.0", true},
		{"1", true},
		{"v1", true},
		{"1.2", true},
		{"1.2.beta", true},
		{"v1.2.beta", true},
		{"foo", true},
		{"1.2-5", true},
		{"v1.2-5", true},
		{"1.2-beta.5", true},
		{"v1.2-beta.5", true},
		{"\n1.2", true},
		{"\nv1.2", true},
		{"1.2.0-x.Y.0+metadata", false},
		{"v1.2.0-x.Y.0+metadata", true},
		{"1.2.0-x.Y.0+metadata-width-hypen", false},
		{"v1.2.0-x.Y.0+metadata-width-hypen", true},
		{"1.2.3-rc1-with-hypen", false},
		{"v1.2.3-rc1-with-hypen", true},
		{"1.2.3.4", true},
		{"v1.2.3.4", true},
		{"1.2.2147483648", false},
		{"1.2147483648.3", false},
		{"2147483648.3.0", false},

		// The SemVer spec in a pre-release expects to allow [0-9A-Za-z-]. But,
		// the lack of all 3 parts in this version should produce an error.
		{"20221209-update-renovatejson-v4", true},

		// Various cases that are invalid semver
		{"1.1.2+.123", true},                             // A leading . in build metadata. This would signify that the first segment is empty
		{"1.0.0-alpha_beta", true},                       // An underscore in the pre-release is an invalid character
		{"1.0.0-alpha..", true},                          // Multiple empty segments
		{"1.0.0-alpha..1", true},                         // Multiple empty segments but one with a value
		{"01.1.1", true},                                 // A leading 0 on a number segment
		{"1.01.1", true},                                 // A leading 0 on a number segment
		{"1.1.01", true},                                 // A leading 0 on a number segment
		{"9.8.7+meta+meta", true},                        // Multiple metadata parts
		{"1.2.31----RC-SNAPSHOT.12.09.1--.12+788", true}, // Leading 0 in a number part of a pre-release segment
		{"1.2.3-0123", true},
		{"1.2.3-0123.0123", true},
		{"+invalid", true},
		{"-invalid", true},
		{"-invalid.01", true},
		{"alpha+beta", true},
		{"1.2.3-alpha_beta+foo", true},
		{"1.0.0-alpha..1", true},
	}

	for _, tc := range tests {
		_, err := StrictNewVersion(tc.version)

		if tc.err {
			require.Error(t, err)
		} else if !tc.err && err != nil {
			require.NoError(t, err)
		}
	}
}

func TestNewVersion(t *testing.T) {
	tests := []struct {
		version string
		err     bool
	}{
		{"1.2.3", false},
		{"1.2.3-alpha.01", true},
		{"1.2.3+test.01", false},
		{"1.2.3-alpha.-1", false},
		{"v1.2.3", false},
		{"1.0", false},
		{"v1.0", false},
		{"1", false},
		{"v1", false},
		{"1.2.beta", true},
		{"v1.2.beta", true},
		{"foo", true},
		{"1.2-5", false},
		{"v1.2-5", false},
		{"1.2-beta.5", false},
		{"v1.2-beta.5", false},
		{"\n1.2", true},
		{"\nv1.2", true},
		{"1.2.0-x.Y.0+metadata", false},
		{"v1.2.0-x.Y.0+metadata", false},
		{"1.2.0-x.Y.0+metadata-width-hypen", false},
		{"v1.2.0-x.Y.0+metadata-width-hypen", false},
		{"1.2.3-rc1-with-hypen", false},
		{"v1.2.3-rc1-with-hypen", false},
		{"1.2.3.4", true},
		{"v1.2.3.4", true},
		{"1.2.2147483648", false},
		{"1.2147483648.3", false},
		{"2147483648.3.0", false},

		// Due to having 4 parts these should produce an error. See
		// https://github.com/Masterminds/semver/issues/185 for the reason for
		// these tests.
		{"12.3.4.1234", true},
		{"12.23.4.1234", true},
		{"12.3.34.1234", true},

		// The SemVer spec in a pre-release expects to allow [0-9A-Za-z-].
		{"20221209-update-renovatejson-v4", false},

		// Various cases that are invalid semver
		{"1.1.2+.123", true},                             // A leading . in build metadata. This would signify that the first segment is empty
		{"1.0.0-alpha_beta", true},                       // An underscore in the pre-release is an invalid character
		{"1.0.0-alpha..", true},                          // Multiple empty segments
		{"1.0.0-alpha..1", true},                         // Multiple empty segments but one with a value
		{"01.1.1", true},                                 // A leading 0 on a number segment
		{"1.01.1", true},                                 // A leading 0 on a number segment
		{"1.1.01", true},                                 // A leading 0 on a number segment
		{"9.8.7+meta+meta", true},                        // Multiple metadata parts
		{"1.2.31----RC-SNAPSHOT.12.09.1--.12+788", true}, // Leading 0 in a number part of a pre-release segment
	}

	for _, tc := range tests {
		_, err := NewVersion(tc.version)
		if tc.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestNew(t *testing.T) {
	// v0.1.2
	v := New(0, 1, 2, "", "")

	require.Equal(t, "0.1.2", v.String())

	// v1.2.3-alpha.1+foo.bar
	v = New(1, 2, 3, "alpha.1", "foo.bar")

	require.Equal(t, "1.2.3-alpha.1+foo.bar", v.String())
}

func TestOriginal(t *testing.T) {
	tests := []string{
		"1.2.3",
		"v1.2.3",
		"1.0",
		"v1.0",
		"1",
		"v1",
		"1.2-5",
		"v1.2-5",
		"1.2-beta.5",
		"v1.2-beta.5",
		"1.2.0-x.Y.0+metadata",
		"v1.2.0-x.Y.0+metadata",
		"1.2.0-x.Y.0+metadata-width-hypen",
		"v1.2.0-x.Y.0+metadata-width-hypen",
		"1.2.3-rc1-with-hypen",
		"v1.2.3-rc1-with-hypen",
	}

	for _, tc := range tests {
		v, err := NewVersion(tc)
		require.NoError(t, err)
		require.Equal(t, tc, v.Original())
	}
}

func TestParts(t *testing.T) {
	v, err := NewVersion("1.2.3-beta.1+build.123")
	require.NoError(t, err)
	require.Equal(t, uint64(1), v.Major())
	require.Equal(t, uint64(2), v.Minor())
	require.Equal(t, uint64(3), v.Patch())
	require.Equal(t, "beta.1", v.Prerelease())
	require.Equal(t, "build.123", v.Metadata())
}

func TestCoerceString(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"1.2.3", "1.2.3"},
		{"v1.2.3", "1.2.3"},
		{"1.0", "1.0.0"},
		{"v1.0", "1.0.0"},
		{"1", "1.0.0"},
		{"v1", "1.0.0"},
		{"1.2-5", "1.2.0-5"},
		{"v1.2-5", "1.2.0-5"},
		{"1.2-beta.5", "1.2.0-beta.5"},
		{"v1.2-beta.5", "1.2.0-beta.5"},
		{"1.2.0-x.Y.0+metadata", "1.2.0-x.Y.0+metadata"},
		{"v1.2.0-x.Y.0+metadata", "1.2.0-x.Y.0+metadata"},
		{"1.2.0-x.Y.0+metadata-width-hypen", "1.2.0-x.Y.0+metadata-width-hypen"},
		{"v1.2.0-x.Y.0+metadata-width-hypen", "1.2.0-x.Y.0+metadata-width-hypen"},
		{"1.2.3-rc1-with-hypen", "1.2.3-rc1-with-hypen"},
		{"v1.2.3-rc1-with-hypen", "1.2.3-rc1-with-hypen"},
	}

	for _, tc := range tests {
		v, err := NewVersion(tc.version)
		require.NoError(t, err)
		require.Equal(t, tc.expected, v.String())
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.2.3", "1.5.1", -1},
		{"2.2.3", "1.5.1", 1},
		{"2.2.3", "2.2.2", 1},
		{"3.2-beta", "3.2-beta", 0},
		{"1.3", "1.1.4", 1},
		{"4.2", "4.2-beta", 1},
		{"4.2-beta", "4.2", -1},
		{"4.2-alpha", "4.2-beta", -1},
		{"4.2-alpha", "4.2-alpha", 0},
		{"4.2-beta.2", "4.2-beta.1", 1},
		{"4.2-beta2", "4.2-beta1", 1},
		{"4.2-beta", "4.2-beta.2", -1},
		{"4.2-beta", "4.2-beta.foo", -1},
		{"4.2-beta.2", "4.2-beta", 1},
		{"4.2-beta.foo", "4.2-beta", 1},
		{"1.2+bar", "1.2+baz", 0},
		{"1.0.0-beta.4", "1.0.0-beta.-2", -1},
		{"1.0.0-beta.-2", "1.0.0-beta.-3", -1},
		{"1.0.0-beta.-3", "1.0.0-beta.5", 1},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		require.NoError(t, err)

		v2, err := NewVersion(tc.v2)
		require.NoError(t, err)
		require.Equal(t, tc.expected, v1.Compare(v2))
	}
}

func TestLessThan(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.2.3", "1.5.1", true},
		{"2.2.3", "1.5.1", false},
		{"3.2-beta", "3.2-beta", false},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		require.NoError(t, err)

		v2, err := NewVersion(tc.v2)
		require.NoError(t, err)

		require.Equal(t, tc.expected, v1.LessThan(v2))
	}
}

func TestLessThanEqual(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.2.3", "1.5.1", true},
		{"2.2.3", "1.5.1", false},
		{"1.5.1", "1.5.1", true},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		require.NoError(t, err)

		v2, err := NewVersion(tc.v2)
		require.NoError(t, err)

		require.Equal(t, tc.expected, v1.LessThanEqual(v2))
	}
}

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.2.3", "1.5.1", false},
		{"2.2.3", "1.5.1", true},
		{"3.2-beta", "3.2-beta", false},
		{"3.2.0-beta.1", "3.2.0-beta.5", false},
		{"3.2-beta.4", "3.2-beta.2", true},
		{"7.43.0-SNAPSHOT.99", "7.43.0-SNAPSHOT.103", false},
		{"7.43.0-SNAPSHOT.FOO", "7.43.0-SNAPSHOT.103", true},
		{"7.43.0-SNAPSHOT.99", "7.43.0-SNAPSHOT.BAR", false},
		{"7.43.0-SNAPSHOT100", "7.43.0-SNAPSHOT99", true},
		{"7.43.0-SNAPSHOT99", "7.43.0-SNAPSHOT100", false},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		require.NoError(t, err)

		v2, err := NewVersion(tc.v2)
		require.NoError(t, err)
		require.Equal(t, tc.expected, v1.GreaterThan(v2))
	}
}

func TestGreaterThanEqual(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected bool
	}{
		{"1.2.3", "1.5.1", false},
		{"2.2.3", "1.5.1", true},
		{"1.5.1", "1.5.1", true},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		require.NoError(t, err)

		v2, err := NewVersion(tc.v2)
		require.NoError(t, err)
		require.Equal(t, tc.expected, v1.GreaterThanEqual(v2))
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		v1       *Version
		v2       *Version
		expected bool
	}{
		{MustParse("1.2.3"), MustParse("1.5.1"), false},
		{MustParse("2.2.3"), MustParse("1.5.1"), false},
		{MustParse("3.2-beta"), MustParse("3.2-beta"), true},
		{MustParse("3.2-beta+foo"), MustParse("3.2-beta+bar"), true},
		{nil, nil, true},
		{nil, MustParse("1.2.3"), false},
		{MustParse("1.2.3"), nil, false},
	}

	for _, tc := range tests {
		require.Equal(t, tc.expected, tc.v1.Equal(tc.v2))
	}
}

func TestInc(t *testing.T) {
	tests := []struct {
		v1               string
		expected         string
		how              string
		expectedOriginal string
	}{
		{"1.2.3", "1.2.4", "patch", "1.2.4"},
		{"v1.2.4", "1.2.5", "patch", "v1.2.5"},
		{"1.2.3", "1.3.0", "minor", "1.3.0"},
		{"v1.2.4", "1.3.0", "minor", "v1.3.0"},
		{"1.2.3", "2.0.0", "major", "2.0.0"},
		{"v1.2.4", "2.0.0", "major", "v2.0.0"},
		{"1.2.3+meta", "1.2.4", "patch", "1.2.4"},
		{"1.2.3-beta+meta", "1.2.3", "patch", "1.2.3"},
		{"v1.2.4-beta+meta", "1.2.4", "patch", "v1.2.4"},
		{"1.2.3-beta+meta", "1.3.0", "minor", "1.3.0"},
		{"v1.2.4-beta+meta", "1.3.0", "minor", "v1.3.0"},
		{"1.2.3-beta+meta", "2.0.0", "major", "2.0.0"},
		{"v1.2.4-beta+meta", "2.0.0", "major", "v2.0.0"},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		require.NoError(t, err)

		var v2 *Version
		switch tc.how {
		case "patch":
			v2 = v1.IncPatch()
		case "minor":
			v2 = v1.IncMinor()
		case "major":
			v2 = v1.IncMajor()
		default:
			t.Fatalf("invalid case")
		}

		require.Equal(t, tc.expected, v2.String())
		require.Equal(t, tc.expectedOriginal, v2.Original())
	}
}

func TestSetPrerelease(t *testing.T) {
	tests := []struct {
		v1                 string
		prerelease         string
		expectedVersion    string
		expectedPrerelease string
		expectedOriginal   string
		expectedErr        error
	}{
		{"1.2.3", "**", "1.2.3", "", "1.2.3", ErrInvalidPrerelease},
		{"1.2.3", "beta", "1.2.3-beta", "beta", "1.2.3-beta", nil},
		{"v1.2.4", "beta", "1.2.4-beta", "beta", "v1.2.4-beta", nil},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		require.NoError(t, err)

		v2, err := v1.SetPrerelease(tc.prerelease)
		require.Equal(t, tc.expectedErr, err)

		require.Equal(t, tc.expectedPrerelease, v2.Prerelease())
		require.Equal(t, tc.expectedVersion, v2.String())
		require.Equal(t, tc.expectedOriginal, v2.Original())
	}
}

func TestSetMetadata(t *testing.T) {
	tests := []struct {
		v1               string
		metadata         string
		expectedVersion  string
		expectedMetadata string
		expectedOriginal string
		expectedErr      error
	}{
		{"1.2.3", "**", "1.2.3", "", "1.2.3", ErrInvalidMetadata},
		{"1.2.3", "meta", "1.2.3+meta", "meta", "1.2.3+meta", nil},
		{"v1.2.4", "meta", "1.2.4+meta", "meta", "v1.2.4+meta", nil},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.v1)
		require.NoError(t, err)

		v2, err := v1.SetMetadata(tc.metadata)
		require.Equal(t, tc.expectedErr, err)

		require.Equal(t, tc.expectedMetadata, v2.Metadata())
		require.Equal(t, tc.expectedVersion, v2.String())
		require.Equal(t, tc.expectedOriginal, v2.Original())
	}
}

func TestOriginalVPrefix(t *testing.T) {
	tests := []struct {
		version string
		vprefix string
	}{
		{"1.2.3", ""},
		{"v1.2.4", "v"},
	}

	for _, tc := range tests {
		v1, err := NewVersion(tc.version)
		require.NoError(t, err)
		require.Equal(t, tc.vprefix, v1.originalVPrefix())
	}
}

func TestJsonMarshal(t *testing.T) {
	sVer := "1.1.1"
	x, err := StrictNewVersion(sVer)
	require.NoError(t, err)

	out, err := json.Marshal(x)
	require.NoError(t, err)

	require.Equal(t, fmt.Sprintf("%q", sVer), string(out))
}

func TestJsonUnmarshal(t *testing.T) {
	sVer := "1.1.1"
	ver := &Version{}
	err := json.Unmarshal([]byte(fmt.Sprintf("%q", sVer)), ver)
	require.NoError(t, err)
	require.Equal(t, sVer, ver.String())
}

func TestTextMarshal(t *testing.T) {
	sVer := "1.1.1"

	x, err := StrictNewVersion(sVer)
	require.NoError(t, err)

	out, err := x.MarshalText()
	require.NoError(t, err)

	require.Equal(t, sVer, string(out))
}

func TestTextUnmarshal(t *testing.T) {
	sVer := "1.1.1"
	ver := &Version{}

	err := ver.UnmarshalText([]byte(sVer))
	require.NoError(t, err)
	require.Equal(t, sVer, ver.String())
}

func TestSQLScanner(t *testing.T) {
	sVer := "1.1.1"
	x, err := StrictNewVersion(sVer)
	require.NoError(t, err)

	var s sql.Scanner = x
	assert.IsType(t, &Version{}, s)
	require.Equal(t, sVer, s.(*Version).String())
}

func TestDriverValuer(t *testing.T) {
	sVer := "1.1.1"
	x, err := StrictNewVersion(sVer)
	require.NoError(t, err)

	got, err := x.Value()
	require.NoError(t, err)

	require.Equal(t, sVer, got)
}

func TestValidatePrerelease(t *testing.T) {
	tests := []struct {
		pre      string
		expected error
	}{
		{"foo", nil},
		{"alpha.1", nil},
		{"alpha.01", ErrSegmentStartsZero},
		{"foo☃︎", ErrInvalidPrerelease},
		{"alpha.0-1", nil},
	}

	for _, tc := range tests {
		err := validatePrerelease(tc.pre)
		require.Equal(t, tc.expected, err)
	}
}

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		meta     string
		expected error
	}{
		{"foo", nil},
		{"alpha.1", nil},
		{"alpha.01", nil},
		{"foo☃︎", ErrInvalidMetadata},
		{"alpha.0-1", nil},
		{"al-pha.1Phe70CgWe050H9K1mJwRUqTNQXZRERwLOEg37wpXUb4JgzgaD5YkL52ABnoyiE", nil},
	}

	for _, tc := range tests {
		err := validateMetadata(tc.meta)
		require.Equal(t, tc.expected, err)
	}
}

func TestPrerelParser(t *testing.T) {
	tests := []struct {
		val string
		tok []string
	}{
		{
			val: "rc0",
			tok: []string{"rc", "0"},
		},
		{
			val: "rc.0",
			tok: []string{"rc", "0"},
		},
	}

	for _, test := range tests {
		res := TokenizePrerel(test.val)
		require.Len(t, res, len(test.tok))
		require.Equal(t, test.tok, res)
	}
}

func FuzzNewVersion(f *testing.F) {
	testcases := []string{"v1.2.3", " ", "......", "1", "1.2.3-beta.1", "1.2.3+foo", "2.3.4-alpha.1+bar", "lorem ipsum"}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(_ *testing.T, a string) {
		_, _ = NewVersion(a)
	})
}

func FuzzStrictNewVersion(f *testing.F) {
	testcases := []string{"v1.2.3", " ", "......", "1", "1.2.3-beta.1", "1.2.3+foo", "2.3.4-alpha.1+bar", "lorem ipsum"}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(_ *testing.T, a string) {
		_, _ = StrictNewVersion(a)
	})
}
