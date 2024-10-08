package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/carlmjohnson/requests"
	"github.com/krateoplatformops/github-provider/apis/collaborator/v1alpha1"
)

// CollaboratorService provides methods for creating and reading collaborators.
type CollaboratorService struct {
	client       *http.Client
	apiUrl       string
	apiExtraPath string
	token        string
}

type CollaboratorPermission struct {
	Permission string `json:"permission"`
	RoleName   string `json:"role_name"`
}

// newCollaboratorService returns a new CollaboratorService.
func newCollaboratorService(httpClient *http.Client, apiUrl, extraPath, token string) *CollaboratorService {
	return &CollaboratorService{
		client:       httpClient,
		apiUrl:       apiUrl,
		apiExtraPath: extraPath,
		token:        token,
	}
}

func (s *CollaboratorService) Create(opts *v1alpha1.CollaboratorSpec) error {
	ok, err := s.isOrg(opts.Org)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Valid organization required to set collaborators on a repository")
	}
	
	pt := path.Join(s.apiExtraPath, fmt.Sprintf("repos/%s/%s/collaborators/%s", opts.Org, opts.Repo, opts.Username))

	githubError := &GithubError{}

	err = requests.URL(s.apiUrl).Path(pt).
		Client(s.client).
		Method(http.MethodPut).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		BodyJSON(map[string]interface{}{
			"permission":      opts.Permission,
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

// Check if a user is a collaborator of a repository.
//
// GitHub API docs: https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2022-11-28#check-if-a-user-is-a-repository-collaborator
func (s *CollaboratorService) Exists(opts *v1alpha1.CollaboratorSpec) (bool, error) {
	pt := path.Join(s.apiExtraPath, fmt.Sprintf("repos/%s/%s/collaborators/%s", opts.Org, opts.Repo, opts.Username))

	err := requests.URL(s.apiUrl).Path(pt).
		Client(s.client).
		Method(http.MethodGet).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		CheckStatus(204).
		Fetch(context.Background())
	if err != nil {
		if requests.HasStatusErr(err, 404) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// Get repository permissions for a user
//
// GitHub API docs: https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2022-11-28#get-repository-permissions-for-a-user
func (s *CollaboratorService) GetPermission(opts *v1alpha1.CollaboratorSpec) (string, error) {
	pt := path.Join(s.apiExtraPath, fmt.Sprintf("repos/%s/%s/collaborators/%s/permission", opts.Org, opts.Repo, opts.Username))

	var res CollaboratorPermission 

	err := requests.URL(s.apiUrl).Path(pt).
		Client(s.client).
		Method(http.MethodGet).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		CheckStatus(200).
		ToJSON(&res).
		Fetch(context.Background())
	if err != nil {
		return "", err
	}

	return res.Permission, nil
}

// https://docs.github.com/en/rest/collaborators/collaborators?apiVersion=2022-11-28#remove-a-repository-collaborator
func (s *CollaboratorService) Delete(opts *v1alpha1.CollaboratorSpec) error {
	pt := path.Join(s.apiExtraPath, fmt.Sprintf("repos/%s/%s/collaborators/%s", opts.Org, opts.Repo, opts.Username))

	err := requests.URL(s.apiUrl).Path(pt).
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

func (s *CollaboratorService) isOrg(owner string) (bool, error) {
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
