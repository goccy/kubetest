# kubetest

[![PkgGoDev](https://pkg.go.dev/badge/github.com/goccy/kubetest)](https://pkg.go.dev/github.com/goccy/kubetest)
![Go](https://github.com/goccy/kubetest/workflows/test/badge.svg)
[![codecov](https://codecov.io/gh/goccy/kubetest/branch/master/graph/badge.svg)](https://codecov.io/gh/goccy/kubetest)


A CLI for efficient use of Kubernetes Cluster resources for distributed processing of time-consuming task processing.

This tool is developed based on the following concept.

- Distributed processing: divide time-consuming tasks based on certain rules, and efficient use of cluster resources by processing each task using different pods
- One container per task: since the divided tasks are processed in different containers, they are less affected by the processing of different tasks.

# Installation

```bash
$ go install github.com/goccy/kubetest/cmd/kubetest
```

# How to use

```
Usage:
  kubetest [OPTIONS]

Application Options:
  -n, --namespace=  specify namespace (default: default)
      --in-cluster  specify whether in cluster
  -c, --config=     specify local kubeconfig path. ( default: $HOME/.kube/config )
      --list=       specify path to get the list for test
      --log-level=  specify log level (debug/info/warn/error)
      --dry-run     specify dry run mode
      --template=   specify template parameter for testjob file

Help Options:
  -h, --help        Show this help message
```

## 1. Run simple task

First, We will introduce a sample that performs the simplest task processing.

Describe the manifest file of task processing as follows and execute it by passing it as an argument of kubetest CLI.

If you've already written a Kubernetes Job, you've probably noticed that the simplest example is the same as using a Kubernetes Job :)

- _examples/simple.yaml

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: simple-testjob
  namespace: default
spec:
  template:
    metadata:
      generateName: simple-testjob-
    spec:
      containers:
        - name: test
          image: alpine
          workingDir: /go/src
          command:
            - echo
          args:
            - "hello"
```

### Run CLI with manifest

```console
kubetest --log-level=info _examples/simple.yaml
```

### Output

The content consists of the following elements.

- Command
- Log of command
- Elapsed time of running command
- Summary of all tasks ( JSON format )

```console
echo hello
hello

[INFO] elapsed time: 0.184144 sec.
{
  "details": [
    {
      "elapsedTimeSec": 0,
      "name": "test",
      "status": "success"
    }
  ],
  "elapsedTimeSec": 10,
  "failureNum": 0,
  "startedAt": "2021-10-05T07:36:07.893339674Z",
  "status": "success",
  "successNum": 1,
  "totalNum": 1
}
```

## 2. Run task with public repository

You'll want to use versioned data and code by `git` when processing tasks.
In kubetest, you can write the repository definition in `repos` and specify it in ` volumes`. The repository defined in `volumes` can be mounted in any container by using `volumeMounts` like `emptyDir` .

- _examples/public-repo.yaml

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: public-repo-testjob
  namespace: default
spec:
  repos:
    - name: kubetest-repo
      value:
        url: https://github.com/goccy/kubetest.git
        branch: master
  template:
    metadata:
      generateName: public-repo-testjob-
    spec:
      containers:
        - name: test
          image: alpine
          workingDir: /work
          command:
            - ls
          args:
            - README.md
          volumeMounts:
            - name: repo
              mountPath: /work
      volumes:
        - name: repo
          repo:
            name: kubetest-repo
```

### Run CLI with manifest

```console
kubetest --log-level=info _examples/public-repo.yaml
```
### Output

```console
[INFO] clone repository: https://github.com/goccy/kubetest.git
ls README.md
README.md

[INFO] elapsed time: 0.050960 sec.
{
  "details": [
    {
      "elapsedTimeSec": 0,
      "name": "test",
      "status": "success"
    }
  ],
  "elapsedTimeSec": 14,
  "failureNum": 0,
  "startedAt": "2021-10-05T07:41:51.54612701Z",
  "status": "success",
  "successNum": 1,
  "totalNum": 1
}
```

## 3. Run task with private repository

You can also use a private repository with kubetest.
You can define GitHub personal token or token by GitHub App in `tokens` .
GitHub persoanl token data or GitHub App key data are managed by Kubernetes Secrets.
`kubetest` get token by referring to them.
By describing the name of the token to be used in the definition of private repository in the form of `token: github-app-token`, the repository will be cloned using that token.

In addition, the token can be mounted on any path using `volumeMounts` by writing the following in `volumes`. By combining this with `prestep`, which will be described later, you can devise so that you do not need a token when processing the main task. This makes task processing more secure.

```yaml
volumes:
- name: token-volume
  token:
    name: <defined token name>
```

- _examples/private-repo.yaml

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: private-repo-testjob
  namespace: default
spec:
  tokens:
    - name: github-app-token
      value:
        githubApp:
          organization: goccy
          appId: 134426
          keyFile:
            name: github-app
            key: private-key
  repos:
    - name: kubetest-repo
      value:
        # specify the private repository url
        url: https://github.com/goccy/kubetest.git
        branch: master
        token: github-app-token
  template:
    metadata:
      generateName: private-repo-testjob-
    spec:
      containers:
        - name: test
          image: alpine
          workingDir: /work
          command:
            - ls
          args:
            - README.md
          volumeMounts:
            - name: repo
              mountPath: /work
      volumes:
        - name: repo
          repo:
            name: kubetest-repo
```

### Output

```console
[INFO] clone repository: https://github.com/goccy/kubetest.git
ls README.md
README.md

[INFO] elapsed time: 0.055823 sec.
{
  "details": [
    {
      "elapsedTimeSec": 0,
      "name": "test",
      "status": "success"
    }
  ],
  "elapsedTimeSec": 14,
  "failureNum": 0,
  "startedAt": "2021-10-05T07:43:34.607701724Z",
  "status": "success",
  "successNum": 1,
  "totalNum": 1
}
```

## 4. Run task with prestep

If there is any pre-processing required before performing the main task processing, you can define it in `preSteps` and pass only the processing result to the subsequent tasks.
By making effective use of this step, the pre-processing required for each distributed process can be limited to one time, and the resources of the cluster can be used efficiently.
Since multiple preSteps can be defined and executed in order, the result of the previous step can be used to execute the next step.

The artifacts created by `preStep` can be reused in the subsequent task processing by describing the container name and path where the artifacts exists in `artifacts` spec.

If you want to use the already created artifacts, you can write the name of the defined artifact in `volumes` as follows. As with the repository, you can use `volumeMounts` to mount it on any path.

```yaml
volumes:
- name: artifact-volume
  artifact:
    name: <defined artifact name>
```

- _examples/prestep.yaml

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: prestep-testjob
  namespace: default
spec:
  repos:
    - name: kubetest-repo
      value:
        url: https://github.com/goccy/kubetest.git
        branch: master
  preSteps:
    - name: create-awesome-stuff
      template:
        metadata:
          generateName: create-awesome-stuff-
        spec:
          artifacts:
            - name: awesome-stuff
              container:
                name: create-awesome-stuff-container
                path: /work/awesome-stuff
          containers:
            - name: create-awesome-stuff-container
              image: alpine
              workingDir: /work
              command: ["sh", "-c"]
              args:
                - |
                  echo "AWESOME!!!" > awesome-stuff
              volumeMounts:
                - name: repo
                  mountPath: /work
          volumes:
            - name: repo
              repo:
                name: kubetest-repo
  template:
    metadata:
      generateName: prestep-testjob-
    spec:
      containers:
        - name: test
          image: alpine
          workingDir: /work
          command:
            - cat
          args:
            - awesome-stuff
          volumeMounts:
            - name: repo
              mountPath: /work
            - name: prestep-artifact
              mountPath: /work/awesome-stuff
      volumes:
        - name: repo
          repo:
            name: kubetest-repo
        - name: prestep-artifact
          artifact:
            name: awesome-stuff
```

### Output

```console
[INFO] clone repository: https://github.com/goccy/kubetest.git
[INFO] run prestep: create-awesome-stuff
sh -c echo "AWESOME!!!" > awesome-stuff

[INFO] elapsed time: 0.062056 sec.
cat awesome-stuff
AWESOME!!!

[INFO] elapsed time: 0.053780 sec.
{
  "details": [
    {
      "elapsedTimeSec": 0,
      "name": "test",
      "status": "success"
    }
  ],
  "elapsedTimeSec": 24,
  "failureNum": 0,
  "startedAt": "2021-10-05T08:00:43.033808187Z",
  "status": "success",
  "successNum": 1,
  "totalNum": 1
}
```

## 5. Run distributed task with static keys

Describes the distributed processing, which is the main feature of kubetest.
Distributed processing is realized by defining a **`distributed key`** and passing that value as an environment variable to different tasks.
The `distributed key` can be determined statically or dynamically.
In the following, we will explain using the static determination pattern.

- _examples/strategy-static.yaml

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: strategy-static-testjob
  namespace: default
spec:
  strategy:
    key:
      env: TASK_KEY
      source:
        static:
          - TASK_KEY_1
          - TASK_KEY_2
          - TASK_KEY_3
    scheduler:
      maxContainersPerPod: 10
      maxConcurrentNumPerPod: 10
  template:
    metadata:
      generateName: strategy-static-testjob-
    spec:
      containers:
        - name: test
          image: alpine
          workingDir: /work
          command:
            - echo
          args:
            - $TASK_KEY
```

Describe the definition of distributed execution under `strategy` as described above.
`key` defines the name of the environment variable to be referenced as the distribution key and the value of the distribution key itself.

In this example, if you refer to the environment variable named `TASK_KEY`, you can get one of the values ​​from `TASK_KEY_1` to `TASK_KEY_3`.
After that, define a command that uses the value of this environment variable in `spec.template.spec.containers[].command`.

In `strategy.scheduler`, define the resources such as `Pod` and `Container` used for distributed execution.
In this example, `maxContainersPerPod` is `10`, which means that up to `10` containers can be launched per Pod, and `maxConcurrentNumPerPod` is also `10`, which means that `10` containers can process tasks at the same time per Pod.
Since the number of distributed keys is `3`, only one Pod will be launched, but if the number of distributed keys exceeds `10`, two Pods will be launched and processed.
Similarly, if you set the number of `maxContainersPerPod` to `1`, only one container will be started per Pod, so three Pods will be started and processed.

### Output

```console
[INFO] found 3 static keys to start distributed task
[TASK_KEY:TASK_KEY_1] echo $TASK_KEY
TASK_KEY_1

[INFO] elapsed time: 0.194488 sec.
[INFO] 1/3 (33.333336%) finished.
[TASK_KEY:TASK_KEY_3] echo $TASK_KEY
TASK_KEY_3

[INFO] elapsed time: 0.194521 sec.
[INFO] 2/3 (66.666672%) finished.
[TASK_KEY:TASK_KEY_2] echo $TASK_KEY
TASK_KEY_2

[INFO] elapsed time: 0.304037 sec.
[INFO] 3/3 (100.000000%) finished.
{
  "details": [
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_1",
      "status": "success"
    },
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_3",
      "status": "success"
    },
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_2",
      "status": "success"
    }
  ],
  "elapsedTimeSec": 13,
  "failureNum": 0,
  "startedAt": "2021-10-05T08:23:11.568491828Z",
  "status": "success",
  "successNum": 3,
  "totalNum": 3
}
```

## 6. Run distributed task with dynamic keys

Use `strategy.key.source.dynamic` to create a distributed key dynamically.
The `distributed key` is the output result of the command defined here divided by the line feed character. ( There is also a way of splitting and a method of filtering unnecessary output results )

- _examples/strategy-dynamic.yaml

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: strategy-dynamic-testjob
  namespace: default
spec:
  strategy:
    key:
      env: TASK_KEY
      source:
        dynamic:
          template:
            metadata:
              generateName: strategy-dynamic-keys-
            spec:
              containers:
                - name: key
                  image: alpine
                  command: ["sh", "-c"]
                  args:
                    - |
                      echo -n "
                      TASK_KEY_1
                      TASK_KEY_2
                      TASK_KEY_3
                      TASK_KEY_4"
    scheduler:
      maxContainersPerPod: 10
      maxConcurrentNumPerPod: 10
  template:
    metadata:
      generateName: strategy-dynamic-testjob-
    spec:
      containers:
        - name: test
          image: alpine
          workingDir: /work
          command:
            - echo
          args:
            - $TASK_KEY
```

### Output

```console
sh -c echo -n "
TASK_KEY_1
TASK_KEY_2
TASK_KEY_3
TASK_KEY_4"


TASK_KEY_1
TASK_KEY_2
TASK_KEY_3
TASK_KEY_4
[INFO] elapsed time: 0.103151 sec.
[INFO] found 4 dynamic keys to start distributed task. elapsed time 0.103151 sec
[TASK_KEY:TASK_KEY_2] echo $TASK_KEY
TASK_KEY_2

[INFO] elapsed time: 0.163853 sec.
[INFO] 1/4 (25.000000%) finished.
[TASK_KEY:TASK_KEY_3] echo $TASK_KEY
TASK_KEY_3

[INFO] elapsed time: 0.201432 sec.
[INFO] 2/4 (50.000000%) finished.
[TASK_KEY:TASK_KEY_4] echo $TASK_KEY
TASK_KEY_4

[INFO] elapsed time: 0.352685 sec.
[INFO] 3/4 (75.000000%) finished.
[TASK_KEY:TASK_KEY_1] echo $TASK_KEY
TASK_KEY_1

[INFO] elapsed time: 0.351710 sec.
[INFO] 4/4 (100.000000%) finished.
{
  "details": [
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_2",
      "status": "success"
    },
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_3",
      "status": "success"
    },
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_4",
      "status": "success"
    },
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_1",
      "status": "success"
    }
  ],
  "elapsedTimeSec": 21,
  "failureNum": 0,
  "startedAt": "2021-10-05T08:40:25.441356613Z",
  "status": "success",
  "successNum": 4,
  "totalNum": 4
}
```

## 7. Export Artifacts

If you want to get the artifacts of task processing, use `exportArtifacts`.

- _examples/export-artifact.yaml

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: strategy-static-testjob
  namespace: default
spec:
  strategy:
    key:
      env: TASK_KEY
      source:
        static:
          - TASK_KEY_1
          - TASK_KEY_2
          - TASK_KEY_3
    scheduler:
      maxContainersPerPod: 10
      maxConcurrentNumPerPod: 10
  exportArtifacts:
    - name: result
      path: /tmp/artifacts
  template:
    metadata:
      generateName: strategy-static-testjob-
    spec:
      artifacts:
        - name: result
          container:
            name: test
            path: /work/result.txt
      containers:
        - name: test
          image: alpine
          workingDir: /work
          command:
            - touch
          args:
            - result.txt
```

### Output

```console
[INFO] found 3 static keys to start distributed task
[TASK_KEY:TASK_KEY_2] touch result.txt
[INFO] elapsed time: 0.191768 sec.
[INFO] 1/3 (33.333336%) finished.
[TASK_KEY:TASK_KEY_3] touch result.txt
[INFO] elapsed time: 0.191810 sec.
[INFO] 2/3 (66.666672%) finished.
[TASK_KEY:TASK_KEY_1] touch result.txt
[INFO] elapsed time: 0.191841 sec.
[INFO] 3/3 (100.000000%) finished.
[INFO] export artifact result
{
  "details": [
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_2",
      "status": "success"
    },
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_3",
      "status": "success"
    },
    {
      "elapsedTimeSec": 0,
      "name": "TASK_KEY_1",
      "status": "success"
    }
  ],
  "elapsedTimeSec": 14,
  "failureNum": 0,
  "startedAt": "2021-10-05T09:12:18.296041274Z",
  "status": "success",
  "successNum": 3,
  "totalNum": 3
}
```

#### Path Rule

Artifacts are created under the directory `<Container Name><Pod Index>-<Container Index>` .

```console
/tmp/artifacts
|-- test0-0
|   `-- result.txt
|-- test0-1
|   `-- result.txt
`-- test0-2
    `-- result.txt
```

## ServiceAccount

The ServiceAccount settings that need to be assigned to Pod that use the kubetest CLI is as follows.

```yaml
kind: ServiceAccount
apiVersion: v1
metadata:
  name: kubetest
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: kubetest
rules:
  - apiGroups:
      - batch
    resources:
      - jobs
    verbs:
      - create
      - delete
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
      - delete
  - apiGroups:
      - ""
    resources:
      - pods/log
    verbs:
      - get
      - watch
  - apiGroups:
      - ""
    resources:
      - pods/exec
    verbs:
      - create
  - apiGroups:
      - ""
    resources:
      - secrets
    verbs:
      - get
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: kubetest
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kubetest
subjects:
- kind: ServiceAccount
  name: kubetest
---
```

# License

MIT