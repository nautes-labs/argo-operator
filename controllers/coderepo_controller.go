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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	resourcev1alpha1 "argo-operator/api/v1alpha1"
)

// CodeRepoReconciler reconciles a CodeRepo object
type CodeRepoReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=resource.nautes.io,resources=coderepoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=resource.nautes.io,resources=coderepoes/status,verbs=get;update;patch
func (r *CodeRepoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("coderepo", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *CodeRepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcev1alpha1.CodeRepo{}).
		Complete(r)
}
