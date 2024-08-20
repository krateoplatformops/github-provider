package v1alpha1

import (
	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TeamRepoSpec defines the desired state of TeamRepo
type TeamRepoSpec struct {
	// ApiUrl: the baseUrl for the REST API provider.
	// +optional
	// +immutable
	ApiUrl string `json:"apiUrl,omitempty"`

	// Credentials required to authenticate ReST API git server.
	Credentials *prv1.CredentialSelectors `json:"credentials"`

	// Verbose is true dumps your client requests and responses.
	// +optional
	Verbose *bool `json:"verbose,omitempty"`

	// Org: The organization name. The name is not case sensitive.
	// +immutable
	Org string `json:"org"`

	// TeamSlug: The slug of the team name.
	// +immutable
	TeamSlug string `json:"teamSlug"`

	// Owner: The account owner of the repository. The name is not case sensitive.
	// +immutable
	Owner string `json:"owner"`

	// Repo: The name of the repository without the .git extension. The name is not case sensitive.
	// +immutable
	Repo string `json:"repo"`

	// Permission: The permission to grant the team on this repository. We accept the following permissions to be set: pull, triage, push, maintain, admin and you can also specify a custom repository role name, if the owning organization has defined any. If no permission is specified, the team's permission attribute will be used to determine what permission to grant the team on this repository.
	// +immutable
	Permission string `json:"permission"`
}

// TeamRepoStatus defines the observed state of Repo
type TeamRepoStatus struct {
	prv1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced,categories={krateo,github}
//+kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"

// TeamRepo is the Schema for the repoes API
type TeamRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamRepoSpec   `json:"spec,omitempty"`
	Status TeamRepoStatus `json:"status,omitempty"`
}

// GetCondition of this TeamRepo.
func (mg *TeamRepo) GetCondition(ct prv1.ConditionType) prv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this TeamRepo.
func (mg *TeamRepo) SetConditions(c ...prv1.Condition) {
	mg.Status.SetConditions(c...)
}

//+kubebuilder:object:root=true

// TeamRepoList contains a list of TeamRepo
type TeamRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeamRepo `json:"items"`
}

// GetItems of this RepoList.
func (l *TeamRepoList) GetItems() []resource.Managed {
	items := make([]resource.Managed, len(l.Items))
	for i := range l.Items {
		items[i] = &l.Items[i]
	}
	return items
}
