# ukv-operator
A Go Operator for creating and managing instances of Unum UStore.

For more information on what is an Operator, visit 
- https://kubernetes.io/docs/concepts/architecture/controller/
- https://kubernetes.io/docs/concepts/extend-kubernetes/#extension-patterns
- https://kubernetes.io/docs/concepts/extend-kubernetes/operator/

This operator was built using Operator SDK. please see: 
- https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/


## Running on a cluster and testing

First, deploy the Operator and the Custom Resource Definition (CRD) - `make deploy`.

If you wish to debug see the "debugging" section below, and either scale down the controller-manager Deployment to 0 first, or use `make install` instead.

Next, deploy the user input Custom Resource example Config Map and UStore yaml:
```
oc apply -f config/samples/sample-config-ucset.yaml
oc apply -f config/samples/unistore_v1alpha1_ukv.yaml 
```
Note: there are more yamls under `config/samples`

### Cleanup
```
oc delete -f config/samples/unistore_v1alpha1_ukv.yaml 
make undeploy
```

## Building locally if needed
```
IMAGE_BUILDER=docker #its podman by default
IMG=<your local operator image to build>
make docker-build
make docker-push
```

### A push to main branch, triggers image build GH Action. image name: quay.io/itroyano/ukv-operator

## Operator Lifecycle Manager (OLM) Bundle and OpenShift Marketplace integration

To generate bundle manifests use `make bundle`.
Observe `config/manifests/bases/ukv-operator.clusterserviceversion.yaml` and the `bundle` folder.

To build the bundle image which can be tested, certified and pushed to marketplace use 
```
IMAGE_BUILDER=docker #its podman by default
IMAGE_TAG_BASE=<name of image. will be appended with -bundle>
make bundle-build
make bundle-push
```

This image can then be added to a test catalog for local testing only
```
make catalog-build
make catalog-push
```

For more information please see
-  https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/

### Debugging in VSCode
```
{
    "version": "0.2.0",
    "configurations": [
        
        {
            "name": "Launch file",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "main.go"
        },
    ]
}
```