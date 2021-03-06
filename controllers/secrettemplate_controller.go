/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"github.com/ibm-garage-cloud/key-management-operator/service/generate_secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keymanagementv1 "github.com/ibm-garage-cloud/key-management-operator/api/v1"
)

// SecretTemplateReconciler reconciles a SecretTemplate object
type SecretTemplateReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=keymanagement.ibm,resources=secrettemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keymanagement.ibm,resources=secrettemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func (r *SecretTemplateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	r.Log.Info("Reconciling SecretTemplate")
	_ = r.Log.WithValues("secrettemplate", req.NamespacedName)

	// Fetch the SecretTemplate instance
	instance := &keymanagementv1.SecretTemplate{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	r.Log.Info("Got SecretTemplate: " + prettyPrint(instance))

	// Define a new Pod object
	secret := newSecretForCR(instance)

	r.Log.Info("Built a secret: " + prettyPrint(secret))

	// Set SecretTemplate instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	r.Log.Info("Looking for existing secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)

	// Check if this Pod already exists
	found := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Create(ctx, secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Secret created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		r.Log.Error(err, "Error getting secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	r.Log.Info("Skip reconcile: Secret already exists", "Secret.Namespace", found.Namespace, "Secret.Name", found.Name)
	return reconcile.Result{}, nil
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "  ")
	return string(s)
}

// newSecretForCR returns a busybox pod with the same name/namespace as the cr
func newSecretForCR(cr *keymanagementv1.SecretTemplate) *corev1.Secret {
	return generate_secret.GenerateSecret(cr)
}

func (r *SecretTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&keymanagementv1.SecretTemplate{}).
		Complete(r)
}
