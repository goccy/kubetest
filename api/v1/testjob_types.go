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

	Git      GitSpec                `json:"git,omitempty"`
	Template corev1.PodTemplateSpec `json:"template"`
	// Log extend parameter to output log.
	Log LogSpec `json:"log,omitempty"`
	// Prepare steps before testing
	Prepare PrepareSpec `json:"prepare,omitempty"`
	// Distributed testing parameter
	DistributedTest *DistributedTestSpec `json:"distributedTest,omitempty"`
}

type GitSpec struct {
	// Image name for clone and checkout by git protocol.
	Image string `json:"image,omitempty"`
	// Checkout whether checkout repository before testing ( default: true ).
	Checkout *bool `json:"checkout,omitempty"`
	// Repository name.
	Repo string `json:"repo"`
	// Branch name.
	Branch string `json:"branch,omitempty"`
	// Revision.
	Rev string `json:"rev,omitempty"`
	// OAuth token to fetch private repository
	Token *TestJobToken `json:"token,omitempty"`
	// CheckoutDir ( default: /git/workspace )
	CheckoutDir string `json:"checkoutDir,omitempty"`
	// Merge base branch
	Merge GitMergeSpec `json:"merge,omitempty"`
}

type GitMergeSpec struct {
	// Base branch name
	Base string `json:"base"`
}

type LogSpec struct {
	ExtParam map[string]string `json:"extParam"`
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
	Image    string            `json:"image"`
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
	// ContainerName container name for running test in template.spec.containers.
	ContainerName string `json:"containerName"`
	// MaxContainersPerPod maximum number of container per pod.
	MaxContainersPerPod int `json:"maxContainersPerPod"`
	// Output testing list to stdout
	List DistributedTestListSpec `json:"list"`
	// Restart testing for failed tests
	Retest DistributedTestRetestSpec `json:"retest,omitempty"`
	// CacheSpec for making cache before testing
	Cache []CacheSpec `json:"cache,omitempty"`
}

type DistributedTestListSpec struct {
	Command []string `json:"command"`
	Args    []string `json:"args"`
	// Delimiter for testing list ( default: new line character ( \n ) )
	Delimiter string `json:"delimiter,omitempty"`
	// Test name pattern ( enable use regular expression )
	Pattern string `json:"pattern,omitempty"`
}

type DistributedTestRetestSpec struct {
	// Enabled restart testing for failed tests
	Enabled bool `json:"enabled"`
	// Delimiter for testing list of retest ( default: white space )
	Delimiter string `json:"delimiter,omitempty"`
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
