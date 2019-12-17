package gitops

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/golang/glog"

	"kubecloud/common/utils"
)

type Git struct {
	Dir         string
	Url         string
	Branch      string
	Token       string
	commandName string
}

func NewGit(dir, url, branch, token string) *Git {
	return &Git{
		Dir:         dir,
		Url:         url,
		Branch:      branch,
		Token:       token,
		commandName: "git",
	}
}

func (g *Git) configUserEmail() error {
	args := []string{"config", "user.email", `"kubecloud@example.com"`}
	out, err := utils.ExecCommand(g.Dir, g.commandName, args...)
	if err != nil {
		glog.Errorf(`git config user.email "kubecloud@example.com" error: %s`, out)
		return fmt.Errorf(out)
	}
	glog.Infof(`git config user.name "kubecloud"`)
	args = []string{"config", "user.name", `"kubecloud"`}
	out, err = utils.ExecCommand(g.Dir, "git", args...)
	if err != nil {
		glog.Errorf(`git config user.name "kubecloud" error: %s`, out)
		return fmt.Errorf(out)
	}
	glog.Infof(`git config user.name "kubecloud"`)
	return nil
}

func (g *Git) Clone() error {
	currentDir, _ := filepath.Abs(`.`)
	urlWithToken := strings.ReplaceAll(g.Url, "//", fmt.Sprintf("//kubecloud:%s@", g.Token))
	args := []string{"clone", urlWithToken, "-b", g.Branch, g.Dir}
	out, err := utils.ExecCommand(currentDir, g.commandName, args...)
	if err != nil && !strings.Contains(out, "already exists and is not an empty directory") {
		glog.Errorf("git clone %s -b %s %s error: %s", g.Url, g.Branch, g.Dir, out)
		return fmt.Errorf(out)
	}
	glog.Infof("git clone %s -b %s", g.Url, g.Branch)
	glog.Infoln(out)
	if err := g.configUserEmail(); err != nil {
		return fmt.Errorf(out)
	}
	return nil
}

func (g *Git) Commit(files []string, msg string) error {
	for _, f := range files {
		args := []string{"add", f}
		out, err := utils.ExecCommand(g.Dir, g.commandName, args...)
		if err != nil {
			glog.Errorf("git add %s error: %s", f, out)
			return fmt.Errorf(out)
		}
		glog.Infof("git add %s", f)
	}
	args := []string{"commit", "-m", fmt.Sprintf(`"%s"`, msg)}
	out, err := utils.ExecCommand(g.Dir, g.commandName, args...)
	if err != nil && !strings.Contains(out, "nothing to commit, working tree clean") {
		glog.Errorf("git commit -m  %s error: %s", fmt.Sprintf(`"%s"`, msg), out)
		return fmt.Errorf(out)
	}
	glog.Infof("git commit -m %s", fmt.Sprintf(`"%s"`, msg))
	glog.Infoln(out)
	return nil
}

func (g *Git) Pull() error {
	args := []string{"pull", "origin", g.Branch}
	out, err := utils.ExecCommand(g.Dir, g.commandName, args...)
	if err != nil {
		glog.Errorf("git pull origin %s error: %s", g.Branch, out)
		return fmt.Errorf(out)
	}
	glog.Infof("git pull origin %s", g.Branch)
	glog.Infoln(out)
	return nil
}

func (g *Git) Push() error {
	args := []string{"push", "origin", g.Branch}
	out, err := utils.ExecCommand(g.Dir, g.commandName, args...)
	if err != nil {
		glog.Errorf("git push origin %s error: %s", g.Branch, out)
		return fmt.Errorf(out)
	}
	glog.Infof("git push origin %s", g.Branch)
	glog.Infoln(out)
	return nil
}
