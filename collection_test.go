package semver

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollection(t *testing.T) {
	raw := []string{
		"1.2.3",
		"1.0",
		"1.3",
		"2",
		"0.4.2",
	}

	vs := make([]*Version, len(raw))
	for i, r := range raw {
		v, err := NewVersion(r)
		require.NoError(t, err)

		vs[i] = v
	}

	sort.Sort(Collection(vs))

	e := []string{
		"0.4.2",
		"1.0.0",
		"1.2.3",
		"1.3.0",
		"2.0.0",
	}

	a := make([]string, len(vs))
	for i, v := range vs {
		a[i] = v.String()
	}

	require.Equal(t, e, a)
}
