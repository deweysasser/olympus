package git

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"os/exec"
)

type SHA256 string
type Repo string
type Branch string
type WorkingDir string

func CurrentSHA(dir string) (SHA256, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir

	log.Debug().Strs("cmd", cmd.Args).Msg("running")

	bytes, err := cmd.Output()

	if err != nil {
		return "", errors.Wrap(err, "Error getting HEAD commit SHA")
	}

	return SHA256(bytes), nil
}
