package repository

import (
	"errors"
	"github.com/zhenorzz/goploy/core"
	"github.com/zhenorzz/goploy/model"
	"github.com/zhenorzz/goploy/utils"
	"os"
	"strconv"
	"strings"
)

type GitRepo struct {
}

// Create -
func (GitRepo) Create(projectID int64) error {
	srcPath := core.GetProjectPath(projectID)
	if _, err := os.Stat(srcPath); err == nil {
		return nil
	}
	project, err := model.Project{ID: projectID}.GetData()
	if err != nil {
		core.Log(core.TRACE, "The project does not exist, projectID:"+strconv.FormatInt(projectID, 10))
		return err
	}
	if err := os.RemoveAll(srcPath); err != nil {
		core.Log(core.TRACE, "The project fail to remove, projectID:"+strconv.FormatInt(project.ID, 10)+" ,error: "+err.Error())
		return err
	}
	git := utils.GIT{}
	if err := git.Clone(project.URL, srcPath); err != nil {
		core.Log(core.ERROR, "The project fail to initialize, projectID:"+strconv.FormatInt(project.ID, 10)+" ,error: "+err.Error()+", detail: "+git.Err.String())
		return err
	}
	if project.Branch != "master" {
		git.Dir = srcPath
		if err := git.Checkout("-b", project.Branch, "origin/"+project.Branch); err != nil {
			core.Log(core.ERROR, "The project fail to switch branch, projectID:"+strconv.FormatInt(project.ID, 10)+" ,error: "+err.Error()+", detail: "+git.Err.String())
			_ = os.RemoveAll(srcPath)
			return err
		}
	}
	core.Log(core.TRACE, "The project success to initialize, projectID:"+strconv.FormatInt(project.ID, 10))
	return nil
}

func (gitRepo GitRepo) Follow(project model.Project, target string) error {
	if err := gitRepo.Create(project.ID); err != nil {
		return err
	}
	git := utils.GIT{Dir: core.GetProjectPath(project.ID)}
	core.Log(core.TRACE, "projectID:"+strconv.FormatInt(project.ID, 10)+" git add .")
	if err := git.Add("."); err != nil {
		core.Log(core.ERROR, err.Error()+", detail: "+git.Err.String())
		return errors.New(git.Err.String())
	}

	core.Log(core.TRACE, "projectID:"+strconv.FormatInt(project.ID, 10)+" git reset --hard")
	if err := git.Reset("--hard"); err != nil {
		core.Log(core.ERROR, err.Error()+", detail: "+git.Err.String())
		return errors.New(git.Err.String())
	}

	// the length of commit id is 40
	if len(target) != 40 {
		core.Log(core.TRACE, "projectID:"+strconv.FormatInt(project.ID, 10)+" git fetch")
		if err := git.Fetch(); err != nil {
			core.Log(core.ERROR, err.Error()+", detail: "+git.Err.String())
			return errors.New(git.Err.String())
		}
	}

	core.Log(core.TRACE, "projectID:"+strconv.FormatInt(project.ID, 10)+" git checkout -B goploy "+target)
	if err := git.Checkout("-B", "goploy", target); err != nil {
		core.Log(core.ERROR, err.Error()+", detail: "+git.Err.String())
		return errors.New(git.Err.String())
	}
	return nil
}

func (GitRepo) CommitLog(projectID int64, rows int) ([]CommitInfo, error) {
	git := utils.GIT{Dir: core.GetProjectPath(projectID)}

	if err := git.Log("--stat", "--pretty=format:`start`%H`%an`%at`%s`%d`", "-n", strconv.Itoa(rows)); err != nil {
		return []CommitInfo{}, errors.New(git.Err.String())
	}

	list := parseGITLog(git.Output.String())
	return list, nil
}

func (GitRepo) BranchLog(projectID int64, branch string, rows int) ([]CommitInfo, error) {
	git := utils.GIT{Dir: core.GetProjectPath(projectID)}

	if err := git.Log(branch, "--stat", "--pretty=format:`start`%H`%an`%at`%s`%d`", "-n", strconv.Itoa(rows)); err != nil {
		return []CommitInfo{}, errors.New(git.Err.String())
	}

	list := parseGITLog(git.Output.String())
	return list, nil
}

func (GitRepo) TagLog(projectID int64, rows int) ([]CommitInfo, error) {
	git := utils.GIT{Dir: core.GetProjectPath(projectID)}
	if err := git.Add("."); err != nil {
		return []CommitInfo{}, errors.New(git.Err.String())
	}

	if err := git.Reset("--hard"); err != nil {
		return []CommitInfo{}, errors.New(git.Err.String())
	}

	if err := git.Pull(); err != nil {
		return []CommitInfo{}, errors.New(git.Err.String())
	}

	if err := git.Log("--tags", "-n", strconv.Itoa(rows), "--no-walk", "--stat", "--pretty=format:`start`%H`%an`%at`%s`%d`"); err != nil {
		return []CommitInfo{}, errors.New(git.Err.String())
	}

	list := parseGITLog(git.Output.String())
	return list, nil
}

func parseGITLog(rawCommitLog string) []CommitInfo {
	unformatCommitList := strings.Split(rawCommitLog, "`start`")
	unformatCommitList = unformatCommitList[1:]
	var commitList []CommitInfo
	for _, commitRow := range unformatCommitList {
		commitRowSplit := strings.Split(commitRow, "`")
		timestamp, _ := strconv.Atoi(commitRowSplit[2])
		commitList = append(commitList, CommitInfo{
			Commit:    commitRowSplit[0],
			Author:    commitRowSplit[1],
			Timestamp: timestamp,
			Message:   commitRowSplit[3],
			Tag:       commitRowSplit[4],
			Diff:      strings.Trim(commitRowSplit[5], "\n"),
		})
	}
	return commitList
}
