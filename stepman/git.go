package stepman

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/go-pathutil/pathutil"
)

// DoGitPull ...
func DoGitPull(pth string) error {
	return RunCommandInDir(pth, "git", []string{"pull"}...)
}

// DoGitClone ...
func DoGitClone(uri, pth string) error {
	if uri == "" {
		return errors.New("Git Clone 'uri' missing")
	}
	if pth == "" {
		return errors.New("Git Clone 'pth' missing")
	}
	return RunCommand("git", []string{"clone", "--recursive", uri, pth}...)
}

// DoGitCheckoutBranch ...
func DoGitCheckoutBranch(repoPath, branch string) error {
	if branch == "" {
		return errors.New("Git checkout 'branch' missing")
	}
	if err := DoGitCheckout(repoPath, branch); err != nil {
		return RunCommandInDir(repoPath, "git", []string{"checkout", "-b", branch}...)
	}
	return nil
}

// DoGitCheckout ...
func DoGitCheckout(repoPath, checkout string) error {
	if checkout == "" {
		return errors.New("Git checkout 'hash' missing")
	}
	return RunCommandInDir(repoPath, "git", []string{"checkout", checkout}...)
}

// DoGitAddFile ...
func DoGitAddFile(repoPath, filePath string) error {
	if filePath == "" {
		return errors.New("Git add 'file' missing")
	}
	return RunCommandInDir(repoPath, "git", []string{"add", filePath}...)
}

// DoGitPush ...
func DoGitPush(repoPath string) error {
	return RunCommandInDir(repoPath, "git", []string{"push"}...)
}

// CheckIsNoGitChanges ...
func CheckIsNoGitChanges(repoPath string) error {
	return RunCommandInDir(repoPath, "git", []string{"diff", "--cahce", "--exit-code", "--quiet"}...)
}

// DoGitCommit ...
func DoGitCommit(repoPath string, message string) error {
	if message == "" {
		return errors.New("Git commit 'message' missing")
	}
	if err := CheckIsNoGitChanges(repoPath); err != nil {
		return RunCommandInDir(repoPath, "git", []string{"commit", "-m", message}...)
	}
	return nil
}

// DoGitCloneWithVersion ...
func DoGitCloneWithVersion(uri, pth, version string) error {
	if uri == "" {
		return errors.New("Git Clone 'uri' missing")
	}
	if pth == "" {
		return errors.New("Git Clone 'pth' missing")
	}
	if version == "" {
		return errors.New("Git Clone 'version' missing")
	}
	return RunCommand("git", []string{"clone", "--recursive", uri, pth, "--branch", version}...)
}

// DoGitGetCommit ...
func DoGitGetCommit(pth string) (string, error) {
	cmd := exec.Command("git", []string{"rev-parse", "HEAD"}...)
	cmd.Dir = pth
	bytes, err := cmd.CombinedOutput()
	cmdOutput := string(bytes)
	if err != nil {
		log.Error(cmdOutput)
		return "", err
	}
	return cmdOutput, nil
}

// DoGitCloneWithCommit ...
func DoGitCloneWithCommit(uri, pth, version, commithash string) error {
	if commithash == "" {
		return errors.New("Git Clone 'hash' missing")
	}
	if err := DoGitCloneWithVersion(uri, pth, version); err != nil {
		return err
	}

	cmdOutput, err := DoGitGetCommit(pth)
	if err != nil {
		return err
	}
	if commithash != cmdOutput {
		return fmt.Errorf("Commit hash doesn't match the one specified for the version tag. (version tag: %s) (expected: %s) (got: %s)", version, cmdOutput, commithash)
	}

	return DoGitCheckout(pth, commithash)
}

// DoGitUpdate ...
func DoGitUpdate(git, pth string) error {
	if exists, err := pathutil.IsPathExists(pth); err != nil {
		return err
	} else if !exists {
		log.Info("[STEPMAN] - Git path does not exist, do clone")
		return DoGitClone(git, pth)
	}

	log.Info("[STEPMAN] - Git path exist, do pull")
	return DoGitPull(pth)
}

func clearPathIfExist(pth string) error {
	if exist, err := pathutil.IsPathExists(pth); err != nil {
		return err
	} else if exist {
		if err := os.RemoveAll(pth); err != nil {
			return err
		}
	}
	return nil
}
