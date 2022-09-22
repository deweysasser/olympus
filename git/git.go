package git

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"os/exec"
)

type CommitSHA string

func CurrentSHA(dir string) (CommitSHA, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir

	log.Debug().Strs("cmd", cmd.Args).Msg("running")

	bytes, err := cmd.Output()

	if err != nil {
		return "", errors.Wrap(err, "Error getting HEAD commit SHA")
	}

	return CommitSHA(bytes), nil
}
