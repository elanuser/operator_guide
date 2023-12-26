# Getting Started With Kubernetes Operators (Helm Based)

## What is an Operator?

Whenever we deploy our application on Kubernetes we leverage multiple Kubernetes objects like deployment, service, role, ingress, config map, etc. As our application gets complex and our requirements become non-generic, managing our application only with the help of native Kubernetes objects becomes difficult and we often need to introduce manual intervention or some other form of automation to make up for it.

Operators solve this problem by making our application first class Kubernetes objects that is we no longer deploy our application as a set of native Kubernetes objects but a custom object/resource of its kind, having a more domain-specific schema and then we bake the “operational intelligence” or the “domain-specific knowledge” into the controller responsible for maintaining the desired state of this object. For example, etcd operator has made etcd-cluster a first class object and for deploying the cluster we create an object of Etcd Cluster kind. With operators, we are able to extend Kubernetes functionalities for custom use cases and manage our applications in a Kubernetes specific way allowing us to leverage Kubernetes APIs and Kubectl tooling.

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
      repository: akash125/pyapp
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
      repository: akash125/pyapp
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
    repository: "akash125/pyapp"
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
  - apiVersion: blog.velotio.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: BookStore
    name: example-bookstore
    uid: 0ef42889-deb4-11e9-ba56-42010a800256
```

### 4.  Deploy the operator and verify it works

Before we deploy the operator, the docker image 

```
make docker-build docker-push IMG="${DOCKER_USER}/bookstore-operator-go:0.0.1"
```

Deploy the operator and artifacts

```
make deploy IMG="${DOCKER_USER}/bookstore-operator-go:0.0.1"
k apply -f config/samples/charts_v1_bookstore.yaml
```

The approach to deploy and verify the working of the bookstore application is similar to what we did before, the only difference being that now we have deployed the MongoDB as a stateful set and even if we restart the pod we will see that the information that we stored will still be available.

![Kubernetes Golang 1.png](https://assets-global.website-files.com/5d2dd7e1b4a76d8b803ac1aa/5dfc8c939c9c5c22ce2441ae_Kubernetes%2BGolang%2B1.png)

### 5. Verify volume expansion

For updating the volume associated with the Mongodb instance we first need to update the size of the volume we specified while creating the bookstore object. 

In the example above I had set it to 2GB let’s increase it to 3GB and update the bookstore object.

Once the bookstore object is updated if we describe the mongodb PVC we will see that it still has 2GB PV but the conditions we will see something like this.

```
Conditions:
  Type                      Status  LastProbeTime                     LastTransitionTime                Reason  Message
  ----                      ------  -----------------                 ------------------                ------  -------
  FileSystemResizePending   True    Mon, 01 Jan 0001 00:00:00 +0000   Mon, 30 Sep 2019 15:07:01 +0530           Waiting for user to (re-)start a pod to finish file system resize of volume on node.
@velotiotech
 
```

It is clear from the message that we need to restart the pod for resizing of volume to reflect. Once we delete the pod it will get restarted and the PVC will get updated to reflect the expanded volume size.

![Kubernetes Golang 2.png](https://assets-global.website-files.com/5d2dd7e1b4a76d8b803ac1aa/5dfc8c937a4372973e1e56ef_Kubernetes%2BGolang%2B2.png)

## Conclusion

Golang based operators are built mostly for [stateful applications](https://www.velotio.com/engineering-blog/exploring-upgrade-strategies-for-stateful-sets-in-kubernetes) like databases. The operator can automate complex operational tasks allow us to run applications with ease. At the same time, building and maintaining it can be quite complex and we should build one only when we are fully convinced that our requirements can’t be met with any other type of operator. Operators are an interesting and emerging area in Kubernetes and I hope this blog series on getting started with it help the readers in learning the basics of it.



# Continuous Integration & Delivery (CI/CD)

## Introduction

Kubernetes is getting adopted rapidly across the software industry and is becoming the most preferred option for deploying and managing containerized applications. Once we have a fully functional [Kubernetes cluster](https://www.velotio.com/engineering-blog/the-ultimate-guide-to-disaster-recovery-for-your-kubernetes-clusters) we need to have an automated process to deploy our applications on it. In this blog post, we will create a fully automated “commit to deploy” pipeline for Kubernetes. We will use CircleCI & helm for it.

## What is CircleCI?

CircleCI is a fully managed SAAS offering which allows us to build, test or deploy our code at every check-in. To get started with Circle we need to log into their web console with our GitHub credentials, add a project for the repository we want to build and then add the CircleCI config file to our repository. 

The CircleCI config file is a yaml file which lists the steps we want to execute every time code is pushed to that repository.

Some excellent features of CircleCI are:

1. Little or no operational overhead as the infrastructure is managed completely by CircleCI.
2. User authentication is done via GitHub so user management is quite simple.
3. It automatically notifies the build status on the GitHub email IDs of the users who are following the project on CircleCI.
4. The UI is quite simple and gives a holistic view of builds.
5. Can be integrated with Slack, Jira, etc.

## What is Helm?

Helm is a chart manager where a chart refers to the package of Kubernetes resources. Helm allows us to bundle related Kubernetes objects into charts and treat them as a single unit of deployment referred to as release.  For example, you have an application app1 which you want to run on Kubernetes. For this app1 you create multiple Kubernetes resources like deployment, service, ingress, horizontal pod scaler, etc. Now while deploying the application you need to create all the Kubernetes resources separately by applying their manifest files. What Helm does is it allows us to group all those files into one chart ([Helm chart](https://www.velotio.com/engineering-blog/helm-3)) and then we just need to deploy the chart. This also makes deleting and upgrading the resources quite simple.

Some other benefits of Helm is:

1. It makes the deployment highly configurable. Thus just by changing the parameters, we can use the same chart for deploying on multiple environments like stag/prod or multiple cloud providers.
2. We can roll back to a previous release with a single helm command.
3. It makes managing and sharing Kubernetes-specific applications much simpler.

## Building the Pipeline

![Building the pipeline](https://assets-global.website-files.com/5d2dd7e1b4a76d8b803ac1aa/5d8b252d8c12c789963b9b00_CICD%2Bfor%2Bkubernetes%2B-%2B1.png)

### Configure the pipeline:

```yaml
build&pushImage:
    working_directory: /go/src/hello-app (1)
    docker:
      - image: circleci/golang:1.10 (2)
    steps:
      - checkout (3)
      - run: (4)
          name: build the binary
          command: go build -o hello-app
      - setup_remote_docker: (5)
          docker_layer_caching: true
      - run: (6)
          name: Set the tag for the image, we will concatenate the app verson and circle build number with a `-` char in between
          command:  echo 'export TAG=$(cat VERSION)-$CIRCLE_BUILD_NUM' >> $BASH_ENV
      - run: (7)
          name: Build the docker image
          command: docker build . -t ${CIRCLE_PROJECT_REPONAME}:$TAG
      - run: (8)
          name: Install AWS cli
          command: export TZ=Europe/Minsk && sudo ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > sudo  /etc/timezone && sudo apt-get update && sudo apt-get install -y awscli
      - run: (9)
          name: Login to ECR
          command: $(aws ecr get-login --region $AWS_REGION | sed -e 's/-e none//g')
      - run: (10)
          name: Tag the image with ECR repo name 
          command: docker tag ${CIRCLE_PROJECT_REPONAME}:$TAG ${HELLOAPP_ECR_REPO}:$TAG    
      - run: (11)
          name: Push the image the ECR repo
          command: docker push ${HELLOAPP_ECR_REPO}:$TAG
```



1. We set the working directory for our job, we are setting it on the gopath so that we don’t need to do anything additional.
2. We set the docker image inside which we want the job to run, as our app is built using golang we are using the image which already has golang installed in it.
3. This step checks out our repository in the working directory
4. In this step, we build the binary
5. Here we setup docker with the help of  **setup_remote_docker** key provided by CircleCI.
6. In this step we create the tag we will be using while building the image, we use the app version available in the VERSION file and append the $CIRCLE_BUILD_NUM value to it, separated by a dash (`-`).
7. Here we build the image and tag.
8. Installing AWS CLI to interact with the ECR later.
9. Here we log into ECR
10. We tag the image build in step 7 with the ECR repository name.
11. Finally, we push the image to ECR.

Now we will deploy our helm charts. For this, we have a separate job deploy.



```yaml
deploy:
    docker: (1)
        - image: circleci/golang:1.10
    steps: (2)
      - checkout
      - run: (3)
          name: Install AWS cli
          command: export TZ=Europe/Minsk && sudo ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > sudo  /etc/timezone && sudo apt-get update && sudo apt-get install -y awscli
      - run: (4)
          name: Set the tag for the image, we will concatenate the app verson and circle build number with a `-` char in between
          command:  echo 'export TAG=$(cat VERSION)-$CIRCLE_PREVIOUS_BUILD_NUM' >> $BASH_ENV
      - run: (5)
          name: Install and confgure kubectl
          command: sudo curl -L https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl && sudo chmod +x /usr/local/bin/kubectl  
      - run: (6)
          name: Install and confgure kubectl aws-iam-authenticator
          command: curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-07-26/bin/linux/amd64/aws-iam-authenticator && sudo chmod +x ./aws-iam-authenticator && sudo cp ./aws-iam-authenticator /bin/aws-iam-authenticator
       - run: (7)
          name: Install latest awscli version
          command: sudo apt install unzip && curl "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" -o "awscli-bundle.zip" && unzip awscli-bundle.zip &&./awscli-bundle/install -b ~/bin/aws
      - run: (8)
          name: Get the kubeconfig file 
          command: export KUBECONFIG=$HOME/.kube/kubeconfig && /home/circleci/bin/aws eks --region $AWS_REGION update-kubeconfig --name $EKS_CLUSTER_NAME
      - run: (9)
          name: Install and configuire helm
          command: sudo curl -L https://storage.googleapis.com/kubernetes-helm/helm-v2.11.0-linux-amd64.tar.gz | tar xz && sudo mv linux-amd64/helm /bin/helm && sudo rm -rf linux-amd64
      - run: (10)
          name: Initialize helm
          command:  helm init --client-only --kubeconfig=$HOME/.kube/kubeconfig
      - run: (11)
          name: Install tiller plugin
          command: helm plugin install https://github.com/rimusz/helm-tiller --kubeconfig=$HOME/.kube/kubeconfig        
      - run: (12)
          name: Release helloapp using helm chart 
          command: bash scripts/release-helloapp.sh $TAG
```



1. Set the docker image inside which we want to execute the job.
2. Check out the code using `checkout` key
3. Install AWS CLI.
4. Setting the value of tag just like we did in case of build&pushImage job. Note that here we are using CIRCLE_PREVIOUS_BUILD_NUM variable which gives us the build number of build&pushImage job and ensures that the tag values are the same.
5. Download kubectl and making it executable.
6. Installing aws-iam-authenticator this is required because my k8s cluster is on EKS.
7. Here we install the latest version of AWS CLI, EKS is a relatively newer service from AWS and older versions of AWS CLI doesn’t have it.
8. Here we fetch the kubeconfig file. This step will vary depending upon where the k8s cluster has been set up. As my cluster is on EKS am getting the kubeconfig file via. AWS CLI similarly if your cluster in on GKE then you need to configure gcloud and use the command  `gcloud container clusters get-credentials <cluster-name> --zone=<zone-name>`. We can also have the kubeconfig file on some other secure storage system and fetch it from there.</zone-name></cluster-name>
9. Download Helm and make it executable
10. Initializing helm, note that we are initializing helm in client only mode so that it doesn’t start the tiller server.
11. Download the tillerless helm plugin
12. Execute the release-helloapp.sh shell script and pass it TAG value from step 4.

In the release-helloapp.sh script we first start tiller, after this, we check if the release is already present or not if it is present then we upgrade otherwise we make a new release. Here we override the value of tag for the image present in the chart by setting it to the tag of the newly built image, finally, we stop the tiller server.



```shell
#!/bin/bash
TAG=$1
echo "start tiller"
export KUBECONFIG=$HOME/.kube/kubeconfig
helm tiller start-ci
export HELM_HOST=127.0.0.1:44134
result=$(eval helm ls | grep helloapp) 
if [ $? -ne "0" ]; then 
   helm install --timeout 180 --name helloapp --set image.tag=$TAG charts/helloapp
else 
   helm upgrade --timeout 180 helloapp --set image.tag=$TAG charts/helloapp
fi
echo "stop tiller"
helm tiller stop 
```



The complete CircleCI config.yml file looks like:



```yaml
version: 2


jobs:
  build&pushImage:
    working_directory: /go/src/hello-app
    docker:
      - image: circleci/golang:1.10
    steps:
      - checkout
      - run:
          name: build the binary
          command: go build -o hello-app
      - setup_remote_docker:
          docker_layer_caching: true
      - run:
          name: Set the tag for the image, we will concatenate the app verson and circle build number with a `-` char in between
          command:  echo 'export TAG=$(cat VERSION)-$CIRCLE_BUILD_NUM' >> $BASH_ENV
      - run:
          name: Build the docker image
          command: docker build . -t ${CIRCLE_PROJECT_REPONAME}:$TAG
      - run:
          name: Install AWS cli
          command: export TZ=Europe/Minsk && sudo ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > sudo  /etc/timezone && sudo apt-get update && sudo apt-get install -y awscli
      - run:
          name: Login to ECR
          command: $(aws ecr get-login --region $AWS_REGION | sed -e 's/-e none//g')
      - run: 
          name: Tag the image with ECR repo name 
          command: docker tag ${CIRCLE_PROJECT_REPONAME}:$TAG ${HELLOAPP_ECR_REPO}:$TAG    
      - run: 
          name: Push the image the ECR repo
          command: docker push ${HELLOAPP_ECR_REPO}:$TAG
  deploy:
    docker:
        - image: circleci/golang:1.10
    steps:
      - attach_workspace:
          at: /tmp/workspace
      - checkout
      - run:
          name: Install AWS cli
          command: export TZ=Europe/Minsk && sudo ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > sudo  /etc/timezone && sudo apt-get update && sudo apt-get install -y awscli
      - run:
          name: Set the tag for the image, we will concatenate the app verson and circle build number with a `-` char in between
          command:  echo 'export TAG=$(cat VERSION)-$CIRCLE_PREVIOUS_BUILD_NUM' >> $BASH_ENV
      - run:
          name: Install and confgure kubectl
          command: sudo curl -L https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl && sudo chmod +x /usr/local/bin/kubectl  
      - run:
          name: Install and confgure kubectl aws-iam-authenticator
          command: curl -o aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.10.3/2018-07-26/bin/linux/amd64/aws-iam-authenticator && sudo chmod +x ./aws-iam-authenticator && sudo cp ./aws-iam-authenticator /bin/aws-iam-authenticator
       - run:
          name: Install latest awscli version
          command: sudo apt install unzip && curl "https://s3.amazonaws.com/aws-cli/awscli-bundle.zip" -o "awscli-bundle.zip" && unzip awscli-bundle.zip &&./awscli-bundle/install -b ~/bin/aws
      - run:
          name: Get the kubeconfig file 
          command: export KUBECONFIG=$HOME/.kube/kubeconfig && /home/circleci/bin/aws eks --region $AWS_REGION update-kubeconfig --name $EKS_CLUSTER_NAME
      - run:
          name: Install and configuire helm
          command: sudo curl -L https://storage.googleapis.com/kubernetes-helm/helm-v2.11.0-linux-amd64.tar.gz | tar xz && sudo mv linux-amd64/helm /bin/helm && sudo rm -rf linux-amd64
      - run:
          name: Initialize helm
          command:  helm init --client-only --kubeconfig=$HOME/.kube/kubeconfig
      - run:
          name: Install tiller plugin
          command: helm plugin install https://github.com/rimusz/helm-tiller --kubeconfig=$HOME/.kube/kubeconfig        
      - run:
          name: Release helloapp using helm chart 
          command: bash scripts/release-helloapp.sh $TAG
workflows:
  version: 2
  primary:
    jobs:
      - build&pushImage
      - deploy:
          requires:
            - build&pushImage
```



At the end of the file, we see the workflows, workflows control the order in which the jobs specified in the file are executed and establishes dependencies and conditions for the job. For example, we may want our deploy job trigger only after my build job is complete so we added a dependency between them. Similarly, we may want to exclude the jobs from running on some particular branch then we can specify those type of conditions as well.

We have used a few environment variables in our pipeline configuration some of them were created by us and some were made available by CircleCI. We created AWS_REGION, HELLOAPP_ECR_REPO, EKS_CLUSTER_NAME, AWS_ACCESS_KEY_ID & AWS_SECRET_ACCESS_KEY variables. These variables are set via. CircleCI web console by going to the projects settings. Other variables that we have used are made available by CircleCI as a part of its environment setup process. Complete list of environment variables set by CircleCI can be found [here](https://circleci.com/docs/2.0/env-vars/#built-in-environment-variables).

### Verify the working of the pipeline:

Once everything is set up properly then our application will get deployed on the k8s cluster and should be available for access. Get the external IP of the helloapp service and make a curl request to the hello endpoint



```shell
$ curl http://a31e25e7553af11e994620aebe144c51-242977608.us-west-2.elb.amazonaws.com/hello && printf "\n"


{"Msg":"Hello World"}
```



Now update the code and change the message “Hello World” to “Hello World Returns” and push your code. It will take a few minutes for the pipeline to complete execution and once it is complete make the curl request again to see the changes getting reflected.



```shell
$ curl http://a31e25e7553af11e994620aebe144c51-242977608.us-west-2.elb.amazonaws.com/hello && printf "\n"


{"Msg":"Hello World Returns"}
```



Also, verify that a new tag is also created for the helloapp docker image on ECR.

## Conclusion

In this blog post, we explored how we can set up a CI/CD pipeline for kubernetes and got basic exposure to CircleCI and Helm. Although helm is not absolutely necessary for building a pipeline, it has lots of benefits and is widely used across the industry. We can extend the pipeline to consider the cases where we have multiple environments like dev, staging & production and make the pipeline deploy the application to any of them depending upon some conditions. We can also add more jobs like integration tests. [All the codes used in the blog post are available here](https://github.com/velotiotech/helloapp).