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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestJobSpec defines the desired state of TestJob
type TestJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Image name for clone and checkout by git protocol.
	GitImage string `json:"gitImage,omitempty"`
	// Checkout whether checkout repository before testing ( default: true ).
	Checkout *bool `json:"checkout,omitempty"`
	// Image name.
	Image string `json:"image"`
	// Repository name.
	Repo string `json:"repo"`
	// Command for testing.
	Command Command `json:"command"`
	// Workdir ( default: /git/workspace )
	Workdir string `json:"workdir,omitempty"`
	// Branch name.
	Branch string `json:"branch,omitempty"`
	// Revision.
	Rev string `json:"rev,omitempty"`
	// OAuth token to fetch private repository
	Token *TestJobToken `json:"token,omitempty"`
	// List of environment variables to set in the container.
	Env []corev1.EnvVar `json:"env,omiempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this TestJobSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Prepare steps before testing
	Prepare PrepareSpec `json:"prepare,omitempty"`
	// Distributed testing parameter
	DistributedTest *DistributedTestSpec `json:"distributedTest,omitempty"`
}

type Command string

type TestJobToken struct {
	SecretKeyRef TestJobSecretKeyRef `json:"secretKeyRef"`
}

type TestJobSecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type PrepareSpec struct {
	// Checkout whether checkout repository before testing ( default: true ).
	Checkout *bool             `json:"checkout,omitempty"`
	Image    string            `json:"image,omitempty"`
	Steps    []PrepareStepSpec `json:"steps"`
}

type PrepareStepSpec struct {
	Name    string  `json:"name"`
	Image   string  `json:"image,omitempty"`
	Command Command `json:"command"`
	// Workdir ( default: /git/workspace )
	Workdir string          `json:"workdir,omitempty"`
	Env     []corev1.EnvVar `json:"env,omiempty"`
}

type DistributedTestSpec struct {
	// Output testing list to stdout
	ListCommand Command `json:"listCommand"`
	// Delimiter for testing list ( default: new line character ( \n ) )
	ListDelimiter string `json:"listDelimiter,omitempty"`
	// Test name pattern ( enable use regular expression )
	Pattern string `json:"pattern,omitempty"`
	// MaxContainersPerPod maximum number of container per pod.
	MaxContainersPerPod int `json:"maxContainersPerPod"`
	// Restart testing for failed tests
	Retest bool `json:"retest"`
	// Delimiter for testing list of retest ( default: white space )
	RetestDelimiter string `json:"retestDelimiter,omitempty"`
	// CacheSpec for making cache before testing
	Cache []CacheSpec `json:"cache,omitempty"`
}

type CacheSpec struct {
	// Name cache identifier
	Name string `json:"name"`
	// Command for making cache
	Command Command `json:"command"`
	// Path specify mount path
	Path string `json:"path"`
}

// TestJobStatus defines the observed state of TestJob
type TestJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Whether the testjob is running
	Running bool `json:"running,omitempty"`
}

// +kubebuilder:object:root=true

// TestJob is the Schema for the testjobs API
type TestJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestJobSpec   `json:"spec,omitempty"`
	Status TestJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TestJobList contains a list of TestJob
type TestJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestJob{}, &TestJobList{})
}
