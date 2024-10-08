package v1alpha1

import (
	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CollaboratorSpec defines the desired state of Collaborator
type CollaboratorSpec struct {
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

	// Repo: the name of the repository.
	// +immutable
	Repo string `json:"repo"`

	// Username: the handle for the GitHub user account.
	// +immutable
	Username string `json:"username"`

	// Permission: The permission to grant the collaborator. Only valid on organization-owned repositories. We accept the following permissions to be set: pull, triage, push, maintain, admin and you can also specify a custom repository role name, if the owning organization has defined any. Default: push
	// +immutable
	Permission string `json:"permission"`
}

// CollaboratorStatus defines the observed state of Collaborator
type CollaboratorStatus struct {
	prv1.ConditionedStatus `json:",inline"`

	// Permission: The permission granted to the collaborator.
	Permission *string `json:"permission,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced,categories={krateo,github}
//+kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"

// Collaborator is the Schema for the collaborators API
type Collaborator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CollaboratorSpec   `json:"spec,omitempty"`
	Status CollaboratorStatus `json:"status,omitempty"`
}

// GetCondition of this Collaborator.
func (mg *Collaborator) GetCondition(ct prv1.ConditionType) prv1.Condition {
	return mg.Status.GetCondition(ct)
}

// SetConditions of this Collaborator.
func (mg *Collaborator) SetConditions(c ...prv1.Condition) {
	mg.Status.SetConditions(c...)
}

//+kubebuilder:object:root=true

// CollaboratorList contains a list of Collaborator
type CollaboratorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Collaborator `json:"items"`
}

// GetItems of this CollaboratorList.
func (l *CollaboratorList) GetItems() []resource.Managed {
	items := make([]resource.Managed, len(l.Items))
	for i := range l.Items {
		items[i] = &l.Items[i]
	}
	return items
}
