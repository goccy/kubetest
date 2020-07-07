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
	coreerrors "errors"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubetestv1 "github.com/goccy/kubetest/api/v1"
)

// TestJobReconciler reconciles a TestJob object
type TestJobReconciler struct {
	client.Client
	ClientSet *kubernetes.Clientset
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubetest.io,resources=testjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubetest.io,resources=testjobs/status,verbs=get;update;patch

func (r *TestJobReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, e error) {
	ctx := context.Background()
	_ = r.Log.WithValues("testjob", req.NamespacedName)

	var job kubetestv1.TestJob
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	defer func() {
		if err := r.Delete(ctx, &job); err != nil {
			if e != nil {
				e = errors.NewInternalError(coreerrors.New(strings.Join([]string{
					e.Error(), err.Error(),
				}, "\n")))
			} else {
				e = err
			}
		}
	}()
	if err := kubetestv1.NewTestJobRunner(r.ClientSet).Run(ctx, job); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *TestJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubetestv1.TestJob{}).
		Complete(r)
}
