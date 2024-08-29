package v1alpha1

import (
	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepoSpec defines the desired state of Repo
type RepoSpec struct {
	// ApiUrl: the baseUrl for the REST API provider.
	// +optional
	// +immutable
	ApiUrl string `json:"apiUrl,omitempty"`

	// Credentials required to authenticate ReST API git server.
	Credentials *prv1.CredentialSelectors `json:"credentials"`

	// Verbose is true dumps your client requests and responses.
	// +optional
	Verbose *bool `json:"verbose,omitempty"`

	// Org: the organization name.
	// +immutable
	Org string `json:"org"`

	// Name: the name of the repository.
	// +immutable
	Name string `json:"name"`

	// Private: whether the repository is private (default: true).
	// +optional
	Private bool `json:"private,omitempty"`

	// Initialize: whether the repository must be initialized (default: true).
	// +optional
	Initialize *bool `json:"initialize,omitempty"`
}

// RepoStatus defines the observed state of Repo
type RepoStatus struct {
	prv1.ConditionedStatus `json:",inline"`

	// Url: repository URL.
	Url *string `json:"url,omitempty"`

	// Private: whether the repository is private.
	Private *bool `json:"private,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced,categories={krateo,github}
//+kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"

// Repo is the Schema for the repoes API
type Repo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepoSpec   `json:"spec,omitempty"`
	Status RepoStatus `json:"status,omitempty"`
}

// GetCondition of this Repo.
func (mg *Repo) GetCondition(ct prv1.ConditionType) prv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this Repo.
func (mg *Repo) SetConditions(c ...prv1.Condition) {
	mg.Status.SetConditions(c...)
}

//+kubebuilder:object:root=true

// RepoList contains a list of Repo
type RepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repo `json:"items"`
}

// GetItems of this RepoList.
func (l *RepoList) GetItems() []resource.Managed {
	items := make([]resource.Managed, len(l.Items))
	for i := range l.Items {
		items[i] = &l.Items[i]
	}
	return items
}
