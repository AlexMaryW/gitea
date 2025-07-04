// Copyright 2017 Gitea. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"net/http"

	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v1/utils"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
	commitstatus_service "code.gitea.io/gitea/services/repository/commitstatus"
)

// NewCommitStatus creates a new CommitStatus
func NewCommitStatus(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/statuses/{sha} repository repoCreateStatus
	// ---
	// summary: Create a commit status
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: username of the user or organization owning the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: sha
	//   in: path
	//   description: sha of the commit
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateStatusOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/CommitStatus"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*api.CreateStatusOption)
	sha := ctx.PathParam("sha")
	if len(sha) == 0 {
		ctx.APIError(http.StatusBadRequest, nil)
		return
	}
	status := &git_model.CommitStatus{
		State:       form.State,
		TargetURL:   form.TargetURL,
		Description: form.Description,
		Context:     form.Context,
	}
	if err := commitstatus_service.CreateCommitStatus(ctx, ctx.Repo.Repository, ctx.Doer, sha, status); err != nil {
		ctx.APIErrorInternal(err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToCommitStatus(ctx, status))
}

// GetCommitStatuses returns all statuses for any given commit hash
func GetCommitStatuses(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/statuses/{sha} repository repoListStatuses
	// ---
	// summary: Get a commit's statuses
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: username of the user or organization owning the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: sha
	//   in: path
	//   description: sha of the commit
	//   type: string
	//   required: true
	// - name: sort
	//   in: query
	//   description: type of sort
	//   type: string
	//   enum: [oldest, recentupdate, leastupdate, leastindex, highestindex]
	//   required: false
	// - name: state
	//   in: query
	//   description: type of state
	//   type: string
	//   enum: [pending, success, error, failure, warning]
	//   required: false
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/CommitStatusList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	getCommitStatuses(ctx, ctx.PathParam("sha"))
}

// GetCommitStatusesByRef returns all statuses for any given commit ref
func GetCommitStatusesByRef(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/commits/{ref}/statuses repository repoListStatusesByRef
	// ---
	// summary: Get a commit's statuses, by branch/tag/commit reference
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: username of the user or organization owning the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: ref
	//   in: path
	//   description: name of branch/tag/commit
	//   type: string
	//   required: true
	// - name: sort
	//   in: query
	//   description: type of sort
	//   type: string
	//   enum: [oldest, recentupdate, leastupdate, leastindex, highestindex]
	//   required: false
	// - name: state
	//   in: query
	//   description: type of state
	//   type: string
	//   enum: [pending, success, error, failure, warning]
	//   required: false
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/CommitStatusList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	refCommit := resolveRefCommit(ctx, ctx.PathParam("ref"), 7)
	if ctx.Written() {
		return
	}
	getCommitStatuses(ctx, refCommit.CommitID)
}

func getCommitStatuses(ctx *context.APIContext, commitID string) {
	repo := ctx.Repo.Repository

	listOptions := utils.GetListOptions(ctx)

	statuses, maxResults, err := db.FindAndCount[git_model.CommitStatus](ctx, &git_model.CommitStatusOptions{
		ListOptions: listOptions,
		RepoID:      repo.ID,
		SHA:         commitID,
		SortType:    ctx.FormTrim("sort"),
		State:       ctx.FormTrim("state"),
	})
	if err != nil {
		ctx.APIErrorInternal(fmt.Errorf("GetCommitStatuses[%s, %s, %d]: %w", repo.FullName(), commitID, ctx.FormInt("page"), err))
		return
	}

	apiStatuses := make([]*api.CommitStatus, 0, len(statuses))
	for _, status := range statuses {
		apiStatuses = append(apiStatuses, convert.ToCommitStatus(ctx, status))
	}

	ctx.SetLinkHeader(int(maxResults), listOptions.PageSize)
	ctx.SetTotalCountHeader(maxResults)

	ctx.JSON(http.StatusOK, apiStatuses)
}

// GetCombinedCommitStatusByRef returns the combined status for any given commit hash
func GetCombinedCommitStatusByRef(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/commits/{ref}/status repository repoGetCombinedStatusByRef
	// ---
	// summary: Get a commit's combined status, by branch/tag/commit reference
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: username of the user or organization owning the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: ref
	//   in: path
	//   description: name of branch/tag/commit
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/CombinedStatus"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	refCommit := resolveRefCommit(ctx, ctx.PathParam("ref"), 7)
	if ctx.Written() {
		return
	}

	repo := ctx.Repo.Repository

	statuses, err := git_model.GetLatestCommitStatus(ctx, repo.ID, refCommit.Commit.ID.String(), utils.GetListOptions(ctx))
	if err != nil {
		ctx.APIErrorInternal(fmt.Errorf("GetLatestCommitStatus[%s, %s]: %w", repo.FullName(), refCommit.CommitID, err))
		return
	}

	count, err := git_model.CountLatestCommitStatus(ctx, repo.ID, refCommit.Commit.ID.String())
	if err != nil {
		ctx.APIErrorInternal(fmt.Errorf("CountLatestCommitStatus[%s, %s]: %w", repo.FullName(), refCommit.CommitID, err))
		return
	}
	ctx.SetTotalCountHeader(count)

	if len(statuses) == 0 {
		ctx.JSON(http.StatusOK, &api.CombinedStatus{})
		return
	}

	combiStatus := convert.ToCombinedStatus(ctx, statuses, convert.ToRepo(ctx, repo, ctx.Repo.Permission))
	ctx.JSON(http.StatusOK, combiStatus)
}
