package modes

import (
	"path/filepath"
	"testing"

	"github.com/aliancn/logcmd/internal/domain/model"
)

func TestResolveLogDir(t *testing.T) {
	t.Parallel()

	mode := &SearchMode{}
	cases := []struct {
		name string
		path string
		want string
	}{
		{
			name: "project root path",
			path: "/tmp/project",
			want: "/tmp/project/.logcmd",
		},
		{
			name: "already logcmd directory",
			path: "/tmp/project/.logcmd",
			want: "/tmp/project/.logcmd",
		},
		{
			name: "trailing slash",
			path: "/tmp/project/.logcmd/",
			want: "/tmp/project/.logcmd",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mode.resolveLogDir(&model.Project{Path: tc.path})
			if got != filepath.Clean(tc.want) {
				t.Fatalf("resolveLogDir(%s) = %s, want %s", tc.path, got, tc.want)
			}
		})
	}
}
