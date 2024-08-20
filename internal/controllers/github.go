package controllers

import (
	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/krateoplatformops/github-provider/internal/controllers/repo"
	"github.com/krateoplatformops/github-provider/internal/controllers/collaborator"
	"github.com/krateoplatformops/github-provider/internal/controllers/teamRepo"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		repo.Setup,
		collaborator.Setup,
		teamRepo.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
