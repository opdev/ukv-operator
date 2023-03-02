# ukv-operator
A Go Operator for creating and managing instances of Unum UniStore UKV

### Installation docs found here https://www.unum.cloud/ukv/install.html https://github.com/unum-cloud/ukv#installation

### For more information visit https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/

### Testing
make deploy
oc apply -f config/samples/unistore_v1alpha1_ukv.yaml 

### Cleanup
oc delete -f config/samples/unistore_v1alpha1_ukv.yaml 
make undeploy