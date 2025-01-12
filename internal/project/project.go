package project

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
)

type Resolver struct {
	cwd string

	once sync.Once
	repo *repo
}

func NewResolver(dir string) *Resolver {
	return &Resolver{cwd: dir}
}

func (r *Resolver) openRepo() *repo {
	r.once.Do(func() {
		gitRepo, err := git.PlainOpen(r.cwd)
		r.repo = &repo{Repository: gitRepo, err: err}
	})
	return r.repo
}

type Project struct {
	BranchName string
	Commit     string
	URL        string
}

func (r *Resolver) Get() (p Project, err error) {
	repo, err := r.openRepo().Clone()
	if err != nil {
		return p, err
	}

	p.BranchName = repo.BranchName()
	p.Commit = repo.Commit()
	p.URL = repo.URL()

	return p, repo.Err()
}

type repo struct {
	*git.Repository
	err error
}

func (r *repo) Clone() (*repo, error) {
	if err := r.Err(); err != nil {
		return nil, err
	}
	return &repo{Repository: r.Repository}, nil
}

func (r *repo) Err() error { return r.err }

func (r *repo) BranchName() string {
	if r.Err() != nil {
		return ""
	}

	ref, err := r.Head()
	if err != nil {
		r.err = err
		return ""
	}

	if ref.Name().IsBranch() {
		return ref.Name().Short()
	}
	return ""
}

func (r *repo) Commit() string {
	if r.Err() != nil {
		return ""
	}

	ref, err := r.Head()
	if err != nil {
		r.err = err
		return ""
	}
	return ref.Hash().String()
}

func (r *repo) URL() string {
	if r.Err() != nil {
		return ""
	}

	remotes, err := r.Remotes()
	if err != nil {
		r.err = err
		return ""
	}
	if len(remotes) == 0 {
		return ""
	}
	return selectRemoteURL(remotes)
}

func selectRemoteURL(remotes []*git.Remote) string {
	sort.Slice(remotes, func(i, j int) bool {
		a, b := remotes[i], remotes[j]
		if a.Config().Name == "origin" {
			return true
		}
		if b.Config().Name == "origin" {
			return false
		}
		aURL, bURL := a.Config().URLs[0], b.Config().URLs[0]
		aIncludesGithub, bIncludesGithub := strings.Contains(aURL, "github"), strings.Contains(bURL, "github")
		if aIncludesGithub != bIncludesGithub {
			if aIncludesGithub {
				return true
			}
			if bIncludesGithub {
				return false
			}
		}
		return strings.Compare(aURL, bURL) == -1
	})
	return remotes[0].Config().URLs[0]
}

func GetCurrentGitEmail(cwd string) (string, error) {
	cmdSlice := []string{"git", "config", "user.email"}
	cmd := exec.Command(cmdSlice[0], cmdSlice[1:]...)
	cmd.Dir = cwd

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

type Branch struct {
	Name        string
	Description string
}

func GetUsersBranchNames(cwd string, email string) ([]Branch, error) {
	// Get a list of branch names for a specific user within this repository
	// Due to limitations of go-git, we use shell for this.
	//
	// NB: user is _not_ sanitized, don't pass untrusted.
	//     Also due to weird exec.Command quoting, don't include whitespace
	//
	cmdSlice := []string{"git", "log", "--format=%s--||--%b", "--merges", "--author=" + strings.Trim(email, "\n")}
	return getBranchNamesFromCommand(cwd, cmdSlice, false)
}

func GetBranchNames(cwd string) ([]Branch, error) {
	cmdSlice := []string{"git", "log", "--format=%s--||--%b", "--merges"}
	return getBranchNamesFromCommand(cwd, cmdSlice, true)
}

func getBranchNamesFromCommand(cwd string, cmdSlice []string, greedy bool) ([]Branch, error) {
	if len(cmdSlice) < 2 {
		return nil, errors.New("command is not long enough")
	}
	cmd := exec.Command(cmdSlice[0], cmdSlice[1:]...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	stdout := string(out)
	branches := getBranchNamesFromStdout(stdout, greedy)
	return branches, nil
}

func getBranchNamesFromStdout(stdout string, greedy bool) []Branch {
	var branches []Branch
	modifier := "(?mU)"
	if greedy {
		modifier = "(?m)"
	}
	re := regexp.MustCompile(fmt.Sprintf(`%s(\S+)[:\/](\S+)\s?$`, modifier))
	for _, line := range strings.Split(stdout, "\n") {
		split := strings.Split(line, "--||--")
		orgbranch := re.FindStringSubmatch(split[0])
		if len(orgbranch) == 3 && len(split) > 1 && len(split[1]) > 1 {
			branches = append(branches, Branch{Name: orgbranch[2], Description: split[1]})
		}
	}
	return branches
}

func GetUsersBranches(repoUser string) ([]Branch, error) {
	var email string
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if len(repoUser) > 0 {
		email = repoUser
	} else {
		email, err = GetCurrentGitEmail(cwd)
		if err != nil {
			return nil, errors.New("could not find current git user")
		}
	}

	branches, err := GetUsersBranchNames(cwd, email)
	if err != nil {
		return nil, errors.New("error while querying user's branches")
	}

	return branches, nil
}

func GetRepoBranches() ([]Branch, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	branches, err := GetBranchNames(cwd)
	if err != nil {
		return nil, errors.New("error while querying repository branches")
	}

	return branches, nil
}
