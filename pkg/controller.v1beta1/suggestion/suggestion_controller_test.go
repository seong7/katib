/*
Copyright 2021 The Kubeflow Authors.

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

package suggestion

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	commonv1beta1 "github.com/kubeflow/katib/pkg/apis/controller/common/v1beta1"
	experimentsv1beta1 "github.com/kubeflow/katib/pkg/apis/controller/experiments/v1beta1"
	suggestionsv1beta1 "github.com/kubeflow/katib/pkg/apis/controller/suggestions/v1beta1"
	trialsv1beta1 "github.com/kubeflow/katib/pkg/apis/controller/trials/v1beta1"
	"github.com/kubeflow/katib/pkg/controller.v1beta1/consts"
	"github.com/kubeflow/katib/pkg/controller.v1beta1/suggestion/composer"
	"github.com/kubeflow/katib/pkg/controller.v1beta1/util"
	suggestionclientmock "github.com/kubeflow/katib/pkg/mock/v1beta1/suggestion/suggestionclient"
	"github.com/kubeflow/katib/pkg/util/v1beta1/katibconfig"
)

const (
	suggestionName  = "test-suggestion"
	resourceName    = "test-suggestion-random"
	namespace       = "kubeflow"
	suggestionImage = "test-image"
	katibConfigName = "katib-config"
	timeout         = time.Second * 40
)

func init() {
	logf.SetLogger(zap.New())
}

func TestAdd(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Test - Try to add suggestion controller to the manager
	g.Expect(Add(mgr)).NotTo(gomega.HaveOccurred())
}

func TestReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSuggestionClient := suggestionclientmock.NewMockSuggestionClient(mockCtrl)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c := mgr.GetClient()

	r := &ReconcileSuggestion{
		Client:           mgr.GetClient(),
		scheme:           mgr.GetScheme(),
		SuggestionClient: mockSuggestionClient,
		Composer:         composer.New(mgr),
		recorder:         mgr.GetEventRecorderFor(ControllerName),
	}

	recFn := SetupTestReconcile(r)
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	// Start test manager.
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		g.Expect(mgr.Start(context.TODO())).NotTo(gomega.HaveOccurred())
	}()

	mockSuggestionClient.EXPECT().ValidateAlgorithmSettings(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSuggestionClient.EXPECT().SyncAssignments(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	instance := newFakeInstance()

	trial := &trialsv1beta1.Trial{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "trial",
			Namespace: namespace,
			Labels: map[string]string{
				consts.LabelExperimentName: suggestionName,
			},
		},
	}

	experiment := &experimentsv1beta1.Experiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      suggestionName,
			Namespace: namespace,
		},
	}

	configMap := newKatibConfigMapInstance()

	// Create kubeflow namespace.
	kubeflowNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	g.Expect(c.Create(context.TODO(), kubeflowNS)).NotTo(gomega.HaveOccurred())
	// Test 1 - Early stopping suggestion run
	// Create ConfigMap with suggestion and early stopping data.
	g.Expect(c.Create(context.TODO(), configMap)).NotTo(gomega.HaveOccurred())
	// Create the suggestion
	g.Expect(c.Create(context.TODO(), instance)).NotTo(gomega.HaveOccurred())
	// Create experiment
	g.Expect(c.Create(context.TODO(), experiment)).NotTo(gomega.HaveOccurred())
	// Create trial
	g.Expect(c.Create(context.TODO(), trial)).NotTo(gomega.HaveOccurred())

	suggestionDeploy := &appsv1.Deployment{}

	// Expect that deployment with appropriate name and image is created
	g.Eventually(func() string {
		if err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: resourceName}, suggestionDeploy); err != nil {
			return ""
		}
		return suggestionDeploy.Spec.Template.Spec.Containers[0].Image
	}, timeout).Should(gomega.Equal(suggestionImage))

	// Expect that service with appropriate name is created
	g.Eventually(func() error {
		return c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: resourceName}, &corev1.Service{})
	}, timeout).Should(gomega.Succeed())

	// Expect that PVC with appropriate name is created
	g.Eventually(func() error {
		return c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: resourceName}, &corev1.PersistentVolumeClaim{})
	}, timeout).Should(gomega.Succeed())

	rbacName := util.GetSuggestionRBACName(instance)

	// Expect that serviceAccount with appropriate name is created
	suggestionServiceAccount := &corev1.ServiceAccount{}
	g.Eventually(func() error {
		return c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: rbacName}, suggestionServiceAccount)
	}, timeout).Should(gomega.Succeed())

	// Expect that Role with appropriate name is created
	suggestionRole := &rbacv1.Role{}
	g.Eventually(func() error {
		return c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: rbacName}, suggestionRole)
	}, timeout).Should(gomega.Succeed())

	// Expect that RoleBinding with appropriate name is created
	suggestionRoleBinding := &rbacv1.RoleBinding{}
	g.Eventually(func() error {
		return c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: rbacName}, suggestionRoleBinding)
	}, timeout).Should(gomega.Succeed())

	// Manually change ready deployment status
	suggestionDeploy.Status = appsv1.DeploymentStatus{
		Conditions: []appsv1.DeploymentCondition{
			{
				Type:   appsv1.DeploymentAvailable,
				Status: corev1.ConditionTrue,
			},
		},
	}

	g.Expect(c.Status().Update(context.TODO(), suggestionDeploy)).NotTo(gomega.HaveOccurred())

	// Expect that suggestion status is running
	suggestion := &suggestionsv1beta1.Suggestion{}
	g.Eventually(func() bool {
		if err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: suggestionName}, suggestion); err != nil {
			return false
		}
		return suggestion.IsRunning()
	}, timeout).Should(gomega.BeTrue())

	// Manually update suggestion status to succeeded
	suggestion.MarkSuggestionStatusSucceeded("test-reason", "test-message")
	g.Expect(c.Status().Update(context.TODO(), suggestion)).NotTo(gomega.HaveOccurred())

	// Expect that suggestion status is succeeded, is not running and deployment is not ready
	g.Eventually(func() bool {
		if err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: suggestionName}, suggestion); err != nil {
			return false
		}
		return !suggestion.IsRunning() && !suggestion.IsDeploymentReady() && suggestion.IsSucceeded()
	}, timeout).Should(gomega.BeTrue())

	// Expect that deployment and service is deleted
	g.Eventually(func() bool {
		return errors.IsNotFound(c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: resourceName}, &appsv1.Deployment{})) &&
			errors.IsNotFound(c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: resourceName}, &corev1.Service{}))
	}, timeout).Should(gomega.BeTrue())

	// Delete the suggestion
	g.Expect(c.Delete(context.TODO(), instance)).NotTo(gomega.HaveOccurred())

	// Expect that suggestion is deleted
	g.Eventually(func() bool {
		return errors.IsNotFound(c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: suggestionName}, &suggestionsv1beta1.Suggestion{}))
	}, timeout).Should(gomega.BeTrue())

	oldS := &suggestionsv1beta1.Suggestion{
		Status: suggestionsv1beta1.SuggestionStatus{
			SuggestionCount: 1,
			Conditions: []suggestionsv1beta1.SuggestionCondition{
				{
					Type:   suggestionsv1beta1.SuggestionFailed,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	newS := &suggestionsv1beta1.Suggestion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-test",
			Namespace: namespace,
		},
		Status: suggestionsv1beta1.SuggestionStatus{
			SuggestionCount: 1,
			Conditions: []suggestionsv1beta1.SuggestionCondition{
				{
					Type:   suggestionsv1beta1.SuggestionFailed,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	// Test 2 - Update status for empty experiment
	g.Expect(r.updateStatus(&suggestionsv1beta1.Suggestion{}, oldS)).To(gomega.HaveOccurred())

	// Test 3 - Update status condition
	g.Expect(r.updateStatusCondition(newS, oldS)).NotTo(gomega.HaveOccurred())

	// Test 4 - Update status condition for empty experiment
	g.Expect(r.updateStatusCondition(&suggestionsv1beta1.Suggestion{}, oldS)).To(gomega.HaveOccurred())

}

func newFakeInstance() *suggestionsv1beta1.Suggestion {
	earlyStoppingSpec := &commonv1beta1.EarlyStoppingSpec{
		AlgorithmName: "median-stop",
		AlgorithmSettings: []commonv1beta1.EarlyStoppingSetting{
			{
				Name:  "min_trials_required",
				Value: "3",
			},
			{
				Name:  "start_step",
				Value: "5",
			},
		},
	}
	return &suggestionsv1beta1.Suggestion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      suggestionName,
			Namespace: namespace,
		},
		Spec: suggestionsv1beta1.SuggestionSpec{
			Requests: 1,
			Algorithm: &commonv1beta1.AlgorithmSpec{
				AlgorithmName: "random",
			},
			ResumePolicy:  experimentsv1beta1.FromVolume,
			EarlyStopping: earlyStoppingSpec,
		},
	}
}

func newKatibConfigMapInstance() *corev1.ConfigMap {
	// Create suggestion config
	suggestionConfig := map[string]katibconfig.SuggestionConfig{
		"random": {
			Image: suggestionImage,
		},
	}
	bSuggestionConfig, _ := json.Marshal(suggestionConfig)

	// Create early stopping config
	earlyStoppingConfig := map[string]katibconfig.EarlyStoppingConfig{
		"median-stop": {
			Image:           "test-image",
			ImagePullPolicy: corev1.PullAlways,
		},
	}
	bEarlyStoppingConfig, _ := json.Marshal(earlyStoppingConfig)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      katibConfigName,
			Namespace: namespace,
		},
		Data: map[string]string{
			consts.LabelSuggestionTag:    string(bSuggestionConfig),
			consts.LabelEarlyStoppingTag: string(bEarlyStoppingConfig),
		},
	}
}
