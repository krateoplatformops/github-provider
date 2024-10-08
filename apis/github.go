package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	repov1alpha1 "github.com/krateoplatformops/github-provider/apis/repo/v1alpha1"
	collaboratorv1alpha1 "github.com/krateoplatformops/github-provider/apis/collaborator/v1alpha1"
	teamRepov1alpha1 "github.com/krateoplatformops/github-provider/apis/teamRepo/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		repov1alpha1.SchemeBuilder.AddToScheme,
		collaboratorv1alpha1.SchemeBuilder.AddToScheme,
		teamRepov1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
