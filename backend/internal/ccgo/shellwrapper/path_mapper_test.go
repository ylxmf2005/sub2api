package shellwrapper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathMapperMapsCwdInsideProjection(t *testing.T) {
	mapper, err := NewPathMapper("/srv/ccgo/workspaces/u1/project", "/Users/alice/project")
	require.NoError(t, err)

	got, err := mapper.MapCwd("/srv/ccgo/workspaces/u1/project/packages/api")
	require.NoError(t, err)
	require.Equal(t, "/Users/alice/project/packages/api", got)
}

func TestPathMapperRejectsCwdOutsideProjection(t *testing.T) {
	mapper, err := NewPathMapper("/srv/ccgo/workspaces/u1/project", "/Users/alice/project")
	require.NoError(t, err)

	_, err = mapper.MapCwd("/srv/ccgo/workspaces/u1/other")
	require.ErrorContains(t, err, "outside projection root")
}

func TestPathMapperMapsServerRootOccurrencesInCommands(t *testing.T) {
	mapper, err := NewPathMapper("/srv/ccgo/workspaces/u1/project", "/Users/alice/My Project")
	require.NoError(t, err)

	got := mapper.MapCommand(`printf "%s" "/srv/ccgo/workspaces/u1/project/package.json" && cat /srv/ccgo/workspaces/u1/project/src/main.go`)
	require.Equal(t, `printf "%s" "/Users/alice/My Project/package.json" && cat '/Users/alice/My Project/src/main.go'`, got)
}

func TestPathMapperMapsBareServerRootToLocalRoot(t *testing.T) {
	mapper, err := NewPathMapper("/srv/ccgo/workspaces/u1/project", "/Users/alice/My Project")
	require.NoError(t, err)

	got := mapper.MapCommand(`ls /srv/ccgo/workspaces/u1/project`)
	require.Equal(t, `ls '/Users/alice/My Project'`, got)
}

func TestPathMapperDoesNotMapSimilarPrefix(t *testing.T) {
	mapper, err := NewPathMapper("/srv/ccgo/workspaces/u1/project", "/Users/alice/project")
	require.NoError(t, err)

	got := mapper.MapCommand(`cat /srv/ccgo/workspaces/u1/project-old/file.txt`)
	require.Equal(t, `cat /srv/ccgo/workspaces/u1/project-old/file.txt`, got)
}

func TestPathMapperSupportsWindowsLocalRootForDisplayMapping(t *testing.T) {
	mapper, err := NewPathMapper("/srv/ccgo/workspaces/u1/project", `C:\Users\alice\project`)
	require.NoError(t, err)

	got, err := mapper.MapCwd("/srv/ccgo/workspaces/u1/project/src")
	require.NoError(t, err)
	require.Equal(t, `C:\Users\alice\project\src`, got)
}
