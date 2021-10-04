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
	kubetestv1 "github.com/goccy/kubetest/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestJobReconciler reconciles a TestJob object
type TestJobReconciler struct {
	client.Client
	Config    *rest.Config
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
	if job.Status.Running {
		return ctrl.Result{}, nil
	}

	job.Status.Running = true
	if err := r.Update(ctx, &job); err != nil {
		return ctrl.Result{}, err
	}

	go func() {
		if err := r.runTestJob(ctx, job); err != nil {
			e = err
		}
	}()
	return ctrl.Result{}, nil
}

func (r *TestJobReconciler) runTestJob(ctx context.Context, job kubetestv1.TestJob) (e error) {
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
	if _, err := kubetestv1.NewRunner(r.Config, kubetestv1.RunModeKubernetes).Run(ctx, job); err != nil {
		return err
	}
	return nil
}

func (r *TestJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubetestv1.TestJob{}).
		Complete(r)
}
