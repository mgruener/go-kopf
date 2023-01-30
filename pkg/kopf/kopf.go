package kopf

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type EventHandler func(*unstructured.Unstructured, *unstructured.Unstructured, logr.Logger) error

type Handler interface {
	Create(group string, version string, kind string, fn EventHandler) Handler
	Update(group string, version string, kind string, fn EventHandler) Handler
	Delete(group string, version string, kind string, fn EventHandler) Handler
	start() bool
}

type handler struct {
	log    logr.Logger
	mgr    manager.Manager
	client client.Client
}

var On, _ = New("")

func init() {
	logf.SetLogger(zap.New())
}

func New(namespace string) (Handler, error) {
	log := logf.Log.WithName("handler")
	opts := manager.Options{}
	if namespace != "" {
		opts.Namespace = namespace
	}
	mgr, err := manager.New(config.GetConfigOrDie(), opts)
	if err != nil {
		log.Error(err, "could not create manager")
		return nil, err
	}
	return &handler{
		log:    log,
		mgr:    mgr,
		client: mgr.GetClient(),
	}, nil
}

func ExecuteOrDie(h Handler) {
	hdl := On
	if h != nil {
		hdl = h
	}

	if ok := hdl.start(); !ok {
		os.Exit(1)
	}
}

func (h *handler) createController(filter predicate.Predicate, group string, version string, kind string, fn EventHandler) error {
	res := &unstructured.Unstructured{}
	res.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	})

	err := builder.
		ControllerManagedBy(h.mgr).
		For(res).
		WithEventFilter(filter).
		Complete(reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
			res := &unstructured.Unstructured{}
			res.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   group,
				Version: version,
				Kind:    kind,
			})
			err := h.client.Get(ctx, req.NamespacedName, res)
			if err != nil {
				return reconcile.Result{}, client.IgnoreNotFound(err)
			}

			patch := &unstructured.Unstructured{}
			patch.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   group,
				Version: version,
				Kind:    kind,
			})
			patch.SetName(req.Name)
			patch.SetNamespace(req.Namespace)
			err = fn(res, patch, h.log.WithName("handler-execute"))
			if err != nil {
				return reconcile.Result{}, err
			}

			err = h.client.Patch(ctx, patch, client.Apply, client.ForceOwnership, client.FieldOwner("mgruener-test"))
			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil
		}))
	if err != nil {
		h.log.Error(err, "could not create controller")
		return err
	}

	return nil
}

func (h *handler) Create(group string, version string, kind string, fn EventHandler) Handler {
	err := h.createController(creationPredicate(), group, version, kind, fn)
	if err != nil {
		h.log.Error(err, "could not create controller")
	}
	return h
}

func (h *handler) Update(group string, version string, kind string, fn EventHandler) Handler {
	err := h.createController(updatePredicate(), group, version, kind, fn)
	if err != nil {
		h.log.Error(err, "could not create controller")
	}
	return h
}

func (h *handler) Delete(group string, version string, kind string, fn EventHandler) Handler {
	err := h.createController(deletionPredicate(), group, version, kind, fn)
	if err != nil {
		h.log.Error(err, "could not create controller")
	}
	return h
}

func (h *handler) start() bool {
	if err := h.mgr.Start(signals.SetupSignalHandler()); err != nil {
		h.log.Error(err, "could not start manager")
		return false
	}
	return true
}
