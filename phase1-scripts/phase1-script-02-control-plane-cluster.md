# Phase 1 — Script 02: Module 3 Control Plane + Module 4 Cluster (Lessons 05–09 detail, 10–13 outline)
## Kubernetes Andar Se — Chapter 0 / Bootstrap (Very Hard Way)

**Status:** Draft v1 — Rao performs on his own Proxmox VM, then recording
**Source:** Notion pages (full text fetched via Notion API, one by one) + iximiuz one-page summary. Syllabus/reference only — video mein iximiuz ki promotion NAHI, playground NAHI.
**Lab:** Rao's own Proxmox VM. Viewer WSL / kisi bhi OS par Linux VM par repeat kare.
**Pattern har lesson mein:** Download → Haath se Configure → systemd service → Observe. Pehle FAIL dikhao, phir fix karo.
**Script 01 se link:** Script 01 mein worker node tayyar hai (containerd 2.2.1 + kubelet v1.34.0, static Pod chal raha hai). Ab hum control-plane machine par jate hain. Worker aur control plane alag machines hain — yehi bottom-up approach hai.

## Version table (Script 01 se wahi, dobara lock)
Kubernetes v1.34.0, containerd 2.2.1, runc v1.4.0, nerdctl 2.2.1, CNI plugins v1.9.0, etcd v3.6.4, Flannel v0.27.2 + flannel CNI v1.7.1-flannel2, CoreDNS 1.12.2, crictl v1.34.0, kubeletctl v1.13, auger 1.0.3.

---

## LESSON 05 — etcd (Module 3: Control Plane — pehla component)

### Overview
- **etcd** distributed key-value store hai, Kubernetes cluster ke saare state ka primary data store. Configuration, state, metadata — sab yahin hai. Ye cluster ka "source of truth" hai.
- Originally CoreOS ne banaya, ab CNCF maintain karta hai. Highly available, consistent, reliable hona zaroori hai, kyunki yehi woh jagah hai jahan cluster ka asal sach rehta hai.

### Objectives
- etcd ka role samajhna.
- etcd ko scratch se install aur configure karna.
- `etcdctl` se etcd ke sath interact karna seekhna.
- etcd ko TLS certificates se secure karna.

### 5.1 etcd kya hai? Raft aur quorum
- etcd **Raft consensus algorithm** use karta hai taake kai nodes par data consistent rahe.
- Cluster tab tak chalta hai jab tak majority (quorum) healthy ho. 1 member wala cluster seekhne ke liye hai — 1 mar jaye to sab khatam. 3 members mein 1 mar sakta hai, 5 mein 2 mar sakte hain. Yehi wajah hai ke production mein odd number rakhte hain.
- **Sab se zaroori baat:** sirf kube-apiserver etcd se seedha baat karta hai. Koi aur component seedha etcd ko nahi chhoota. Yehi access control ka pehla darwaza hai.

### 5.2 etcd install karna — step by step
```bash
ETCD_VERSION=v3.6.4
```
```bash
curl -fsSLO https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz
tar xzvof etcd-${ETCD_VERSION}-linux-amd64.tar.gz
sudo install -m 755 etcd-${ETCD_VERSION}-linux-amd64/{etcd,etcdctl,etcdutl} /usr/local/bin
etcdctl completion bash | sudo tee /etc/bash_completion.d/etcdctl
```
- etcd ke liye alag system user banao (etcd apni files ka malik ho, root na ho):
```bash
sudo adduser \
    --system \
    --group \
    --disabled-login \
    --disabled-password \
    --home /var/lib/etcd \
    etcd
```
- systemd unit file:
```bash
sudo wget -O /etc/systemd/system/etcd.service https://labs.iximiuz.com/content/files/courses/kubernetes-the-very-hard-way-0cbfd997/03-control-plane/01-etcd/__static__/etcd.service?v=1770643331
```
- **Rao ke liye note (video mein URL promotion mat dikhao):** Apni copy rakho, jaise Script 01 mein kubelet unit ke liye kaha tha.
- Service start karo:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now etcd
```

### 5.3 Pehle plain HTTP par — dekhna ke andar kya hai
- Pehle etcd plain HTTP par chalta hai, taake tum dekh sako ke andar kya store hota hai. Ye seekhne ke liye hai, production mein nahi.
- Check karo:
```bash
curl http://127.0.0.1:2379/health
etcdctl endpoint health
```
- Key-value likho aur parho:
```bash
etcdctl put foo bar
etcdctl get foo
etcdctl get foo --print-value-only
```
- Hierarchical keys samjhao:
```bash
etcdctl put /foo bar
etcdctl put /foo/bar baz
etcdctl put /foo/bar/baz bat
etcdctl get --prefix /foo --keys-only
```
- **Bolne wali line:** "Yehi key-value pattern hai jo aage /registry ke neeche Kubernetes objects ke liye use hoga. Prefix se poora tree nikal sakte ho."

### 5.4 TLS se secure karna — apna CA, server cert, client cert
- etcd ka apna CA hai, Kubernetes wale CA se **alag**. Sirf 2 certificates sign karta hai: etcd ka server cert, aur ek client cert (jo etcdctl aur apiserver share karenge).
- CA banao:
```bash
sudo mkdir -p /etc/etcd/pki
cd /etc/etcd/pki
```
```bash
sudo openssl genrsa -out ca.key 4096
sudo openssl req -x509 -new -nodes -key ca.key -out ca.crt -subj "/CN=etcd" -sha256 -days 3650
```
- Server cert ke liye config file (SANs mein localhost, hostname, 127.0.0.1, ::1, aur host ka eth IP):
```bash
cat <<EOF | sudo tee server.cnf
[ req ]
default_bits       = 2048
distinguished_name = req_distinguished_name
req_extensions     = req_ext
prompt             = no

[ req_distinguished_name ]
CN = server

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = localhost
DNS.2 = $(hostname)
IP.1  = 127.0.0.1
IP.2  = ::1
IP.3 = $(ip -o -4 addr show | grep 'eth' | awk '{split($4,a,"/"); print a[1]}' | paste -sd,)
EOF
```
```bash
sudo openssl genrsa -out server.key 2048
sudo openssl req -new -key server.key -out server.csr -config server.cnf
sudo openssl x509 -req -in server.csr -out server.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365 -extfile server.cnf -extensions req_ext
```
- Ownership theek karo:
```bash
sudo chown -R etcd:etcd .
```
- Client cert: CA se client certificate banao — course ke baaki certs wala hi pattern hai (genrsa → req → x509 -req -CA), CN=client:
```bash
sudo openssl genrsa -out client.key 2048
sudo openssl req -new -key client.key -out client.csr -subj "/CN=client"
sudo openssl x509 -req -in client.csr -out client.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365
```
- Client key sab users ko readable rakho taake etcdctl chala sake:
```bash
sudo chmod 644 client.key
```
- **Perform karke verify karo:** `etcdctl endpoint health` TLS ke baad chalna chahiye; bina client cert ke `curl https://127.0.0.1:2379/health` fail hona chahiye — yehi mTLS ka proof hai.
- etcd ko TLS par lao:
```bash
cat <<EOF | sudo tee -a /etc/default/etcd

ETCD_LISTEN_CLIENT_URLS=https://0.0.0.0:2379

ETCD_CLIENT_CERT_AUTH=true
ETCD_CERT_FILE=/etc/etcd/pki/server.crt
ETCD_KEY_FILE=/etc/etcd/pki/server.key
ETCD_TRUSTED_CA_FILE=/etc/etcd/pki/ca.crt

ETCD_NAME=$(hostname)
ETCD_ADVERTISE_CLIENT_URLS=https://$(hostname):2379

EOF
```
```bash
sudo systemctl restart etcd
```
- etcdctl ke liye environment variables set karo (taake har baar cert flags na likhne paden):
```bash
cat <<EOF | tee -a $HOME/.bashrc $HOME/.profile

export ETCDCTL_CACERT=/etc/etcd/pki/ca.crt
export ETCDCTL_CERT=/etc/etcd/pki/client.crt
export ETCDCTL_KEY=/etc/etcd/pki/client.key
export ETCDCTL_ENDPOINTS=https://127.0.0.1:2379
EOF
```
- Check karo:
```bash
bash --login -c "etcdctl endpoint health"
```
- **Samjhao — TLS sirf encryption nahi hai:** mTLS se client authentication bhi hoti hai. `ETCD_CLIENT_CERT_AUTH=true` ka matlab hai: bina client cert ke koi etcd ko chhoo bhi nahi sakta. Yehi wajah hai ke sirf apiserver ke paas client cert hai.

### 5.5 Summary — etcd
- etcd cluster state ka single source of truth hai, Raft se consistent, quorum se zinda.
- Sirf apiserver us se baat karta hai.
- TLS + mTLS se encrypted aur authenticated hai. Apna CA alag hai.
- **Dependency yaad rakho:** etcd ke baghair apiserver kuch store nahi kar sakta. etcd mar jaye to cluster ka dimagh mar jata hai — is liye Phase 2 mein HA aur backup aaye ga.

---

## LESSON 06 — kube-apiserver (Module 3 — control plane ka dil)

### Overview
- **kube-apiserver** control plane ka frontend aur central hub hai. Cluster ki har operation isi se guzarti hai. Yehi woh component hai jo etcd se seedha baat karta hai, aur baaki sab us se baat karte hain.
- Har request 3 darwazon se guzarti hai: **authentication** (tum kaun ho), **authorization** (tum kya kar sakte ho), **validation** (request sahi hai ya nahi) — phir ja kar state tak pahunchti hai.

### Objectives
- apiserver ka role samajhna, scratch se install/configure karna.
- Authentication aur authorization configure karna.
- `curl` se seedha Kubernetes API se baat karna.
- `kubectl` install aur configure karna.
- TLS certificates se secure karna.
- Dekhna ke apiserver data ko etcd mein kaise store karta hai.

### 6.1 Install karna
```bash
KUBE_VERSION=v1.34.0
```
```bash
curl -fsSLO https://dl.k8s.io/${KUBE_VERSION}/bin/linux/amd64/kube-apiserver
sudo install -m 755 kube-apiserver /usr/local/bin
```
```bash
sudo wget -O /etc/systemd/system/kube-apiserver.service https://labs.iximiuz.com/content/files/courses/kubernetes-the-very-hard-way-0cbfd997/03-control-plane/02-kube-apiserver/__static__/kube-apiserver.service?v=1771369104
```
- **Rao note:** Unit file ki apni copy rakho, video mein iximiuz URL mat dikhao.
- Service-account keys banao:
```bash
sudo mkdir -p /etc/kubernetes/pki
cd /etc/kubernetes/pki
```
```bash
sudo openssl genrsa -out sa.key 2048
sudo openssl rsa -in sa.key -pubout -out sa.pub
```
```bash
cd
```
- Pehli baar sirf ye flags kaafi hain (unit file mein ye ExecStart hai):
```
--service-cluster-ip-range=10.96.0.0/12
--service-account-issuer=https://kubernetes.default.svc.cluster.local
--service-account-key-file=/etc/kubernetes/pki/sa.pub
--service-account-signing-key-file=/etc/kubernetes/pki/sa.key
--etcd-cafile=/etc/etcd/pki/ca.crt
--etcd-certfile=/etc/etcd/pki/client.crt
--etcd-keyfile=/etc/etcd/pki/client.key
--etcd-servers=https://127.0.0.1:2379
```
- **Samjhao:** `10.96.0.0/12` woh range hai jahan se Service IPs milengi. Pehla address `10.96.0.1` `kubernetes` Service ko jaye ga — is liye woh apiserver cert ke SANs mein hona chahiye (neeche dekho).
- Service start karo:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kube-apiserver
```

### 6.2 Authentication aur authorization — staged, pehle fail dikhao
- Default mein anonymous + AlwaysAllow ka combination invalid hai — apiserver anonymous ko khud disable kar deta hai. Is liye pehle token lagate hain.
- Static token file banao:
```bash
echo "iximiuz,admin,admin,system:masters" | sudo tee /etc/kubernetes/tokens.csv
```
- Unit file mein ye flags add karo:
```
--anonymous-auth=false
--token-auth-file=/etc/kubernetes/tokens.csv
```
- Phir authorization add karo:
```
--authorization-mode=Node,RBAC
```
- **Samjhao:** Authorization modes order mein evaluate hote hain, aur default deny hai. `Node` kubelet ko apne Node tak limit karta hai, `RBAC` baaki sab ko role se control karta hai.
- Restart karo:
```bash
sudo systemctl daemon-reload
sudo systemctl restart kube-apiserver
```

### 6.3 curl se API se seedha baat karna
- Bina token ke fail hona chahiye — dikhao:
```bash
curl -f -k https://localhost:6443/api/v1/namespaces
```
- Token ke sath chalega:
```bash
curl -f -k -H "Authorization: Bearer iximiuz" https://localhost:6443/api/v1/namespaces
```
```bash
curl -f -k https://127.0.0.1:6443/api/v1/namespaces \
    -H "Authorization: Bearer iximiuz" \
    | jq -r '.items[].metadata.name'
```
- Namespace banao, parho, patch karo — sab curl se:
```bash
curl -f -k https://127.0.0.1:6443/api/v1/namespaces \
    -H "Authorization: Bearer iximiuz" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{
      "apiVersion": "v1",
      "kind": "Namespace",
      "metadata": {
        "name": "test-curl"
      }
    }'
```
```bash
curl -f -k https://127.0.0.1:6443/api/v1/namespaces/test-curl \
    -H "Authorization: Bearer iximiuz" | jq
```
```bash
curl -f -k https://127.0.0.1:6443/api/v1/namespaces/test-curl \
    -H "Authorization: Bearer iximiuz" \
    -X PATCH \
    -H "Content-Type: application/merge-patch+json" \
    -d '{"metadata": {"labels": {"foo": "bar"}}}'
```
```bash
curl -f -k https://127.0.0.1:6443/api/v1/namespaces/test-curl \
    -H "Authorization: Bearer iximiuz" | jq '.metadata.labels'
```
- **Bolne wali line:** "kubectl bhi andar se yahi karta hai — HTTP request bhejta hai. kubectl koi jaadu nahi hai, ye API ka dostana chehra hai."

### 6.4 kubectl — CLI install aur pehla kubeconfig
```bash
KUBE_VERSION=v1.34.0
```
```bash
curl -fsSLO https://dl.k8s.io/${KUBE_VERSION}/bin/linux/amd64/kubectl
sudo install -m 755 kubectl /usr/local/bin
kubectl completion bash | sudo tee /etc/bash_completion.d/kubectl
```
- Pehla kubeconfig — token + insecure-skip-tls-verify (seekhne ke liye, production mein **kabhi nahi**):
```bash
kubectl config set-cluster default \
    --insecure-skip-tls-verify \
    --server=https://localhost:6443
kubectl config set-credentials default --token=iximiuz
kubectl config set-context default --cluster=default --user=default
kubectl config use-context default
```
- Check karo:
```bash
kubectl cluster-info
kubectl get namespaces
kubectl create namespace test-kubectl
kubectl get namespace test-kubectl
```

### 6.5 TLS se secure karna — Kubernetes CA, apiserver cert, admin cert
- Kubernetes CA banao (ye etcd wale CA se **alag** hai):
```bash
sudo mkdir -p /etc/kubernetes/pki
cd /etc/kubernetes/pki
```
```bash
sudo openssl genrsa -out ca.key 2048
sudo openssl req -x509 -new -nodes -key ca.key -out ca.crt -subj "/CN=kubernetes" -sha256 -days 3650
```
- apiserver cert ke liye config (SANs yaad rakho — yehi woh list hai jo aksar bhool jati hai):
```bash
cat <<EOF | sudo tee apiserver.cnf
[ req ]
default_bits       = 2048
distinguished_name = req_distinguished_name
req_extensions     = req_ext
prompt             = no

[ req_distinguished_name ]
CN = kube-apiserver

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = localhost
DNS.2 = kubernetes
DNS.3 = kubernetes.default
DNS.4 = kubernetes.default.svc
DNS.5 = kubernetes.default.svc.cluster.local
DNS.6 = control-plane
IP.1  = 127.0.0.1
IP.2  = ::1
IP.3  = 10.96.0.1
IP.4  = $(ip -o -4 addr show | grep 'eth' | awk '{split($4,a,"/"); print a[1]}' | paste -sd,)
EOF
```
```bash
sudo openssl genrsa -out apiserver.key 2048
sudo openssl req -new -key apiserver.key -out apiserver.csr -config apiserver.cnf
sudo openssl x509 -req -in apiserver.csr -out apiserver.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365 -extfile apiserver.cnf -extensions req_ext
```
- Admin cert — `O=system:masters` yaad rakho, yehi RBAC group hai:
```bash
sudo openssl genrsa -out admin.key 2048
sudo openssl req -new -key admin.key -out admin.csr -subj "/CN=admin/O=system:masters"
sudo openssl x509 -req -in admin.csr -out admin.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365
```
```bash
sudo chmod 644 admin.key
```
- **Samjhao — O field ka jaadu:** Certificate mein `O=system:masters` likha hai, RBAC us field ko user ka group parhta hai. `system:masters` group ko cluster-admin milta hai. Is liye admin cert se har kaam ho jata hai — aur is liye is cert ko sambhal kar rakhna hai.
- Unit file mein TLS flags add karo:
```
--tls-cert-file=/etc/kubernetes/pki/apiserver.crt
--tls-private-key-file=/etc/kubernetes/pki/apiserver.key
--client-ca-file=/etc/kubernetes/pki/ca.crt
```
- **Note:** Bina TLS flags ke apiserver khud self-signed cert bana leta hai — unreliable authentication, client warnings. Is liye hum apne cert lagate hain.
- Restart karo, phir kubectl ko token se hata kar CA + client cert par lao (`--token=""` karke). Perform karke exact kubectl config commands yahan likh do.
- Check karo: `kubectl cluster-info` ab CA verify karke chalna chahiye, bina `--insecure-skip-tls-verify` ke.

### 6.6 Data etcd mein — /registry ke neeche protobuf
- Namespace banaya tha, woh etcd mein kahan gaya? Dekho:
```bash
etcdctl get --prefix /registry --keys-only
etcdctl get --prefix /registry/namespaces --keys-only
etcdctl get /registry/namespaces/default --print-value-only
```
- Objects **protobuf** mein store hote hain, plain JSON nahi. Is liye seedha parhna mushkil hai.
- **auger 1.0.3** se decode karo:
```bash
auger decode
auger decode -o json | jq
```
- **Bolne wali line:** "kubectl kehte hai ke namespace ban gaya. etcd kehte hai ke /registry/namespaces/test-curl ke neeche ek protobuf blob para hai. Dono ek hi sach hain, do zubaanon mein."

### 6.7 Summary — apiserver
- apiserver hi woh darwaza hai jis se cluster state tak jate hain. Sirf wahi etcd se baat karta hai.
- Har request authn → authz → validation se guzarti hai.
- TLS encryption + certificate authentication dono deta hai.
- kubectl API ka dostana chehra hai, kubeconfig us ka address book hai.
- **Dependency yaad rakho:** etcd ke baghair apiserver kuch store nahi kar sakta. apiserver ke baghair koi component kuch nahi kar sakta — scheduler, controller-manager, kubelet sab isi se baat karte hain.
- Config paths: `/etc/kubernetes/pki/{ca.crt,ca.key,apiserver.crt,apiserver.key,admin.crt,admin.key,sa.pub,sa.key}`, `/etc/kubernetes/tokens.csv`, `/etc/systemd/system/kube-apiserver.service`, `/etc/etcd/pki/{ca.crt,client.crt,client.key}`. Port `:6443`.

---

## LESSON 07 — kube-scheduler (Module 3)

### Overview
- **kube-scheduler** ka kaam hai naye banne wale Pods ko Nodes par assign karna. Bas. Woh Pod nahi chalata, woh sirf decide karta hai ke Pod kis Node par chalega.

### Objectives
- Scheduler ka role samajhna.
- Manual Pod assignment dekhna (direct + Binding).
- Scheduler ko scratch se install/configure karna.
- Automated scheduling ko action mein dekhna.

### 7.1 Scheduler install karne se PEHLE — woh sirf ek field likhta hai
- Pehle ye demo karo: Pod banao, us ka `spec.nodeName` khaali hoga:
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: podinfo-unscheduled
spec:
  automountServiceAccountToken: false
  containers:
    - name: podinfo
      image: ghcr.io/stefanprodan/podinfo:latest
EOF
```
```bash
kubectl get pod podinfo-unscheduled -o jsonpath='{.spec.nodeName}'
```
- **Output khaali hai — Pod unscheduled hai.**
- Haath se ek fake Node banao (abhi koi asli worker nahi hai):
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Node
metadata:
  name: worker-01
status:
  conditions:
    - type: Ready
      status: "True"
EOF
```
- Direct `nodeName` se Pod banao:
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: podinfo-node-name
spec:
  nodeName: worker-01
  automountServiceAccountToken: false
  containers:
    - name: podinfo
      image: ghcr.io/stefanprodan/podinfo:latest
EOF
```
```bash
kubectl get pod podinfo-node-name -o jsonpath='{.spec.nodeName}'
```
- Ab manual **Binding** se unscheduled Pod ko assign karo — yehi woh kaam hai jo scheduler karta hai:
```bash
curl -f -k https://127.0.0.1:6443/api/v1/namespaces/default/pods/podinfo-unscheduled/binding \
    --cacert /etc/kubernetes/pki/ca.crt \
    --cert /etc/kubernetes/pki/admin.crt \
    --key /etc/kubernetes/pki/admin.key \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{
      "apiVersion": "v1",
      "kind": "Binding",
      "metadata": {
        "name": "podinfo-unscheduled"
      },
      "target": {
        "apiVersion": "v1",
        "kind": "Node",
        "name": "worker-01"
      }
    }'
```
```bash
kubectl get pod podinfo-unscheduled -o jsonpath='{.spec.nodeName}'
kubectl get pods -o wide
```
- **Bolne wali line:** "Scheduler ka poora kaam yehi hai — Binding subresource se `spec.nodeName` likhna. Pod mostly immutable hoti hai, is liye poora Pod update nahi, sirf Binding se node set karte hain. Ye samajh lo to scheduling ka aadha raaz khul gaya."

### 7.2 Scheduler install karna
```bash
KUBE_VERSION=v1.34.0
```
```bash
curl -fsSLO https://dl.k8s.io/${KUBE_VERSION?}/bin/linux/amd64/kube-scheduler
sudo install -m 755 kube-scheduler /usr/local/bin
```
```bash
sudo wget -O /etc/systemd/system/kube-scheduler.service https://labs.iximiuz.com/content/files/courses/kubernetes-the-very-hard-way-0cbfd997/03-control-plane/03-kube-scheduler/__static__/kube-scheduler.service?v=1771369104
```
- **Rao note:** Unit file ki apni copy rakho, video mein URL mat dikhao.
- Scheduler ke liye certificate banao — `CN=system:kube-scheduler`:
```bash
cd /etc/kubernetes/pki
sudo openssl genrsa -out scheduler.key 2048
sudo openssl req -new -key scheduler.key -out scheduler.csr -subj "/CN=system:kube-scheduler"
sudo openssl x509 -req -in scheduler.csr -out scheduler.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365
```
```bash
cd
```
- kubeconfig banao — `/etc/kubernetes/scheduler.conf`:
```bash
sudo kubectl config set-cluster default \
    --kubeconfig=/etc/kubernetes/scheduler.conf \
    --certificate-authority=/etc/kubernetes/pki/ca.crt \
    --embed-certs=true \
    --server=https://127.0.0.1:6443
sudo kubectl config set-credentials default \
    --kubeconfig=/etc/kubernetes/scheduler.conf \
    --client-certificate=/etc/kubernetes/pki/scheduler.crt \
    --client-key=/etc/kubernetes/pki/scheduler.key \
    --embed-certs=true
sudo kubectl config set-context default \
    --kubeconfig=/etc/kubernetes/scheduler.conf \
    --cluster=default \
    --user=default
sudo kubectl config use-context default \
    --kubeconfig=/etc/kubernetes/scheduler.conf
```
- **Samjhao — `--embed-certs=true` kyun:** kubeconfig file mein certs embed ho jate hain, alag files ka rasta nahi rakhna padta. File kahin bhi le jao, certs sath hain.
- Service start karo:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kube-scheduler
```

### 7.3 Scheduling dekhna
- **Note:** Extracted Notion mein is check wale Pod mein bhi `nodeName: worker-01` likha hai — perform karte waqt asli test ye hona chahiye: `nodeName` ke **baghair** Pod banao, phir dekho ke scheduler khud `nodeName` likhta hai ya nahi. Agar tum `nodeName` likh kar banao ge to scheduler ka kaam test hi nahi hoga. Is liye check ko is tarah perform karo:
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: podinfo-scheduler
spec:
  automountServiceAccountToken: false
  containers:
    - name: podinfo
      image: ghcr.io/stefanprodan/podinfo:latest
EOF
```
```bash
kubectl get pod podinfo-scheduler -o jsonpath='{.spec.nodeName}'
```
- **Expected:** `worker-01` aaye. Pod **Pending** rahe ga — kyunki us Node se koi kubelet attached nahi hai. Ye theek hai, yehi lesson hai: scheduler ne apna kaam kar diya (node chun liya), ab kubelet ka kaam baaki hai.
- **Samjhao — scheduler decide kaise karta hai:** Unscheduled Pods ko watch karta hai, phir resources, node affinity, Pod priorities aur doosre factors dekh kar best Node chunta hai. Default scheduler ko custom scheduler se replace bhi kar sakte ho — scheduling framework is ke liye bana hai.

### 7.4 Summary — scheduler
- Scheduler sirf ek field likhta hai: `spec.nodeName`, Binding se.
- Woh Pod nahi chalata, woh sirf jagah decide karta hai.
- **Dependency yaad rakho:** Scheduler apiserver ke baghair kuch nahi dekh sakta. Scheduler ke baghair Pod unscheduled rehta hai — banta hai, chalta nahi.

---

## LESSON 08 — kube-controller-manager (Module 3)

### Overview
- **kube-controller-manager** woh component hai jo Kubernetes ko "simple data store" se "self-healing, declarative system" banata hai. Ye core control loops chalata hai.

### Objectives
- Reconciliation loops samajhna — desired state vs actual state.
- Kaunse controllers bundled hain, kya karte hain.
- Scratch se install/configure karna.
- Action mein dekhna: ReplicaSets, self-healing, Deployments.

### 8.1 Reconciliation loop kya hai?
- Har controller ek loop mein kaam karta hai: **Observe → Compare → Act.**
- Desired state (tum ne kaha 3 replicas chahiye) vs Actual state (kitne chal rahe hain) — farq ho to controller farq khatam karta hai.
- Controllers 3 qisam ke hote hain: **Core** (ReplicaSet, Deployment, Job, Namespace, ServiceAccount, Node Lifecycle), **Cloud** (cloud-controller-manager, cloud-specific logic), **Custom** (operators, tumhare apne resources ke liye).
- **Cascading composition samjhao:** Deployment → ReplicaSet banata hai → ReplicaSet Pod banata hai → Scheduler Node chunta hai → kubelet container chalata hai. Har controller sirf apna kaam karta hai, doosre par bharosa karta hai.

### 8.2 Controller ke baghair zindagi — pehle demo karo
- ReplicaSet banao, 3 replicas:
```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: podinfo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: podinfo
  template:
    metadata:
      labels:
        app: podinfo
    spec:
      automountServiceAccountToken: false
      containers:
        - name: podinfo
          image: ghcr.io/stefanprodan/podinfo:latest
EOF
```
```bash
kubectl get replicaset podinfo
kubectl get pods -l app=podinfo
```
- **Output: No resources found.** API server ne ReplicaSet store kar liya, lekin Pod kisi ne nahi banaye. Kyunki Pod banana controller ka kaam hai, apiserver ka nahi. Yehi farq hai jo log bhool jate hain.

### 8.3 Install karna
```bash
KUBE_VERSION=v1.34.0
```
```bash
curl -fsSLO https://dl.k8s.io/${KUBE_VERSION?}/bin/linux/amd64/kube-controller-manager
sudo install -m 755 kube-controller-manager /usr/local/bin
```
```bash
sudo wget -O /etc/systemd/system/kube-controller-manager.service https://labs.iximiuz.com/content/files/courses/kubernetes-the-very-hard-way-0cbfd997/03-control-plane/04-kube-controller-manager/__static__/kube-controller-manager.service?v=1772116344
```
- **Rao note:** Unit file ki apni copy rakho.
- Certificate banao — `CN=system:kube-controller-manager`:
```bash
(
cd /etc/kubernetes/pki
sudo openssl genrsa -out controller-manager.key 2048
sudo openssl req -new -key controller-manager.key -out controller-manager.csr -subj "/CN=system:kube-controller-manager"
sudo openssl x509 -req -in controller-manager.csr -out controller-manager.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365
)
```
- kubeconfig banao — `/etc/kubernetes/controller-manager.conf` (scheduler wale pattern se):
```bash
sudo kubectl config set-cluster default \
    --kubeconfig=/etc/kubernetes/controller-manager.conf \
    --certificate-authority=/etc/kubernetes/pki/ca.crt \
    --embed-certs=true \
    --server=https://127.0.0.1:6443
sudo kubectl config set-credentials default \
    --kubeconfig=/etc/kubernetes/controller-manager.conf \
    --client-certificate=/etc/kubernetes/pki/controller-manager.crt \
    --client-key=/etc/kubernetes/pki/controller-manager.key \
    --embed-certs=true
sudo kubectl config set-context default \
    --kubeconfig=/etc/kubernetes/controller-manager.conf \
    --cluster=default \
    --user=default
sudo kubectl config use-context default \
    --kubeconfig=/etc/kubernetes/controller-manager.conf
```
- **Samjhao — controller-manager ke flags mein 3 cheezein khaas hain:** `--cluster-signing-cert-file` (CSR sign karne ke liye CA cert), CA private key (CSR sign karne ke liye), aur ServiceAccount tokens sign karne ke liye private key. Har controller apne service account se authenticate karta hai.
- Service start karo:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kube-controller-manager
```

### 8.4 Controllers in action — self-healing aur cascading
- Ab wahi ReplicaSet ke Pods dikhen ge:
```bash
kubectl get pods -l app=podinfo
```
- **Expected: 3 Pods.** Ek Pod delete karo, replacement aaye ga:
```bash
POD_NAME=$(kubectl get pods -l app=podinfo -o jsonpath='{.items[0].metadata.name}')
kubectl delete pod $POD_NAME
kubectl get pods -l app=podinfo
```
- **Bolne wali line:** "Tum ne Pod delete kiya, controller ne wapas bana diya. Tum ne kuch nahi kaha, controller ne desired state dekhi (3 chahiye), actual dekha (2 hain), farq khatam kar diya. Yehi self-healing hai."
- Deployment se cascading dekho:
```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      automountServiceAccountToken: false
      containers:
        - name: nginx
          image: nginx:latest
EOF
```
```bash
kubectl get deployment nginx
kubectl get replicasets -l app=nginx
kubectl get pods -l app=nginx
```
- **Samjhao:** Deployment ne ReplicaSet banaya, ReplicaSet ne Pods banaye. Tum ne sirf Deployment kaha tha, baaki controllers ne apna apna kaam kiya.

### 8.5 Summary — controller-manager
- Ek binary mein darjanon controllers hain, operational simplicity ke liye.
- Reconciliation loop hi Kubernetes ko declarative banata hai.
- **Dependency yaad rakho:** controller-manager ke baghair API sirf store hai, system nahi. Scheduler ke baghair Pod assign nahi hota, controller-manager ke baghair Pod banta hi nahi. Dono apiserver par dependent hain.
- **Aage ka ishara:** controller-manager CSR sign bhi karta hai (Lesson 09 mein kaam aaye ga) aur Node ko Pod network ka /24 slice bhi deta hai (`--allocate-node-cidrs`, 10.244.0.0/16).

---

## LESSON 09 — Joining Nodes to the Cluster (Module 4 — pehla cluster)

### Overview
- Ab tak worker node alag tha, control plane alag tha. Is lesson mein dono ko jor kar **functioning cluster** banate hain. Functioning ka matlab hai: cluster workload chala sakta hai. Networking abhi baaki hai, woh Lesson 10 mein aaye ga.

### Objectives
- "Join" karne ka matlab samajhna.
- Control plane ko configure karna ke worker nodes ko accept aur authenticate kare.
- **TLS bootstrapping** se worker node ko securely register karna.
- Node registration aur certificate approval ko observe karna.
- Worker par workload deploy karke dekhna.

### 9.1 Forming a cluster — trust problem kya hai?
- Worker node ko control plane par bharosa karna hai, aur control plane ko worker par. Dono ek doosre ko pehle se nahi jante.
- Control plane components (scheduler, controller-manager) ke certs hum ne haath se banaye aur usi machine par rakhe — woh scale nahi karta. Socho 100 nodes hon to har ek ka cert haath se banao ge?
- **TLS bootstrapping** is ka hal hai: kubelet ek temporary, low-privilege token se apiserver se ek proper client certificate maangta hai. Certificate approve aur issue ho jaye to kubelet usi se baat karta hai.

### 9.2 Control plane tayyar karo — 3 cheezein chahiye
1. Apiserver ko kubelet se baat karne ke liye **client certificate**.
2. Naye kubelet ke liye **bootstrap token authentication**.
3. Bootstrap token ke liye **RBAC rules** taake woh certificate maang sake.
- Apiserver-kubelet-client cert banao — `CN=kube-apiserver-kubelet-client/O=system:masters`:
```bash
(
cd /etc/kubernetes/pki
sudo openssl genrsa -out apiserver-kubelet-client.key 2048
sudo openssl req -new -key apiserver-kubelet-client.key -out apiserver-kubelet-client.csr -subj "/CN=kube-apiserver-kubelet-client/O=system:masters"
sudo openssl x509 -req -in apiserver-kubelet-client.csr -out apiserver-kubelet-client.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365
)
```
- Apiserver config update karo (purane flags + 3 naye):
```
--kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt
--kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key
--enable-bootstrap-token-auth=true
```
- Poori list (yaad rakho, purane flags bhi rehte hain):
```
--service-cluster-ip-range=10.96.0.0/12
--service-account-issuer=https://kubernetes.default.svc.cluster.local
--service-account-key-file=/etc/kubernetes/pki/sa.pub
--service-account-signing-key-file=/etc/kubernetes/pki/sa.key
--etcd-cafile=/etc/etcd/pki/ca.crt
--etcd-certfile=/etc/etcd/pki/client.crt
--etcd-keyfile=/etc/etcd/pki/client.key
--etcd-servers=https://127.0.0.1:2379
--anonymous-auth=false
--token-auth-file=/etc/kubernetes/tokens.csv
--authorization-mode=Node,RBAC
--tls-cert-file=/etc/kubernetes/pki/apiserver.crt
--tls-private-key-file=/etc/kubernetes/pki/apiserver.key
--client-ca-file=/etc/kubernetes/pki/ca.crt
--kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt
--kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key
--enable-bootstrap-token-auth=true
```
- Restart karo:
```bash
sudo systemctl daemon-reload
sudo systemctl restart kube-apiserver
kubectl cluster-info
```
- **Samjhao:** Apiserver ko kubelet se baat karne ke liye bhi client cert chahiye, warna woh channel insecure hota. Dono taraf se authentication hai — mutual auth.

### 9.3 Bootstrap token banao
- Token ka format hai `[token-id].[token-secret]`, masalan `abcdef.0123456789abcdef`. Ye `kube-system` namespace mein ek Secret ki tarah store hota hai, special type ke sath:
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: bootstrap-token-abcdef
  namespace: kube-system
type: bootstrap.kubernetes.io/token
stringData:
  token-id: abcdef
  token-secret: 0123456789abcdef
  usage-bootstrap-authentication: "true"
  description: "Bootstrap token for kubelets to authenticate to the API server"
EOF
```
- **Production note bolna zaroori hai:** Production mein bootstrap tokens short-lived hone chahiye, regularly rotate hone chahiye, aur har kubelet ke liye unique hone chahiye. kubeadm har join ke liye unique, expiring token banata hai. Is course mein hum seekhne ke liye fixed token use kar rahe hain.

### 9.4 RBAC — 3 ClusterRoleBindings
- Bootstrap token kubelet ko `system:bootstrappers` group ka member banata hai. By default is group ko kuch karne ki permission nahi hai. Is liye 3 bindings chahiye:
- **1. kubelet-bootstrap** — CSR banana ki permission:
```bash
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubelet-bootstrap
subjects:
  - kind: Group
    name: system:bootstrappers
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: system:node-bootstrapper
  apiGroup: rbac.authorization.k8s.io
EOF
```
- **2. node-autoapprove-bootstrap** — pehle node cert ko auto-approve karna:
```bash
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: node-autoapprove-bootstrap
subjects:
  - kind: Group
    name: system:bootstrappers
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: system:certificates.k8s.io:certificatesigningrequests:nodeclient
  apiGroup: rbac.authorization.k8s.io
EOF
```
- **3. node-autoapprove-certificate-rotation** — cert expire hone ke baad renewal auto-approve karna:
```bash
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: node-autoapprove-certificate-rotation
subjects:
  - kind: Group
    name: system:nodes
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: system:certificates.k8s.io:certificatesigningrequests:selfnodeclient
  apiGroup: rbac.authorization.k8s.io
EOF
```
- **Samjhao — teesri binding kyun zaroori hai:** Certificates expire hote hain. Agar rotation auto-approve na ho to node cert expire hone ke baad authenticate nahi kar paye ga. Pehle 2 bindings initial bootstrap ke liye hain, teesri us ke baad ke liye hai. Bootstrap ke baad kubelet ki identity `system:node:<name>` ban jati hai, group `system:nodes`.

### 9.5 Worker node tayyar karo — sirf CA copy karo
- Worker ko sirf cluster CA chahiye taake woh apiserver ki pehchan verify kar sake. **Yehi woh akela PKI material hai jo haath se copy karna hai.** Baaki sab (kubelet ka client cert aur key) bootstrap se automatically mile ga.
```bash
sudo mkdir -p /etc/kubernetes/pki
sudo scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null control-plane:/etc/kubernetes/pki/ca.crt /etc/kubernetes/pki/ca.crt
```
- kubelet API ke liye TLS lagao — Lesson 04 mein anonymous tha, ab x509 hai. File:
  - Path: `/var/lib/kubelet/config.d/70-authnz.conf`
```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration

authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: false
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt

authorization:
  mode: AlwaysAllow
```
- **Note:** Authentication ab x509 hai, lekin authorization abhi bhi AlwaysAllow hai is stage par. Ye bhi aage secure karna hai — yehi woh jagah hai jahan Rao ka security attack demo aaye ga (Lesson 04 wala anonymous :10250 vs ab wala x509).
- Bootstrap kubeconfig banao — `/etc/kubernetes/bootstrap-kubelet.conf`:
```bash
sudo kubectl config set-cluster default \
    --kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf \
    --certificate-authority=/etc/kubernetes/pki/ca.crt \
    --embed-certs=true \
    --server=https://control-plane:6443
sudo kubectl config set-credentials default \
    --kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf \
    --token=abcdef.0123456789abcdef
sudo kubectl config set-context default \
    --kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf \
    --cluster=default \
    --user=default
sudo kubectl config use-context default \
    --kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf
```
- kubelet service update karo — 2 flags:
```
--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf
--kubeconfig=/etc/kubernetes/kubelet.conf
```
```bash
sudo systemctl daemon-reload
sudo systemctl restart kubelet
```

### 9.6 Join flow — 6 steps, yehi video ka climax hai
1. kubelet bootstrap token se apiserver se connect karta hai.
2. Apni private key banata hai, aur **CertificateSigningRequest (CSR)** bhejta hai.
3. CSR controller (controller-manager ke andar) RBAC ke hisab se CSR ko **auto-approve** karta hai.
4. controller-manager cluster CA se certificate **sign** karta hai.
5. kubelet signed certificate leta hai, aur us ke sath naya kubeconfig save karta hai.
6. kubelet naye kubeconfig + certificate se aage ki saari baat karta hai.
- Check karo:
```bash
kubectl wait --for=create node worker
kubectl get nodes
kubectl get csr
```
- **Expected:** CSR status `Approved,Issued` dikhe ga. Node dikhe ga, lekin **NotReady** hoga. Ye theek hai.
- Kyun NotReady? Dekho:
```bash
kubectl describe node worker
```
```
container runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:Network plugin returns error: cni plugin not initialized
```
- **Samjhao:** kubelet ke paas network config nahi hai. Is ka workaround hai loopback CNI config — sirf Ready karne ke liye, asli networking Lesson 10 mein aaye gi:
  - Path: `/etc/cni/net.d/99-loopback.conf`
```json
{
  "cniVersion": "0.3.1",
  "name": "lo",
  "type": "loopback"
}
```
```bash
kubectl wait --for=condition=Ready node worker
kubectl get nodes
```

### 9.7 Pehla workload — cluster kaam kar raha hai
- ReplicaSet banao, 1 replica, hostNetwork (kyunki asli Pod networking abhi nahi hai):
```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: podinfo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: podinfo
  template:
    metadata:
      labels:
        app: podinfo
    spec:
      hostNetwork: true
      containers:
        - name: podinfo
          image: ghcr.io/stefanprodan/podinfo:latest
          ports:
            - containerPort: 9898
EOF
```
```bash
kubectl get replicaset podinfo
kubectl get pods -l app=podinfo -w
kubectl get pods -l app=podinfo -o wide
```
- **Bolne wali line:** "Worker node cluster ka hissa ban gaya, aur us par workload chal raha hai. Ab tak hum ne cluster banaya hai, lekin Pod networking abhi baaki hai — woh agla lesson hai."

### 9.8 Summary — joining
- TLS bootstrapping: temporary token → CSR → auto-approve → sign → cert. Manual cert distribution ki zaroorat khatam.
- Mutual authentication: apiserver kubelet ko cert se pehchanta hai, kubelet apiserver ko CA se pehchanta hai.
- RBAC: 3 bindings — CSR banana, pehla cert auto-approve, rotation auto-approve.
- Node registration: kubelet bootstrap kubeconfig se Node object banata hai, scheduling ke liye available hota hai.

---

## LESSONS 10–13 — Outline (detail Script 03 mein, verified one-page summary se)

### Lesson 10 — Network (Flannel)
- 2 workers: worker-1 `10.244.0.0/24`, worker-2 `10.244.1.0/24`. Bridge CNI se same-node Pod baat kar sakte hain, cross-node nahi.
- Pehle manual routes haath se add karke dikhao — routing hi missing piece hai — phir routes hata kar Flannel install karo.
- Flannel v0.27.2, flannel CNI v1.7.1-flannel2, VXLAN backend, interface `flannel.1`, net-conf.json Network `10.244.0.0/16`.
- Check: worker-1 se client Pod se worker-2 wale podinfo ko curl — pehle timeout, Flannel ke baad version jawab deta hai.
- **Rao ka addition yahan:** CNI deep — veth pair, bridge, VXLAN packet kahan se kahan jata hai, aur Flannel ke baghair kya toot-ta hai.

### Lesson 11 — kube-proxy
- Service ke 3 podinfo Pods ko ClusterIP milta hai, EndpointSlice mein Pods hain, lekin ClusterIP par request timeout hoti hai — kyunki woh address kisi machine par nahi hai, sirf ek rule hai.
- kube-proxy v1.34.0, iptables mode, config `/etc/kubernetes/kube-proxy/config.yaml`, clusterCIDR `10.244.0.0/16`.
- kube-proxy har worker par `KUBE-SERVICES` chain mein rules likhta hai, destination ko random Pod par rewrite karta hai.
- Check: `sudo iptables -t nat -L KUBE-SERVICES -n | grep podinfo`, phir ClusterIP curl — jawab aaye ga.

### Lesson 12 — CoreDNS
- kubelet ki setting `clusterDNS: [172.16.0.2]`, `clusterDomain: cluster.local`. Pehle kuch jawab nahi deta, `curl http://podinfo` fail hota hai.
- CoreDNS 1.12.2 systemd service ki tarah control-plane par, `172.16.0.2:53` par sunta hai. kubeadm use Service `10.96.0.10` ke peeche rakhta — yahan Pods seedha host address use karte hain.
- CoreDNS API server se Services ko watch karta hai, `cluster.local` ka jawab deta hai.
- Check: `kubectl exec client -- cat /etc/resolv.conf` (nameserver 172.16.0.2, search default.svc.cluster.local), phir `curl http://podinfo:80`.

### Lesson 13 — Conclusion
- Finished cluster: 1 control-plane, 2 workers, Flannel, kube-proxy, CoreDNS — sab systemd services.
- Course kya chhorta hai: HA control plane, backups, tighter security, monitoring. Agle steps: cluster dobara memory se banao, kubeadm se compare karo, Flannel / kube-proxy ko Cilium se replace karo.
- Aakhri exercise: ek ek component band karo, log parhne se pehle symptom predict karo.
- **Rao ka Phase 2 yahin se shuru hota hai:** HA, multiple etcd, kubeadm/k3s/cloud — Conclusion ko Phase 2 ka trailer banao, Phase 1 ka end nahi.

---

## SCRIPT 02 ka final check
- [ ] Lesson 05: etcd v3.6.4, plain HTTP pehle, phir TLS, CA alag, client cert, quorum samjhaya?
- [ ] Lesson 06: apiserver :6443, token pehle, phir Node,RBAC, curl se API, kubectl, TLS SANs, admin O=system:masters, /registry + auger — sab perform hua?
- [ ] Lesson 07: Binding demo pehle, phir scheduler install, nodeName check (nodeName ke baghair Pod wala test — Script 02 mein flagged hai, perform karke confirm karo)?
- [ ] Lesson 08: ReplicaSet pehle kuch nahi banata, phir controller-manager ke baad 3 Pods + self-healing + Deployment cascading — sab perform hua?
- [ ] Lesson 09: bootstrap token + 3 RBAC bindings + TLS bootstrapping 6 steps + CSR Approved,Issued + NotReady → loopback → Ready + pehla workload — sab perform hua?
- [ ] Koi iximiuz playground / promotion video mein nahi aaya? Unit file URLs sirf script ke notes mein hain?
- [ ] Har lesson ke end par dependency bola: kaun kis par dependent hai, kiske baghair kya toot-ta hai?

## Sources for Script 02 (verify karne ke liye)
- `workspace/kubernetes-teaching/source-notion/etcd.json` (23 code blocks, verified)
- `workspace/kubernetes-teaching/source-notion/apiserver.json` + `apiserver.extracted.md` (52 code blocks)
- `workspace/kubernetes-teaching/source-notion/scheduler.json` + `scheduler.extracted.md` (17 code blocks)
- `workspace/kubernetes-teaching/source-notion/controller-manager.json` + `controller-manager.extracted.md` (14 code blocks)
- `workspace/kubernetes-teaching/source-notion/joining.json` + `joining.extracted.md` (23 code blocks)
- `workspace/kubernetes-teaching/source-notion/module1.json` + `module1.extracted.md` (Versions, Prerequisites, systemd pattern)
- iximiuz one-page summary for Lessons 10–13 (Flannel, kube-proxy, CoreDNS, Conclusion)

---

## ADDENDUM (2026-11-17) — Rao ki ijazat se jo missing tha, add kiya gaya

### A. Certificate inventory — kaunsa cert kahan hai, kis ka CN kya hai
| Cert | Location | CN / O | Kaun use karta hai |
|---|---|---|---|
| etcd CA | /etc/etcd/pki/ca.crt | CN=etcd | etcd server + client certs sign karta hai |
| etcd server | /etc/etcd/pki/server.crt | CN=server | etcd |
| etcd client | /etc/etcd/pki/client.crt | CN=client | etcdctl + apiserver |
| K8s CA | /etc/kubernetes/pki/ca.crt | CN=kubernetes | saare K8s component certs sign karta hai |
| apiserver | /etc/kubernetes/pki/apiserver.crt | CN=kube-apiserver | apiserver TLS |
| admin | /etc/kubernetes/pki/admin.crt | CN=admin, O=system:masters | kubectl admin (RBAC group system:masters) |
| scheduler | /etc/kubernetes/pki/scheduler.crt | CN=system:kube-scheduler | kube-scheduler |
| controller-manager | /etc/kubernetes/pki/controller-manager.crt | CN=system:kube-controller-manager | kube-controller-manager |
| apiserver-kubelet-client | /etc/kubernetes/pki/apiserver-kubelet-client.crt | CN=kube-apiserver-kubelet-client, O=system:masters | apiserver → kubelet |
| kubelet (bootstrap se) | /var/lib/kubelet/pki/ | CN=system:node:<name> | kubelet → apiserver |
- Sab certs 365 din ke hain (CA 3650 din). Expiry check: `openssl x509 -in <file> -noout -dates -subject`
- **Do CA yaad rakho:** etcd ka apna CA, Kubernetes ka apna CA — alag alag. Milana mat.

### B. Dependency map — control plane
| Component | Dependent hai | Baghair is ke kya toot-ta hai |
|---|---|---|
| etcd | kuch nahi (single member) | apiserver kuch store nahi kar sakta — cluster ka dimagh band |
| apiserver | etcd | koi component kuch nahi kar sakta — scheduler, controller, kubelet sab isi se baat karte hain |
| scheduler | apiserver | Pod unscheduled rehta hai — banta hai, chalta nahi |
| controller-manager | apiserver | ReplicaSet/Deployment se Pod nahi banta, CSR sign nahi hota, Node ko Pod CIDR nahi milta |
| kubelet (worker) | apiserver + containerd | Node register nahi hota, Pod nahi chalta |

### C. Troubleshooting — control plane
```bash
sudo systemctl status etcd kube-apiserver kube-scheduler kube-controller-manager
sudo journalctl -u kube-apiserver -n 100 --no-pager
sudo journalctl -u etcd -n 100 --no-pager
bash --login -c "etcdctl endpoint health"
kubectl cluster-info
kubectl get nodes
kubectl get csr
```
- Apiserver nahi uth raha? Pehle etcd check karo — apiserver ka sab se pehla dependency etcd hai.
- CSR Pending hai? RBAC bindings check karo (Lesson 09 ke 3 ClusterRoleBindings).
- Node NotReady hai? `kubectl describe node` — NetworkReady=false matlab CNI config missing hai.
