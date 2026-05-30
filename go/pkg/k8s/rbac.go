package k8s

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type rbacRequirement struct {
	key           string
	group         string
	version       string
	resource      string
	verb          string
	namespace     string
	clusterScoped bool
}

func (c *client) ValidateRBACPermissions(ctx context.Context) (*RBACValidationResult, error) {
	scanAllNamespaces := c.config.Namespace == ""
	reportNamespace := c.config.Namespace

	requirements := []rbacRequirement{
		{key: "persistentvolumes/list", resource: "persistentvolumes", verb: "list", clusterScoped: true},
		{key: "persistentvolumes/get", resource: "persistentvolumes", verb: "get", clusterScoped: true},
	}

	pvcNamespace := c.config.Namespace
	pvcListKey := "persistentvolumeclaims/list"
	pvcGetKey := "persistentvolumeclaims/get"
	if scanAllNamespaces {
		pvcListKey = "persistentvolumeclaims/list (all namespaces)"
		pvcGetKey = "persistentvolumeclaims/get (all namespaces)"
	}

	requirements = append(requirements,
		rbacRequirement{key: pvcListKey, resource: "persistentvolumeclaims", verb: "list", namespace: pvcNamespace},
		rbacRequirement{key: pvcGetKey, resource: "persistentvolumeclaims", verb: "get", namespace: pvcNamespace},
	)

	if c.snapshotClient != nil {
		snapNS := c.config.Namespace
		snapListKey := "volumesnapshots.snapshot.storage.k8s.io/list"
		snapGetKey := "volumesnapshots.snapshot.storage.k8s.io/get"
		if scanAllNamespaces {
			snapListKey = "volumesnapshots.snapshot.storage.k8s.io/list (all namespaces)"
			snapGetKey = "volumesnapshots.snapshot.storage.k8s.io/get (all namespaces)"
		}
		requirements = append(requirements,
			rbacRequirement{
				key:       snapListKey,
				group:     "snapshot.storage.k8s.io",
				version:   "v1",
				resource:  "volumesnapshots",
				verb:      "list",
				namespace: snapNS,
			},
			rbacRequirement{
				key:       snapGetKey,
				group:     "snapshot.storage.k8s.io",
				version:   "v1",
				resource:  "volumesnapshots",
				verb:      "get",
				namespace: snapNS,
			},
		)
	}

	permissionChecks := make(map[string]bool, len(requirements))
	var missing []string
	var notes []string

	for _, req := range requirements {
		allowed, err := c.checkSelfSubjectAccess(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("rbac validation failed for %s: %w", req.key, err)
		}
		permissionChecks[req.key] = allowed
		if !allowed {
			missing = append(missing, req.key)
		}
	}

	if c.snapshotClient == nil {
		notes = append(notes, "skipped: volumesnapshots (snapshot client unavailable)")
	}

	return &RBACValidationResult{
		HasRequiredPermissions: len(missing) == 0,
		MissingPermissions:     append(missing, notes...),
		PermissionChecks:       permissionChecks,
		ServiceAccount:         "current",
		Namespace:              reportNamespace,
	}, nil
}

func (c *client) checkSelfSubjectAccess(ctx context.Context, req rbacRequirement) (bool, error) {
	version := req.version
	if version == "" {
		version = "v1"
	}

	attrs := &authorizationv1.ResourceAttributes{
		Verb:     req.verb,
		Group:    req.group,
		Version:  version,
		Resource: req.resource,
	}
	if !req.clusterScoped && req.namespace != "" {
		attrs.Namespace = req.namespace
	}

	review := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: attrs,
		},
	}

	result, err := c.clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(
		ctx,
		review,
		metav1.CreateOptions{},
	)
	if err != nil {
		return false, err
	}
	return result.Status.Allowed, nil
}
