package collaborator

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

	collaboratorv1alpha1 "github.com/krateoplatformops/github-provider/apis/collaborator/v1alpha1"
	"github.com/krateoplatformops/github-provider/internal/clients"
	"github.com/krateoplatformops/github-provider/internal/clients/github"
	"github.com/krateoplatformops/provider-runtime/pkg/ptr"
)

const (
	errNotCollaborator = "managed resource is not a collaborator custom resource"
)

// Setup adds a controller that reconciles Token managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := reconciler.ControllerName(collaboratorv1alpha1.CollaboratorGroupKind)

	log := o.Logger.WithValues("controller", name)

	recorder := mgr.GetEventRecorderFor(name)

	r := reconciler.NewReconciler(mgr,
		resource.ManagedKind(collaboratorv1alpha1.CollaboratorGroupVersionKind),
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
		For(&collaboratorv1alpha1.Collaborator{}).
		Complete(ratelimiter.New(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube     client.Client
	log      logging.Logger
	recorder record.EventRecorder
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (reconciler.ExternalClient, error) {
	cr, ok := mg.(*collaboratorv1alpha1.Collaborator)
	if !ok {
		return nil, errors.New(errNotCollaborator)
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
	cr, ok := mg.(*collaboratorv1alpha1.Collaborator)
	if !ok {
		return reconciler.ExternalObservation{}, errors.New(errNotCollaborator)
	}

	spec := cr.Spec.DeepCopy()

	ok, err := e.ghCli.Collaborators().Exists(spec)
	if err != nil {
		return reconciler.ExternalObservation{}, err
	}

	if ok {
		permission, err := e.ghCli.Collaborators().GetPermission(spec)
		if err != nil {
			return reconciler.ExternalObservation{}, err
		}

		cr.SetConditions(prv1.Available())
		if permission == spec.Permission {
			e.log.Debug("Collaborator already exists", "org", spec.Org, "repo", spec.Repo, "username", spec.Username)
			e.rec.Eventf(cr, corev1.EventTypeNormal, "AlreadyExists", "Collaborator '%s/%s/%s' already exists", spec.Org, spec.Repo, spec.Username)
			return reconciler.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			}, nil
		} else {
			return reconciler.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: false,
			}, nil
		}
	}

	e.log.Debug("Collaborator does not exists", "org", spec.Org, "repo", spec.Repo, "username", spec.Username)

	return reconciler.ExternalObservation{
		ResourceExists:   false,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*collaboratorv1alpha1.Collaborator)
	if !ok {
		return errors.New(errNotCollaborator)
	}

	cr.SetConditions(prv1.Creating())

	spec := cr.Spec.DeepCopy()

	err := e.ghCli.Collaborators().Create(spec)
	if err != nil {
		return err
	}
	e.log.Debug("Collaborator created", "org", spec.Org, "repo", spec.Repo, "username", spec.Username)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "CollaboratorCreated", "Collaborator '%s/%s/%s' created", spec.Org, spec.Repo, spec.Username)

	return nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) error {
	return e.Create(ctx, mg)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*collaboratorv1alpha1.Collaborator)
	if !ok {
		return errors.New(errNotCollaborator)
	}

	cr.SetConditions(prv1.Deleting())

	spec := cr.Spec.DeepCopy()

	err := e.ghCli.Collaborators().Delete(spec)
	if err != nil {
		return err
	}
	e.log.Debug("Collaborator deleted", "org", spec.Org, "repo", spec.Repo, "username", spec.Username)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RepDeleted", "Collaborator '%s/%s/%s' deleted", spec.Org, spec.Repo, spec.Username)

	return nil
}
