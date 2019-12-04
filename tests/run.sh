#!/bin/bash
function on_exit {
    kubectl delete namespace test
}
trap on_exit EXIT

set -e

# cluster component
kubectl get componentstatuses

# create test namespace and deploy busybox
kubectl create namespace test
kubectl apply -f tests/busybox.yaml
kubectl -n test wait --for condition=available deployment/busybox --timeout=60s

# services dns lookup from different namespaces and different servers
for s in $(kubectl -n kube-system get svc -o wide | awk 'FNR > 1 {print $1}')
do
    for p in $(kubectl -n test get pods -l app=busybox -o name); do kubectl -n test exec -ti $p  -- nslookup $s.kube-system; done
done

# pod reachability from different namespaces and different servers
for p in $(kubectl -n kube-system get pods -o wide | awk 'FNR > 1 {print $6}')
do
    for b in $(kubectl -n test get pods -l app=busybox -o name); do kubectl -n test exec -ti $b  -- ping -c 2 $p; done
done

# metric server
kubectl top node
for p in $(kubectl -n kube-system get pods -l k8s-app=kube-dns -o wide | awk 'FNR > 1 {print $1}'); do kubectl -n kube-system top pod $p; done

# csi-plugin create pvc
kubectl -n test apply -f tests/pvc-nginx.yaml
pvc=""
until [ "${pvc}" == 'Bound' ]; do
  echo "pvc not ready"
  sleep 3
  pvc=$(kubectl -n test get pvc -o jsonpath='{.items[*].status.phase}')
done
echo "pvc is ready now"

storage=$(kubectl -n test get pvc -o jsonpath='{.items[*].status.capacity.storage}')
if [ "${storage}" != '3Gi' ]; then
  echo "The pvc storage capacity does not match the requested capacity"
  exit 1
fi
echo "The pvc storage capacity matches the requested capacity"

# csi-plugin patch pvc (expand volume)
kubectl -n test patch pvc nginx-pv-claim --patch '{"spec": {"resources": {"requests": {"storage": "10Gi"}}}}'
sleep 20
storage=$(kubectl -n test get pvc -o jsonpath='{.items[*].status.capacity.storage}')
if [ "${storage}" != '10Gi' ]; then
  echo "The pvc storage capacity does not match the requested capacity"
  exit 1
fi
echo "The pvc storage capacity matches the requested capacity after storage expansion"

# deploy nginx ingress with hello-node application
kubectl -n test apply -f tests/nginx-ingress.yaml
kubectl -n test create deployment hello-node --image=gcr.io/hello-minikube-zero-install/hello-node
kubectl -n test expose deployment hello-node --port 8080
kubectl -n test expose deployment hello-node --port 8080 --name hello-node-nodeport --type=NodePort
kubectl -n test apply -f tests/hello-node-ingress.yaml
kubectl -n test wait --for condition=available deployment/hello-node --timeout=60s

loadbalancer_ip=""
until [ ! -z "${loadbalancer_ip}" ]; do
  echo "loadbalancer not ready"
  sleep 3
  loadbalancer_ip=$(kubectl get service -n test ingress-nginx -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
done
echo "loadbalancer ready"

# hello-node via Loadbalancer
curl -s "http://hello.${loadbalancer_ip}.nip.io" | grep -q 'Hello World!'
echo "loadbalancer replies correctly"

# hello-node via NodePort on all nodes
node_port=$(kubectl -n test get service hello-node-nodeport -o jsonpath='{.spec.ports[0].nodePort}')
# TODO: determine why nodes do not know about their external ip.
# See https://kubernetes.io/docs/tutorials/services/source-ip/#source-ip-for-services-with-type-nodeport
node_ips=$(kubectl get nodes -o jsonpath='{ $.items[*].status.addresses[?(@.type=="ExternalIP")].address }')
for node_ip in ${node_ips}; do
  curl -s "http://${node_ip}:${node_port}" | grep -q 'Hello World!'
  echo "NodePort ${node_ip}:${node_port} replies correctly"
done
