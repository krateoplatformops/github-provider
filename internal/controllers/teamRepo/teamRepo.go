package teamRepo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/krateoplatformops/provider-runtime/pkg/controller"
	"github.com/krateoplatformops/provider-runtime/pkg/ratelimiter"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krateoplatformops/provider-runtime/pkg/event"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/reconciler"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"

	teamRepov1alpha1 "github.com/krateoplatformops/github-provider/apis/teamRepo/v1alpha1"
	"github.com/krateoplatformops/github-provider/internal/clients"
	"github.com/krateoplatformops/github-provider/internal/clients/github"
	"github.com/krateoplatformops/provider-runtime/pkg/ptr"
)

const (
	errNotTeamRepo = "managed resource is not a teamRepo custom resource"
)

// Setup adds a controller that reconciles Token managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := reconciler.ControllerName(teamRepov1alpha1.TeamRepoGroupKind)

	log := o.Logger.WithValues("controller", name)

	recorder := mgr.GetEventRecorderFor(name)

	r := reconciler.NewReconciler(mgr,
		resource.ManagedKind(teamRepov1alpha1.TeamRepoGroupVersionKind),
		reconciler.WithExternalConnecter(&connector{
			kube:     mgr.GetClient(),
			log:      log,
			recorder: recorder,
		}),
		reconciler.WithPollInterval(o.PollInterval),
		reconciler.WithLogger(log),
		reconciler.WithRecorder(event.NewAPIRecorder(recorder)))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&teamRepov1alpha1.TeamRepo{}).
		Complete(ratelimiter.New(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube     client.Client
	log      logging.Logger
	recorder record.EventRecorder
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (reconciler.ExternalClient, error) {
	cr, ok := mg.(*teamRepov1alpha1.TeamRepo)
	if !ok {
		return nil, errors.New(errNotTeamRepo)
	}

	spec := cr.Spec.DeepCopy()

	csr := spec.Credentials.SecretRef
	if csr == nil {
		return nil, fmt.Errorf("no credentials secret referenced")
	}

	token, err := resource.GetSecret(ctx, c.kube, csr.DeepCopy())
	if err != nil {
		return nil, err
	}

	opts := github.ClientOpts{
		ApiURL: spec.ApiUrl,
		Token:  token,
	}
	opts.HttpClient = clients.DefaultHttpClient()

	if ptr.Deref(cr.Spec.Verbose, false) {
		opts.HttpClient = &http.Client{
			Transport: &clients.VerboseTracer{RoundTripper: http.DefaultTransport},
			Timeout:   50 * time.Second,
		}
	}

	return &external{
		kube:  c.kube,
		log:   c.log,
		ghCli: github.NewClient(opts),
		rec:   c.recorder,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube  client.Client
	log   logging.Logger
	ghCli *github.Client
	rec   record.EventRecorder
}

func (c *external) Disconnect(_ context.Context) error {
	return nil // NOOP
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (reconciler.ExternalObservation, error) {
	cr, ok := mg.(*teamRepov1alpha1.TeamRepo)
	if !ok {
		return reconciler.ExternalObservation{}, errors.New(errNotTeamRepo)
	}

	spec := cr.Spec.DeepCopy()

	permissions, err := e.ghCli.TeamRepo().GetPermissions(spec)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}

	if len(permissions) == 0 {
		e.log.Debug("Team not permitted", "org", spec.Org, "team", spec.TeamSlug, "owner", spec.Owner, "repo", spec.Repo)
		e.rec.Eventf(cr, corev1.EventTypeNormal, "NotPermitted", "Team %s/%s not permitted any access to repo %s (or repo not existent)", spec.Org, spec.TeamSlug, spec.Repo)

		return reconciler.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: false,
		}, nil
	}

	// Test that only spec.Permission is given. All others permissions must not.
	for permission, given := range permissions {
		if (permission == spec.Permission && !given) || (permission != spec.Permission && given) {
			e.log.Debug("Team missing permission", "org", spec.Org, "team", spec.TeamSlug, "owner", spec.Owner, "repo", spec.Repo)
			e.rec.Eventf(cr, corev1.EventTypeNormal, "NotPermitted", "Team %s/%s misses exact %s permission to repo %s", spec.Org, spec.TeamSlug, spec.Permission, spec.Repo)
	
			return reconciler.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: false,
			}, nil
		}
	}

	e.log.Debug("Team already permitted", "org", spec.Org, "team", spec.TeamSlug, "owner", spec.Owner, "repo", spec.Repo)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "AlreadyPermitted", "Team %s/%s already permitted access to repo %s", spec.Org, spec.TeamSlug, spec.Repo)

	cr.SetConditions(prv1.Available())

	return reconciler.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*teamRepov1alpha1.TeamRepo)
	if !ok {
		return errors.New(errNotTeamRepo)
	}

	cr.SetConditions(prv1.Creating())

	spec := cr.Spec.DeepCopy()

	err := e.ghCli.TeamRepo().Create(spec)
	if err != nil {
		return err
	}
	return nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) error {
	return e.Create(ctx, mg)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*teamRepov1alpha1.TeamRepo)
	if !ok {
		return errors.New(errNotTeamRepo)
	}

	cr.SetConditions(prv1.Deleting())

	spec := cr.Spec.DeepCopy()

	err := e.ghCli.TeamRepo().Delete(spec)
	if err != nil {
		return err
	}
	e.log.Debug("Team permission revoked", "org", spec.Org, "team", spec.TeamSlug, "owner", spec.Owner, "repo", spec.Repo)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "PermissionRevoked", "Revoked access for team %s/%s to repo %s", spec.Org, spec.TeamSlug, spec.Repo)

	return nil
}
