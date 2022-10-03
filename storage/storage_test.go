package storage

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/deweysasser/olympus/git"
	"github.com/deweysasser/olympus/run"
	"github.com/deweysasser/olympus/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestStorage_buildFile(t *testing.T) {
	tm, err := time.Parse(timeFormat, "2000-01-02-03-04-05")
	require.NoError(t, err)
	tests := []struct {
		name                   string
		key, branch, workspace string
		want                   string
	}{
		{name: "simple", key: "test/one/two/three", branch: "foo", workspace: "bar", want: "/test/one/two/three/2000-01-02-03-04-05__foo__bar.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Storage{
				dir: "/",
			}
			key := ParseKey(tt.key)
			if got := s.buildFile(key,
				&run.PlanRecord{End: tm, Workspace: terraform.Workspace(tt.workspace), Branch: git.Branch(tt.branch)}); got != tt.want {
				t.Errorf("buildFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name string
		args string
		want Key
	}{
		{name: "1 level", args: "foo", want: []string{"foo"}},
		{name: "2 levels", args: "foo/bar", want: []string{"foo", "bar"}},
		{name: "3 levels", args: "foo/bar/baz", want: []string{"foo", "bar", "baz"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseKey(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore(t *testing.T) {
	if _, err := os.Stat("./test-output"); err == nil {
		err = os.RemoveAll("./test-output")
		require.NoError(t, err)
	}

	storage := New("./test-output")
	tm, err := time.Parse(timeFormat, "2000-01-02-03-04-05")
	require.NoError(t, err)

	err = storage.Store(ParseKey("A/1/b"), &run.PlanRecord{End: tm, Branch: "foo", Workspace: "default"})
	assert.NoError(t, err)
	_, err = os.Stat("./test-output/A/1/b/2000-01-02-03-04-05__foo__default.json")
	assert.NoError(t, err)

	assert.Equal(t, "foo", setAsString(storage.Branches()))
	assert.Equal(t, "default", setAsString(storage.Workspaces()))

	err = storage.Store(ParseKey("A/1/c"), &run.PlanRecord{End: tm, Branch: "foo", Workspace: "default"})
	assert.NoError(t, err)
	_, err = os.Stat("./test-output/A/1/c/2000-01-02-03-04-05__foo__default.json")
	assert.NoError(t, err)

	assert.Equal(t, "foo", setAsString(storage.Branches()))
	assert.Equal(t, "default", setAsString(storage.Workspaces()))

	err = storage.Store(ParseKey("A/1/b"), &run.PlanRecord{End: tm, Branch: "baz", Workspace: "default"})
	assert.NoError(t, err)
	_, err = os.Stat("./test-output/A/1/c/2000-01-02-03-04-05__foo__default.json")
	assert.NoError(t, err)

	assert.Equal(t, "baz,foo", setAsString(storage.Branches()))
	assert.Equal(t, "default", setAsString(storage.Workspaces()))

	// Now a new storage should get the same result

	storage2 := New("./test-output")

	assert.Equal(t, "baz,foo", setAsString(storage2.Branches()))
	assert.Equal(t, "default", setAsString(storage2.Workspaces()))

	runs, err := storage2.Summary(ParseKey("A"), git.Branch("foo"), terraform.Workspace("default"))
	assert.NoError(t, err)
	assert.Equal(t, "A", runs.Name)
	assert.Equal(t, 1, len(runs.Records))
	assert.Equal(t, 2, len(runs.Records[0].Records))

	runs, err = storage2.Summary(ParseKey("A/1"), git.Branch("foo"), terraform.Workspace("default"))
	assert.NoError(t, err)

}

func setAsString[T comparable](set mapset.Set[T]) string {
	var s []string

	set.Each(func(t T) bool {

		s = append(s, fmt.Sprint(t))
		return false
	})

	sort.Strings(s)
	return strings.Join(s, ",")

}
