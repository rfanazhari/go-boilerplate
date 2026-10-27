package architecture_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type importCase struct {
	name    string
	pattern string
	forbid  []string
}

func TestLayerDependencies(t *testing.T) {
	tests := []importCase{
		{
			name:    "domain does not import outer layers",
			pattern: "github.com/rfanazhari/go-boilerplate/domain/...",
			forbid:  []string{"/application/", "/delivery/", "/infrastructure/", "/cmd/"},
		},
		{
			name:    "application does not import delivery or infrastructure",
			pattern: "github.com/rfanazhari/go-boilerplate/application/...",
			forbid:  []string{"/delivery/", "/infrastructure/", "/cmd/"},
		},
		{
			name:    "delivery does not import infrastructure",
			pattern: "github.com/rfanazhari/go-boilerplate/delivery/...",
			forbid:  []string{"/infrastructure/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \",\"}}", tt.pattern)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			for _, line := range strings.Split(string(out), "\n") {
				for _, banned := range tt.forbid {
					require.NotContains(t, line, banned, line)
				}
			}
		})
	}
}
