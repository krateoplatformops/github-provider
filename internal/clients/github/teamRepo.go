package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/carlmjohnson/requests"
	"github.com/krateoplatformops/github-provider/apis/teamRepo/v1alpha1"
)

// TeamRepoService provides methods for creating and reading team permissions to repositories.
type TeamRepoService struct {
	client       *http.Client
	apiUrl       string
	apiExtraPath string
	token        string
}

type TeamRepoPermissions struct {
	Permissions map[string]bool `json:"permissions"`
	RoleName      string `json:"role_name"`
}

// newTeamRepoService returns a new TeamRepoService.
func newTeamRepoService(httpClient *http.Client, apiUrl, extraPath, token string) *TeamRepoService {
	return &TeamRepoService{
		client:       httpClient,
		apiUrl:       apiUrl,
		apiExtraPath: extraPath,
		token:        token,
	}
}

// Create att team permission to a repository.
//
// GitHub API docs: https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#add-or-update-team-repository-permissions
func (s *TeamRepoService) Create(opts *v1alpha1.TeamRepoSpec) error {
	_, err := s.isOrg(opts.Org)
	if err != nil {
		return err
	}

	pt := path.Join(s.apiExtraPath, fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", opts.Org, opts.TeamSlug, opts.Owner, opts.Repo))

	githubError := &GithubError{}

	err = requests.URL(s.apiUrl).Path(pt).
		Client(s.client).
		Method(http.MethodPut).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		BodyJSON(map[string]interface{}{
			"permission":   opts.Permission,
		}).
		AddValidator(ErrorJSON(githubError, 204)).
		Fetch(context.Background())
	if err != nil {
		var gerr *GithubError
		if errors.As(err, &gerr) {
			return fmt.Errorf(gerr.Error())
		}
		return err
	}

	return nil
}

// Get the permission of a team over a given repository
//
// GitHub API docs: https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#check-team-permissions-for-a-repository
func (s *TeamRepoService) GetPermissions(opts *v1alpha1.TeamRepoSpec) (map[string]bool, error) {
	_, err := s.isOrg(opts.Org)
	if err != nil {
		return nil, err
	}
	pt := path.Join(s.apiExtraPath, fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", opts.Org, opts.TeamSlug, opts.Owner, opts.Repo))

	var res TeamRepoPermissions

	err = requests.URL(s.apiUrl).Path(pt).
		Client(s.client).
		Method(http.MethodGet).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		Header("Accept", "application/vnd.github.v3.repository+json").
		CheckStatus(200).
		ToJSON(&res).
		Fetch(context.Background())
	if err != nil {
		if requests.HasStatusErr(err, 404) {
			return res.Permissions, nil
		}

		return res.Permissions, err
	}

	return res.Permissions, nil
}

// Deleting a repository requires admin access. If OAuth is used, the delete_repo scope is required.
// https://docs.github.com/en/rest/repos/repos#get-a-repository
func (s *TeamRepoService) Delete(opts *v1alpha1.TeamRepoSpec) error {
	_, err := s.isOrg(opts.Org)
	if err != nil {
		return err
	}
	pt := path.Join(s.apiExtraPath, fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", opts.Org, opts.TeamSlug, opts.Owner, opts.Repo))

	err = requests.URL(s.apiUrl).Path(pt).
		Client(s.client).
		Method(http.MethodDelete).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		CheckStatus(204).
		Fetch(context.Background())
	if err != nil {
		if requests.HasStatusErr(err, 404) {
			return nil
		}

		return err
	}

	return nil
}

func (s *TeamRepoService) isOrg(owner string) (bool, error) {
	pt := path.Join(s.apiExtraPath, fmt.Sprintf("/orgs/%s", owner))
	err := requests.URL(s.apiUrl).Path(pt).
		Client(s.client).
		Method(http.MethodGet).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		CheckStatus(200).
		Fetch(context.Background())
	if err != nil {
		if requests.HasStatusErr(err, 404) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
