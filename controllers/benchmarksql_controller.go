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
	"fmt"
	"github.com/go-logr/logr"
	kbatch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	pingcapv1 "test-operator/api/v1"
	"time"
)

// BenchmarkSQLReconciler reconciles a BenchmarkSQL object
type BenchmarkSQLReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pingcap.pingcap.com,resources=benchmarksqls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pingcap.pingcap.com,resources=benchmarksqls/status,verbs=get;update;patch

func (r *BenchmarkSQLReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("benchmarksql", req.NamespacedName)
	// your logic here
	//ctrl.CreateOrUpdate()
	var testRequest pingcapv1.BenchmarkSQL
	if err := r.Get(ctx, req.NamespacedName, &testRequest); err != nil {
		log.Error(err, "unable to fetch testRequest")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var childJobs kbatch.JobList
	if err := r.List(ctx, &childJobs); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}
	constructJob := func(request pingcapv1.BenchmarkSQL) (*kbatch.Job, error) {
		// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
		name := fmt.Sprintf("test")

		job := &kbatch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        name,
				Namespace:   "default",
			},
			Spec: kbatch.JobSpec{
				Parallelism:           nil,
				Completions:           nil,
				ActiveDeadlineSeconds: nil,
				BackoffLimit:          nil,
				Selector:              nil,
				ManualSelector:        nil,
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: v1.PodSpec{
						RestartPolicy: "OnFailure",
						Containers: []v1.Container{{
							Name:  "test",
							Image: "longfangsong/benchmarksql:1582222183",
							Env: []v1.EnvVar{
								{Name: "CONN", Value: request.Spec.Conn},
								{Name: "WAREHOUSES", Value: strconv.Itoa(int(request.Spec.Warehouses))},
								{Name: "LOADWORKERS", Value: strconv.Itoa(int(request.Spec.LoadWorkers))},
								{Name: "TERMINALS", Value: strconv.Itoa(int(request.Spec.Terminals))},
							},
						}},
					},
				},
				TTLSecondsAfterFinished: nil,
			},
		}
		job.Annotations["createTime"] = time.Now().Format(time.RFC3339)
		return job, nil
	}
	job, err := constructJob(testRequest)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for CronJob", "job", job)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *BenchmarkSQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pingcapv1.BenchmarkSQL{}).
		Complete(r)
}
