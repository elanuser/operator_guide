# Getting Started With Kubernetes Operators (Helm Based)

## What is an Operator?

Whenever we deploy our application on Kubernetes we leverage multiple Kubernetes objects like deployment, service, role, ingress, config map, etc. As our application gets complex and our requirements become non-generic, managing our application only with the help of native Kubernetes objects becomes difficult and we often need to introduce manual intervention or some other form of automation to make up for it.

Operators solve this problem by making our application first-class Kubernetes objects that is we no longer deploy our application as a set of native Kubernetes objects but a custom object/resource of its kind, having a more domain-specific schema and then we bake the “operational intelligence” or the “domain-specific knowledge” into the controller responsible for maintaining the desired state of this object. For example, the etcd operator has made the etcd-cluster a first-class object and for deploying the cluster we create an object of Etcd Cluster kind. With operators, we are able to extend Kubernetes functionalities for custom use cases and manage our applications in a Kubernetes-specific way allowing us to leverage Kubernetes APIs and Kubectl tooling.

Operators combine CRD and custom controllers and intend to eliminate the requirement for manual intervention (human operator) while performing tasks like an upgrade, handling failure recovery, scaling in case of complex (often stateful) applications and make them more resilient and self-sufficient.

## How to Build Operators ?

For building and managing operators we mostly leverage the [Operator Framework](https://github.com/operator-framework) which is an open source tool kit allowing us to build operators in a highly automated, scalable and effective way.  Operator framework comprises of [three subcomponents](https://sdk.operatorframework.io/docs/building-operators/):

1. **Operator SDK:** Operator SDK is the most important component of the operator framework. It allows us to bootstrap our operator project in minutes. It exposes higher level APIs and abstraction and saves developers the time to dig deeper into kubernetes APIs and focus more on building the operational logic. It performs common tasks like getting the controller to watch the custom resource (cr) for changes etc as part of the project setup process.
2. **Operator Lifecycle Manager:** Operators also run on the same kubernetes clusters in which they manage applications and more often than not we create multiple operators for multiple applications. Operator lifecycle manager (OLM) provides us a declarative way to install, upgrade and manage all the operators and their dependencies in our cluster.
3. **Operator Metering:** Operator metering is currently an alpha project. It records historical cluster usage and can generate usage reports showing usage breakdown by pod or namespace over arbitrary time periods.

## Types of Operators

Currently there are three different types of operator we can build:

1. **Helm based operators:** Helm-based operators allow us to use our existing Helm charts and build operators using them. Helm based operators are quite easy to build and are preferred to deploy a stateless application using an operator pattern.
2. **Ansible based Operator:** Ansible-based operator allows us to use our existing Ansible playbooks and roles and build operators using them. They are also easy to build and generally preferred for stateless applications.
3. **Go based operators:** Go based operators are built to solve the most complex use cases and are generally preferred for stateful applications. In case of an golang based operator, we build the controller logic ourselves providing it with all our custom requirements. This type of operators is also relatively complex to build.

## Operator Maturity Model
![operator-capability-level](https://sdk.operatorframework.io/operator-capability-level.png)

## Building a Helm based operator

### 1. Let’s first install the operator sdk

```shell
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
git checkout master
make install

# operator-sdk version  
operator-sdk version: "v1.32.0", commit: "4dcbbe343b29d325fd8a14cc60366335298b40a3", kubernetes version: "1.26.0", go version: "go1.19.13", GOOS: "darwin", GOARCH: "amd64"
```

Now we will have the operator-sdk binary in the $GOPATH/bin folder.    

### 2.  Setup the project

For building a helm-based operator we can use an existing Helm chart. We will be using the `book-store Helm chart` which deploys a simple Python app and MongoDB instances. This app allows us to perform CRUD operations via. rest endpoints.

Now we will use the operator-sdk to create our Helm based bookstore-operator project.

```shell
cd helm-operator && operator-sdk init --plugins=helm --domain=example.com --helm-chart=helm/book-store
```

In the above command, The operator sdk takes only this much information and creates the custom resource definition(CRD) and also the custom resource (CR) of its type for us (remember we talked about the high-level abstraction operator sdk provides). The above command bootstraps a project with the below folder structure.

```
# tree helm-operator -L 1
helm-operator
├── Dockerfile
├── Makefile
├── PROJECT
├── config
├── helm
├── helm-charts
└── watches.yaml
```

We had discussed the operator-sdk automates setting up the operator projects and that is exactly what we can observe here.  We have the Dockerfile to build the operator image. Under `config` folder, we have a `crd` folder containing the definition of the CRD. Under folder `manager`, it has `manager.yaml` file using which we will run the operator in our cluster, along with this we have rbac files for role, rolebinding and service account to be used while deploying the operator.  We have the book-store helm chart under helm-charts. In the watches.yaml file.

```yaml
# Use the 'create api' subcommand to add watches to this file.
- group: charts.example.com
  version: v1alpha1
  kind: BookStore
  chart: helm-charts/book-store
#+kubebuilder:scaffold:watch
```

We can see that the bookstore-operator watches events related to BookStore kind objects and executes the helm chart specified.

If we take a look at the cr file under the `config/samples` (charts_v1alpha1_bookstore.yaml) folder then we can see that it looks like the values.yaml file of the book-store helm chart.

```yaml
apiVersion: charts.example.com/v1alpha1
kind: BookStore
metadata:
  name: bookstore-sample
spec:
  # Default values copied from <project_dir>/helm-charts/book-store/values.yaml
  affinity: {}
  image:
    app:
      pullPolicy: IfNotPresent
      repository: custom/pyapp
      tag: latest
    mongodb:
      pullPolicy: IfNotPresent
      repository: mongo
      tag: latest
  nodeSelector: {}
  replicaCount: 1
  resources: {}
  service:
    app:
      port: 80
      targetPort: 3000
      type: LoadBalancer
    mongodb:
      port: 27017
      targetPort: 27017
      type: ClusterIP
  tolerations: []
```

In the case of Helm charts, we use the values.yaml file to pass the parameter to our Helm releases, Helm based operator converts all these configurable parameters into the spec of our custom resource. This allows us to express the values.yaml with a custom resource (CR) which, as a native Kubernetes object, enables the benefits of RBAC applied to it and an audit trail. Now when we want to update the deployed artifacts we can simply modify the CR and apply it, and the operator will ensure that the changes we made are reflected in our app.

For each object of  `BookStore` kind  the bookstore operator will perform the following actions:

1. Create the bookstore app deployment if it doesn’t exists.
2. Create the bookstore app service if it doesn’t exists.
3. Create the mongodb deployment if it doesn’t exists.
4. Create the mongodb service if it doesn’t exists.
5. Ensure deployments and services match their desired configurations like the replica count, image tag, service port etc.  

### 3. Build the Bookstore-operator Image

The Dockerfile for building the operator image is already in our build folder we need to run the below command from the root folder of our operator project to build the image.

```shell
docker login -u ${DOCKER_HUB_USRE} -p ${DOCKER_HUB_PASSWD} docker.io
make docker-build docker-push IMG="{DOCKER_USER}/bookstore-operator-helm:0.0.1"
```

### 4. Run the Bookstore-operator

As we have our operator image ready we can now go ahead and run it.  By updating the image with the below command, we are deploying the operator.

```yaml
# make deploy IMG="{DOCKER_USER}/bookstore-operator-helm:0.0.1"
cd config/manager && /usr/local/bin/kustomize edit set image controller=/bookstore-operator-helm:0.0.1
/usr/local/bin/kustomize build config/default | kubectl apply -f -
namespace/helm-operator-system unchanged
customresourcedefinition.apiextensions.k8s.io/bookstores.charts.example.com unchanged
serviceaccount/helm-operator-controller-manager unchanged
role.rbac.authorization.k8s.io/helm-operator-leader-election-role unchanged
clusterrole.rbac.authorization.k8s.io/helm-operator-manager-role unchanged
clusterrole.rbac.authorization.k8s.io/helm-operator-metrics-reader unchanged
clusterrole.rbac.authorization.k8s.io/helm-operator-proxy-role unchanged
rolebinding.rbac.authorization.k8s.io/helm-operator-leader-election-rolebinding unchanged
clusterrolebinding.rbac.authorization.k8s.io/helm-operator-manager-rolebinding unchanged
clusterrolebinding.rbac.authorization.k8s.io/helm-operator-proxy-rolebinding unchanged
service/helm-operator-controller-manager-metrics-service unchanged
deployment.apps/helm-operator-controller-manager unchanged
```

***Note:\*** *The role created might have more permissions then actually required for the operator so it is always a good idea to review it and trim down the permissions in production setups.*

```
failed to check CRD: failed to list CRDs: customresourcedefinitions.apiextensions.k8s.io is forbidden: User \"system:serviceaccount:helm-operator-system:helm-operator-controller-manager\" cannot list resource \"customresourcedefinitions\" in API group \"apiextensions.k8s.io\" at the cluster scope"
```

If the above error message occurs in the operator,  please consider adding the following powerful allow-all privilege to the ClusterRole `manager-role`.

```
- apiGroups:
  - "*"
  resources:
  - "*"
  verbs:
  - "*"
```

Verify that the operator pod is in running state.

```sh
# k -n helm-operator-system get pods -l control-plane=controller-manager
NAME                                                READY   STATUS    RESTARTS   AGE
helm-operator-controller-manager-84958dcd9c-n9ltd   2/2     Running   0          2m51s
```

### 5. Deploy the Bookstore App

Now we have the bookstore operator running in the cluster we just need to create the custom resource for deploying our bookstore app.

```shell
k apply -f config/samples/charts_v1alpha1_bookstore.yaml
```

Now we can see that our operator has deployed out book-store app.

## Conclusion

Since its early days, Kubernetes was believed to be a great tool for managing stateless applications but managing stateful applications on Kubernetes was always considered difficult. Operators are a big leap towards managing stateful applications and other complex distributed, multi cloud workloads with the same ease that we manage stateless applications.



# Getting Started With Kubernetes Operators (Ansible Based)


Before we start building the operator let’s spend some time in understanding the operator maturity model. Operator maturity model gives an idea of the kind of application management capabilities different types of operators can have. As we can see in the diagram above the model describes five generic phases of maturity/capability for operators. The minimum expectation/requirement from an operator is that they should be able to deploy/install and upgrade application and that is provided by all the operators. 

Helm-based operators are the simplest of all of them as Helm is Chart manager and we can do only install and upgrades using it. Ansible-based operators can be more mature as Ansible has modules to perform a wide variety of operational tasks, we can use these modules in the Ansible roles/playbooks we use in our operator and make them handle more complex applications or use cases. In the case of Golang-based operators, we write the operational logic ourselves so we have the liberty to customize it as per our requirements.

## Building an Ansible Based Operator

### 1. Let’s first install the operator sdk

```
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
git checkout master
make install

# operator-sdk version  
operator-sdk version: "v1.32.0", commit: "4dcbbe343b29d325fd8a14cc60366335298b40a3", kubernetes version: "1.26.0", go version: "go1.19.13", GOOS: "darwin", GOARCH: "amd64"
```

### 2. Setup the project

```
operator-sdk init --domain example.com --plugins ansible
operator-sdk create api --group charts --version v1alpha1 --kind BookStore --generate-role
```

In the above command, we have set the operator type as Ansible as we want an Ansible-based operator. It creates a folder structure as shown below

```
# tree ansible-operator -L 1
ansible-operator
├── Dockerfile
├── Makefile
├── PROJECT
├── config
├── molecule
├── playbooks
├── requirements.yml
├── roles
└── watches.yaml
```

Inside the `roles` folder, it creates an Ansible role named `bookstore`. This role is bootstrapped with all the directories and files which are part of the standard ansible roles.

Now let’s take a look at the watches.yaml file:

```yaml
---
# Use the 'create api' subcommand to add watches to this file.
- version: v1alpha1
  group: charts.example.com
  kind: BookStore
  role: bookstore
#+kubebuilder:scaffold:watch
```

Here we can see that it looks just like the operator is going to watch the events related to the objects of kind `BookStore` and execute the ansible role `bookstore`. Drawing parallels from our helm-based operator we can see that the behavior in both cases is similar the only difference being that in the case of Helm based operator, the operator used to execute the helm chart specified in response to the events related to the object it was watching and here we are executing an ansible role.

In the case of Ansible-based operators, we can get the operator to execute an Ansible playbook as well rather than an Ansible role.

### 3. Building the bookstore Ansible role

Now we need to modify the bookstore Ansible roles created for us by the operator-framework.

Firstly, we will update the custom resource (CR) file (`charts_v1alpha1_bookstore.yaml`) available at `config/samples` location. In this CR we can configure all the values which we want to pass to the bookstore Ansible role. 

By default the CR contains only the size field, we will update it to include other fields that we need in our role. To keep things simple, we will just include some basic variables like image name, tag, etc. in our spec.

```yaml
apiVersion: charts.example.com/v1alpha1
kind: BookStore
metadata:
  labels:
    app.kubernetes.io/name: bookstore
    app.kubernetes.io/instance: bookstore-sample
    app.kubernetes.io/part-of: ansible-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: ansible-operator
  name: bookstore-sample
  namespace: default
spec:
  metadata:
    namespace: default
  image:
    app:
      repository: custom/pyapp
      tag: latest
      pullPolicy: Always
    mongodb:
      repository: mongo
      tag: latest
      pullPolicy: Always
  service:
    app:
      type: ClusterIP
    mongodb:
      type: ClusterIP
```

The Ansible operator passes the key-value pairs listed in the spec of the CR as variables to Ansible. The operator changes the name of the variables to snake_case before running Ansible so when we use the variables in our role we will refer to the values in the snake case.

Next, we need to create the tasks the bookstore roles will execute. Now we will update the tasks to define our deployment. By default, an Ansible role executes the tasks defined at `roles/bookstore/tasks/main.yml`. For defining our deployment we will leverage the [k8s](https://docs.ansible.com/ansible/latest/modules/k8s_module.html) module of Ansible. We will create a Kubernetes deployment and service for our app as well as MongoDB.

```yaml
---
# tasks file for bookstore

- name: Create the mongodb deployment
  k8s:
    definition:
      kind: Deployment
      apiVersion: apps/v1
      metadata:
        name: mongodb-deployment
        namespace: '{{ metadata.namespace }}'
      spec:
        replicas: 1
        selector:
          matchLabels:
            app: book-store-mongodb
        template:
          metadata:
            labels:
              app: book-store-mongodb
          spec:
            containers:
            - name: mongodb
              image: "{{image.mongodb.repository}}:{{image.mongodb.tag}}"
              imagePullPolicy: "{{ image.mongodb.pull_policy }}"
              ports:
              - containerPort: 27017

- name: Create the mongodb service
  k8s:
    definition:
      apiVersion: v1
      kind: Service
      metadata:
        name: mongodb-service
        namespace: '{{ metadata.namespace }}'
        labels:
          app: book-store-mongodb
      spec:
        type: "{{service.mongodb.type}}"
        ports:
        - name: elb-port
          port: 27017
          protocol: TCP
          targetPort: 27017
        selector:
          app: book-store-mongodb
          
- name: Create the bookstore deployment
  k8s:
    definition: 
      kind: Deployment
      apiVersion: apps/v1
      metadata:
        name: book-store
        namespace: '{{ metadata.namespace }}'
      spec:
        replicas: 1
        selector:
          matchLabels:
            app: book-store
        template:
          metadata:
            labels:
              app: book-store
          spec:
            containers:
            - name: book-store
              image: "{{image.app.repository}}:{{image.app.tag}}"
              imagePullPolicy: "{{image.app.pull_policy}}"         
              ports:
              - containerPort: 3000

- name: Create the bookstore service
  k8s:
    definition:
      apiVersion: v1
      kind: Service
      metadata:
        name: book-store
        namespace: '{{ metadata.namespace }}'
        labels:
          app: book-store
      spec:
        type: "{{service.app.type}}"
        ports:
        - name: elb-port
          port: 80
          protocol: TCP
          targetPort: 3000
        selector:
          app: book-store
```

In the above file, we can see that we have used the pullPolicy field defined in our cr spec as `pull_policy` in our tasks. Here we have used inline definition to create our k8s objects as our app is quite simple. For large applications creating objects using separate definition files would be a better approach.

### 4. Build the bookstore-operator image

The Dockerfile for building the operator image is already in our build folder we need to run the below command from the root folder of our operator project to build the image.

```sh
docker login -u ${DOCKER_USER} -p ${DOCKER_HUB_PASSWD} docker.io
make docker-build docker-push IMG="{DOCKER_USER}/bookstore-operator-ansible:0.0.1"
```

You can use your own docker repository instead of `{DOCKER_USER}`

### 5. Run the bookstore-operator

As we have our operator image ready we can now go ahead and run it. The deployment file (operator.yaml under deploy folder) for the operator was created as a part of our project setup we just need to set the image for this deployment to the one we built in the previous step.

After updating the image in the operator.yaml we are ready to deploy the operator.

```sh
make deploy IMG="{DOCKER_USER}/bookstore-operator-ansible:0.0.1"
```

> ***Note:\*** *The role created might have more permissions than required for the operator so it is always a good idea to review it and trim down the permissions in production setups.*

Verify that the operator pod is in running state.

```sh
# k -n ansible-operator-system get pods
NAME                                                   READY   STATUS    RESTARTS   AGE
ansible-operator-controller-manager-668f785c87-4ddls   2/2     Running   0          66s
```

Here two containers have been started as part of the operator deployment. One is the operator and the other one is ansible. The Ansible pod exists only to make the logs available to stdout in Ansible format.

### 6. Deploy the bookstore app

Now we have the bookstore-operator running in our cluster we just need to create the custom resource for deploying our bookstore app.

```sh
k apply -f config/samples/charts_v1alpha1_bookstore.yaml 
```

Now we can see that our operator has deployed our book-store app:

## Conclusion

Ansible based operators are a great way to combine the power of Ansible and Kubernetes as it allows us to deploy our applications using Ansible role and playbooks and we can pass parameters to them (control them) using custom K8s resources. If Ansible is being heavily used across your organization and you are migrating to Kubernetes then Ansible based operators are an ideal choice for managing deployments. In the next blog, we will learn about Golang based operators.



# Getting Started With Kubernetes Operators (Golang Based) 

## Introduction

In the case of Helm-based operators, we were executing a Helm chart when changes were made to the custom object type of our application, similarly in the case of an Ansible-based operator we executed an Ansible role. In the case of Golang-based operators we write the code for the action we need to perform (reconcile logic) whenever the state of our custom object changes, this makes the Golang-based operators quite powerful and flexible, at the same time making them the most complex to build out of the 3 types.

### What Will We Build?

The database server we deployed as part of our book store app in previous blogs didn’t have any persistent volume attached to it and we would lose data in case the pod restarts, to avoid this we will attach a persistent volume attached to the host (K8s worker nodes ) and run our database as a statefulset rather than deployment. We will also add a feature to expand the persistent volume associated with the Mongodb pod.

## Building the Operator

### 1. Create a project directory and initialize the project:  

```
mkdir bookstore-operator
cd bookstore-operator
operator-sdk init --domain example.com --repo github.com/example/bookstore-operator
```

After the command is executed, the below result is shown.

```
Writing kustomize manifests for you to edit...
Writing scaffold for you to edit...
Get controller runtime:
$ go get sigs.k8s.io/controller-runtime@v0.14.1
Update dependencies:
$ go mod tidy
Next: define a resource with:
$ operator-sdk create api
```

### 2. Add the custom resources and controller

```
operator-sdk create api --group charts --version v1 --kind BookStore --resource --controller
```

The above command creates the CRD and CR for the BookStore type. It also creates the Golang structs (api/v1/bookstore_types.go)  for **BookStore** types.  It also registers the custom type through `SchemeBuilder` and generates deep-copy methods as well. Here we can see that all the generic tasks are being done by the operator framework itself allowing us to focus on building an object and the controller. We will update the spec of the BookStore object later. 

We will update the spec of the BookStore type to include two custom types BookApp and BookDB.

```go
type BookApp struct {
	Repository      string             `json:"repository,omitempty"`
	Tag             string             `json:"tag,omitempty"`
	ImagePullPolicy corev1.PullPolicy  `json:"imagePullPolicy,omitempty"`
	Replicas        int32              `json:"replicas,omitempty"`
	Port            int32              `json:"port,omitempty"`
	TargetPort      int                `json:"targetPort,omitempty"`
	ServiceType     corev1.ServiceType `json:"serviceType,omitempty"`
}

type BookDB struct {
	Repository      string            `json:"repository,omitempty"`
	Tag             string            `json:"tag,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Replicas        int32             `json:"replicas,omitempty"`
	Port            int32             `json:"port,omitempty"`
	DBSize          resource.Quantity `json:"dbSize,omitempty"`
}

type BookStoreSpec struct {
	BookApp BookApp     `json:"bookApp,omitempty"`
	BookDB  BookDB      `json:"bookDB,omitempty"`
}
```

Let’s also update the BookStore CR (`config/samples/charts_v1_bookstore.yaml`)

```yaml
apiVersion: charts.example.com/v1
kind: BookStore
metadata:name: bookstore-sample
spec:
  bookApp: 
    repository: "custom/pyapp"
    tag: latest
    imagePullPolicy: "Always"
    replicas: 1
    port: 80
    targetPort: 3000
    serviceType: "ClusterIP"
  bookDB:
    repository: "mongo"
    tag: latest
    imagePullPolicy: "Always"
    replicas: 1
    port: 27017
    dbSize: 2Gi
```

If we take a look at the add function in the `controllers/bookstore_controller.go` file we can see that a new controller is created here and added to the manager so that the manager can start the controller when it (manager) comes up, all the technical details have been wrapped into the `controller-runtime` library.

```go
// SetupWithManager sets up the controller with the Manager.
func (r *BookStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chartsv1.BookStore{}).
		Complete(r)
}

// For defines the type of Object being *reconciled*, and configures the ControllerManagedBy to respond to create / delete /
// update events by *reconciling the object*.
// This is the equivalent of calling
// Watches(&source.Kind{Type: apiType}, &handler.EnqueueRequestForObject{}).
func (blder *Builder) For(object client.Object, opts ...ForOption) *Builder 


// Complete builds the Application Controller.
func (blder *Builder) Complete(r reconcile.Reconciler) error
```

This ensures that for any events related to any object of BookStore type, a reconcile request (a namespace/name key) is sent to the `Reconcile` method associated with the reconciler object `BookStoreReconciler` .

### 3. Build the reconciled logic

The reconcile logic is implemented inside the method `Reconcile` of the reconciler object which implements the reconcile loop.

The following steps belong to the reconciliation logic:

1. Create the bookstore app deployment if it doesn’t exist.
2. Create the bookstore app service if it doesn’t exist.
3. Create the MongoDB Statefulset if it doesn’t exist.
4. Create the MongoDB service if it doesn’t exist.
5. Ensure deployments and services match their desired configurations like the replica count, image tag, service port, size of the PV associated with the Mongodb statefulset etc.

 Three possible events can happen with the BookStore object

1. **The object got created:** Whenever an object of kind BookStore is created we create all the k8s resources we mentioned above
2. **The object has been updated:** When the object gets updated then we update all the k8s resources associated with it..
3. **The object has been deleted:** When creating the K8s objects we set the `BookStore` type as its owner which will ensure that all the K8s objects are associated with it, the deletion operation will automatically delete them when the `BookStore` object has been deleted.

On receiving the reconcile request, the first step is to lookup for the object.

If the object is not found, we assume that it got deleted and doesn’t require reconciliations.

If any error occurs while doing the `Reconcile` then we return the error and whenever we return a non-nil error value the controller re-queues the request.

```go
// +kubebuilder:rbac:groups=charts.example.com,resources=bookstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=charts.example.com,resources=bookstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=charts.example.com,resources=bookstores/finalizers,verbs=update
func (r *BookStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling BookStore")

	// fetch the latest status of bookStore
	bookStore := chartsv1.BookStore{}
	if err := r.Client.Get(ctx, req.NamespacedName, &bookStore); err != nil {
		// Request object not found, could have been deleted after reconcile request.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		reqLogger.Error(err, "Failed to retrieve the bookstore object")
		return reconcile.Result{}, err
	}

	err := r.ReconcileBookStore(ctx, &bookStore)
	if err != nil {
		reqLogger.Error(err, "Failed to reconcile the bookstore resources")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
```

In the reconcile logic we call the BookStore method which creates or updates all the k8s objects associated with the BookStore objects based on whether the object has been created or updated.

```go
func (r *BookStoreReconciler) ReconcileBookStore(ctx context.Context, bookstore *chartsv1.BookStore) error {
	// Create/Update the bookStore deployment
	// Create/Update the bookStore service
	// Create/Update the mongodb statefulSet
	// Create/Update the mongodb service
}
```

The implementation of the above method is a bit hacky but gives an idea of the flow. In the above function, we are setting the BookStore type as an owner for all the resources through `controllerutil.SetControllerReference`. If we look at the owner reference for these objects we would see something like this.

```yaml
  ownerReferences:
  - apiVersion: charts.example.com/v1
    blockOwnerDeletion: true
    controller: true
    kind: BookStore
    name: bookstore
    uid: 38f512ef-018b-4842-a868-44cb702d1683
```

### 4.  Deploy the operator and verify it works

Before we deploy the operator, the docker image should be built and pushed to the remote docker registry.

```sh
make docker-build docker-push IMG="${DOCKER_USER}/bookstore-operator-go:0.0.1"
```

After the docker image is ready; the next step is to deploy the operator 

```sh
make deploy IMG="${DOCKER_USER}/bookstore-operator-go:0.0.1"
k -n go-operator-system delete pods -l control-plane=controller-manager
k -n go-operator-system logs -l control-plane=controller-manager -f
```

Below, we can observe the outcome from the cluster.

```sh
# k -n go-operator-system get pods -l control-plane=controller-manager
NAME                                              READY   STATUS    RESTARTS   AGE
go-operator-controller-manager-64b5d5f959-6rd89   2/2     Running   0          52s

# k -n go-operator-system logs -l control-plane=controller-manager -f
I1227 11:20:35.907770       1 leaderelection.go:258] successfully acquired lease go-operator-system/333fff00.example.com
2023-12-27T11:20:35Z	DEBUG	events	go-operator-controller-manager-64b5d5f959-vk2p8_59a96f5a-1fcf-48e9-ac46-41e3fab11b9e became leader	{"type": "Normal", "object": {"kind":"Lease","namespace":"go-operator-system","name":"333fff00.example.com","uid":"15d48030-fcdd-4d7e-a1ae-f4b4e4d6b0e7","apiVersion":"coordination.k8s.io/v1","resourceVersion":"388475"}, "reason": "LeaderElection"}
2023-12-27T11:20:35Z	INFO	Starting EventSource	{"controller": "bookstore", "controllerGroup": "charts.example.com", "controllerKind": "BookStore", "source": "kind source: *v1.BookStore"}
2023-12-27T11:20:35Z	INFO	Starting Controller	{"controller": "bookstore", "controllerGroup": "charts.example.com", "controllerKind": "BookStore"}
2023-12-27T11:20:36Z	INFO	Starting workers	{"controller": "bookstore", "controllerGroup": "charts.example.com", "controllerKind": "BookStore", "worker count": 1}
```

Finally,  Deploy the operator and artifacts to the cluster.

```sh
k apply -f config/samples/charts_v1_bookstore.yaml
```

The approach to deploy and verify the working of the bookstore application is similar to what we did before, the only difference being that now we have deployed the MongoDB as a stateful set and even if we restart the pod we will see that the information that we stored will still be available.

```sh
# k get pods
NAME                                              READY   STATUS             RESTARTS        AGE
bookstore-85b84c8fc5-d6d8l                        1/1     Running            0               25m
go-operator-controller-manager-64b5d5f959-vk2p8   2/2     Running            0               43s
mongodb-0                                         1/1     Running            0               23s
```



### 5. Verify volume expansion

For updating the volume associated with the Mongodb instance we first need to update the size of the volume we specified while creating the bookstore object. 

In the example above, I increased the volume from 2GB to 20GB in the bookstore object.

Once the bookstore object is updated if we describe the mongodb PVC we will see that it still has 2GB PV but the conditions we will see something like this.

```sh
# k describe pvc mongodb-pvc-mongodb-0
Name:          mongodb-pvc-mongodb-0
Used By:       mongodb-0
Conditions:
  Type                      Status  LastProbeTime                     LastTransitionTime                Reason  Message
  ----                      ------  -----------------                 ------------------                ------  -------
Events:
  Type     Reason                    Age                From                                                                                        Message
  ----     ------                    ----               ----                                                                                        -------
  Normal   WaitForFirstConsumer      17m                persistentvolume-controller                                                                 waiting for first consumer to be created before binding
  Normal   Provisioning              17m                ebs.csi.aws.com_csi-driver-controller-9d88b6dfc-6827n_5a83b3fc-a9e0-4601-84db-934ea6224f24  External provisioner is provisioning volume for claim "go-operator-system/mongodb-pvc-mongodb-0"
  Normal   ExternalProvisioning      17m (x3 over 17m)  persistentvolume-controller                                                                 waiting for a volume to be created, either by external provisioner "ebs.csi.aws.com" or manually created by system administrator
  Normal   ProvisioningSucceeded     17m                ebs.csi.aws.com_csi-driver-controller-9d88b6dfc-6827n_5a83b3fc-a9e0-4601-84db-934ea6224f24  Successfully provisioned volume pv-2e26-475c-80fa-e4c2d9704813
  Warning  ExternalExpanding         7m21s              volume_expand                                                                               Ignoring the PVC: didn't find a plugin capable of expanding the volume; waiting for an external controller to process this PVC.
  Normal   Resizing                  7m21s              external-resizer ebs.csi.aws.com                                                            External resizer is resizing volume pv-2e26-475c-80fa-e4c2d9704813
  Normal   FileSystemResizeRequired  7m15s              external-resizer ebs.csi.aws.com                                                            Require file system resize of volume on node
 
```

It is clear from the message that we need to restart the pod for resizing of volume to reflect. Once we delete the pod it will get restarted and the PVC will get updated to reflect the expanded volume size.

```sh
2023-12-27T11:20:35Z    INFO    Starting Controller     {"controller": "bookstore", "controllerGroup": "charts.example.com", "controllerKind": "BookStore"}
2023-12-27T11:20:36Z    INFO    Starting workers        {"controller": "bookstore", "controllerGroup": "charts.example.com", "controllerKind": "BookStore", "worker count": 1}
2023-12-27T11:20:36Z    INFO    controller_bookstore    Reconciling BookStore   {"Request.Namespace": "go-operator-system", "Request.Name": "bookstore"}
2023-12-27T11:20:36Z    INFO    controller_bookstore    bookstore deployment updated    {"Namespace": "go-operator-system"}
2023-12-27T11:20:36Z    INFO    controller_bookstore    bookstore service updated       {"Namespace": "go-operator-system"}
2023-12-27T11:20:36Z    INFO    controller_bookstore    failed to retrieve mongodb StatefulSet  {"Namespace": "go-operator-system"}
2023-12-27T11:20:36Z    INFO    controller_bookstore    start to create the mongodb StatefulSet {"Namespace": "go-operator-system"}
2023-12-27T11:31:01Z    INFO    controller_bookstore    Reconciling BookStore   {"Request.Namespace": "go-operator-system", "Request.Name": "bookstore"}
2023-12-27T11:31:01Z    INFO    controller_bookstore    bookstore deployment updated    {"Namespace": "go-operator-system"}
2023-12-27T11:31:01Z    INFO    controller_bookstore    bookstore service updated       {"Namespace": "go-operator-system"}
2023-12-27T11:31:01Z    INFO    controller_bookstore    Need to expand the mongodb volume       {"Namespace": "go-operator-system"}
2023-12-27T11:31:01Z    INFO    controller_bookstore    mongodb volume updated successfully     {"Namespace": "go-operator-system"}
2023-12-27T11:31:01Z    INFO    controller_bookstore    mongodb StatefulSet updated     {"Namespace": "go-operator-system"}
2023-12-27T11:31:01Z    INFO    controller_bookstore    mongodb-service updated {"Namespace": "go-operator-system"}
```


## Conclusion

Golang-based operators are built mostly for stateful applications like databases. The operator can automate complex operational tasks allowing us to run applications with ease. At the same time, building and maintaining it can be quite complex and we should build one only when we are fully convinced that our requirements can’t be met with any other type of operator. Operators are an interesting and emerging area in Kubernetes and I hope this blog series on getting started with it helps the readers in learning the basics of it.


# Continuous Integration & Delivery (CI/CD)

## Introduction

Kubernetes is getting adopted rapidly across the software industry and is becoming the most preferred option for deploying and managing containerized applications. Once we have fully functional operators we need to have an automated process to deploy the operator to the Kubernetes cluster. In this blog post, we will create a fully automated `commit to deploy` a pipeline for Kubernetes. We will use CircleCI to achieve this goal.

## What is CircleCI?

CircleCI is a fully managed SAAS offering which allows us to build, test or deploy our code at every check-in. To get started with Circle we need to log into their web console with our GitHub credentials, add a project for the repository we want to build and then add the CircleCI config file to our repository. 

The CircleCI config file is a yaml file which lists the steps we want to execute every time code is pushed to that repository.

Some excellent features of CircleCI are:

1. Little or no operational overhead as the infrastructure is managed completely by CircleCI.
2. User authentication is done via GitHub so user management is quite simple.
3. It automatically notifies the build status on the GitHub email IDs of the users who are following the project on CircleCI.
4. The UI is quite simple and gives a holistic view of builds.
5. Can be easily integrated with Slack, Jira, etc.

## Building the Pipeline

![Building the pipeline](https://assets-global.website-files.com/5d2dd7e1b4a76d8b803ac1aa/5d8b252d8c12c789963b9b00_CICD%2Bfor%2Bkubernetes%2B-%2B1.png)

### Configure the pipeline:

```yaml
version: 2.1
orbs:
  docker: circleci/docker@2.4.0
  kubernetes: circleci/kubernetes@1.3.1

jobs:
  build:
    docker:
      - image: cimg/go:1.21.5
        auth:
          username: $DOCKERHUB_USER
          password: $DOCKERHUB_PASS
    steps:
      - checkout
      - setup_remote_docker:
          docker_layer_caching: true
      - run:
          name: Build and Push Docker Image
          command: |
            make docker-build IMG=$BOOKSTORE_OPERATOR_IMG
            make docker-push IMG=$BOOKSTORE_OPERATOR_IMG
```

1. Retrieve username and password and use them to login the DockerHub.
2. Enable the docker layer caching to accelerate the process of further docker build.
3. Build the operator docker image and push it to the DockerHub.



```yaml
  deploy:
    docker:
    - image: cimg/go:1.21.5  # needs the go environment to install tools(controller-gen.. etc)
    steps:
      - checkout
      - kubernetes/install-kubectl
      - run:
          name: Install Helm 3
          command: |
            curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash      
      - kubernetes/install-kubeconfig:
          kubeconfig: KUBECONFIG_DATA
      - run:
          name: Deploy Kubernetes Artifacts to Cluster
          command: |
            make deploy IMG=$BOOKSTORE_OPERATOR_IMG
```

1. Check out the code and Install the necessary tools.
2. Retrieve the kubeconfig file and persist it locally.
3. Deploy the operator and related artifacts to the Kubernetes cluster.



```yaml
workflows:
  workflow:
    jobs:
      - build:
          context:
            - build-env-vars
            - docker-hub-creds
      - deploy:
          requires:
            - build
```

At the end of the file, we see the workflows, workflows control the order in which the jobs specified in the file are executed and establish dependencies and conditions for the job. 

We have used a few environment variables in our pipeline configuration some of them were created by us and some were made available by CircleCI. We created DOCKERHUB_USER, DOCKERHUB_PASS, BOOKSTORE_OPERATOR_IMG and KUBECONFIG_DATA variables. These variables are set via the CircleCI web console by going to the project settings. 

### Verify the working of the pipeline:

Leverage the below command to submit a dummy commit to trigger the workflow.

```shell
git commit --allow-empty -m 'trigger'
```



Also, verify that the newest image is updated on Docker Hub.

## Conclusion

In this blog post, we explored how we can set up a CI/CD pipeline for Kubernetes and got basic exposure to CircleCI. We can extend the pipeline to consider the cases where we have multiple environments like dev, staging & production and make the pipeline deploy the application to any of them depending upon some conditions. 