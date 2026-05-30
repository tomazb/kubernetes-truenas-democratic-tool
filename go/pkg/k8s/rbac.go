package k8s

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type rbacRequirement struct {
	key             string
	group           string
	version         string
	resource        string
	verb            string
	namespace       string
	clusterScoped   bool
}

func (c *client) ValidateRBACPermissions(ctx context.Context) (*RBACValidationResult, error) {
	namespace := c.config.Namespace
	if namespace == "" {
		namespace = "default"
	}

	requirements := []rbacRequirement{
		{key: "persistentvolumes/list", resource: "persistentvolumes", verb: "list", clusterScoped: true},
		{key: "persistentvolumes/get", resource: "persistentvolumes", verb: "get", clusterScoped: true},
		{key: "persistentvolumeclaims/list", resource: "persistentvolumeclaims", verb: "list", namespace: namespace},
		{key: "persistentvolumeclaims/get", resource: "persistentvolumeclaims", verb: "get", namespace: namespace},
	}

	if c.snapshotClient != nil {
		requirements = append(requirements,
			rbacRequirement{
				key:       "volumesnapshots.snapshot.storage.k8s.io/list",
				group:     "snapshot.storage.k8s.io",
				version:   "v1",
				resource:  "volumesnapshots",
				verb:      "list",
				namespace: namespace,
			},
			rbacRequirement{
				key:       "volumesnapshots.snapshot.storage.k8s.io/get",
				group:     "snapshot.storage.k8s.io",
				version:   "v1",
				resource:  "volumesnapshots",
				verb:      "get",
				namespace: namespace,
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
		Namespace:              namespace,
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
	if !req.clusterScoped {
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
