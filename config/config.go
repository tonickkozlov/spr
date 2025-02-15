package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ejoffe/rake"
	"github.com/ejoffe/spr/git"
)

type Config struct {
	Repo *RepoConfig
	User *UserConfig
}

// Config object to hold spr configuration
type RepoConfig struct {
	GitHubRepoOwner string `yaml:"githubRepoOwner"`
	GitHubRepoName  string `yaml:"githubRepoName"`

	RequireChecks   bool `default:"true" yaml:"requireChecks"`
	RequireApproval bool `default:"true" yaml:"requireApproval"`

	GitHubRemote string `default:"origin" yaml:"githubRemote"`
	GitHubBranch string `default:"master" yaml:"githubBranch"`
}

type UserConfig struct {
	ShowPRLink       bool `default:"true" yaml:"showPRLink"`
	LogGitCommands   bool `default:"false" yaml:"logGitCommands"`
	LogGitHubCalls   bool `default:"false" yaml:"logGitHubCalls"`
	StatusBitsHeader bool `default:"true" yaml:"statusBitsHeader"`

	Stargazer bool `default:"false" yaml:"stargazer"`
	RunCount  int  `default:"0" yaml:"runcount"`
}

func EmptyConfig() *Config {
	return &Config{
		Repo: &RepoConfig{},
		User: &UserConfig{},
	}
}

func DefaultConfig() *Config {
	cfg := EmptyConfig()
	rake.LoadSources(cfg.Repo,
		rake.DefaultSource(),
	)
	rake.LoadSources(cfg.User,
		rake.DefaultSource(),
	)
	return cfg
}

func ParseConfig(gitcmd git.GitInterface) *Config {
	cfg := EmptyConfig()

	rake.LoadSources(cfg.Repo,
		rake.DefaultSource(),
		GitHubRemoteSource(cfg, gitcmd),
		rake.YamlFileSource(RepoConfigFilePath(gitcmd)),
		rake.YamlFileWriter(RepoConfigFilePath(gitcmd)),
	)

	if cfg.Repo.GitHubRepoOwner == "" {
		fmt.Println("unable to auto configure repository owner - must be set manually in .spr.yml")
		os.Exit(3)
	}

	if cfg.Repo.GitHubRepoName == "" {
		fmt.Println("unable to auto configure repository name - must be set manually in .spr.yml")
		os.Exit(4)
	}

	rake.LoadSources(cfg.User,
		rake.DefaultSource(),
		rake.YamlFileSource(UserConfigFilePath()),
	)

	cfg.User.RunCount = cfg.User.RunCount + 1
	rake.LoadSources(cfg.User,
		rake.YamlFileWriter(UserConfigFilePath()))

	return cfg
}

func RepoConfigFilePath(gitcmd git.GitInterface) string {
	rootdir := gitcmd.RootDir()
	filepath := filepath.Clean(path.Join(rootdir, ".spr.yml"))
	return filepath
}

func UserConfigFilePath() string {
	rootdir, err := os.UserHomeDir()
	check(err)
	filepath := filepath.Clean(path.Join(rootdir, ".spr.yml"))
	return filepath
}

func GitHubRemoteSource(config *Config, gitcmd git.GitInterface) *remoteSource {
	return &remoteSource{
		gitcmd: gitcmd,
		config: config,
	}
}

type remoteSource struct {
	gitcmd git.GitInterface
	config *Config
}

func (s *remoteSource) Load(_ interface{}) {
	var output string
	err := s.gitcmd.Git("remote -v", &output)
	check(err)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		repoOwner, repoName, match := getRepoDetailsFromRemote(line)
		if match {
			s.config.Repo.GitHubRepoOwner = repoOwner
			s.config.Repo.GitHubRepoName = repoName
			break
		}
	}
}

func getRepoDetailsFromRemote(remote string) (string, string, bool) {
	// Allows "https://", "ssh://" or no protocol at all (this means ssh)
	protocolFormat := `(?:(https://)|(ssh://))?`
	// This may or may not be present in the address
	userFormat := `(git@)?`
	// "/" is expected in "http://" or "ssh://" protocol, when no protocol given
	// it should be ":"
	repoFormat := `github.com(/|:)(?P<repoOwner>\w+)/(?P<repoName>[\w-]+)`
	// This is neither required in https access nor in ssh one
	suffixFormat := `(.git)?`
	regexFormat := fmt.Sprintf(`^origin\s+%s%s%s%s \(push\)`,
		protocolFormat, userFormat, repoFormat, suffixFormat)
	regex := regexp.MustCompile(regexFormat)
	matches := regex.FindStringSubmatch(remote)
	if matches != nil {
		repoOwnerIndex := regex.SubexpIndex("repoOwner")
		repoNameIndex := regex.SubexpIndex("repoName")
		return matches[repoOwnerIndex], matches[repoNameIndex], true
	}
	return "", "", false
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

/*
func installCommitHook() {
	var rootdir string
	mustgit("rev-parse --show-toplevel", &rootdir)
	rootdir = strings.TrimSpace(rootdir)
	err := os.Chdir(rootdir)
	check(err)
	path, err := exec.LookPath("spr_commit_hook")
	check(err)
	cmd := exec.Command("ln", "-s", path, ".git/hooks/commit-msg")
	_, err = cmd.CombinedOutput()
	check(err)
	fmt.Printf("- Installed commit hook in .git/hooks/commit-msg\n")
}
*/
