# ukv-operator
A Go Operator for creating and managing instances of Unum UniStore UKV

### Installation docs found here https://www.unum.cloud/ukv/install.html https://github.com/unum-cloud/ukv#installation

### For more information visit https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/

### Building
```
IMAGE_BUILDER=docker #its podman by default
make docker-build
make docker-push
```

A push to main triggers image build GH Action.

#### image name: quay.io/itroyano/ukv-operator

### Testing
```
make deploy
oc apply -f config/samples/sample-config.yaml
oc apply -f config/samples/unistore_v1alpha1_ukv.yaml 
```
### Cleanup
```
oc delete -f config/samples/unistore_v1alpha1_ukv.yaml 
make undeploy
```