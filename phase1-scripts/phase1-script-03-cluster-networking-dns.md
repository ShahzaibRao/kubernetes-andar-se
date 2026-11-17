# Phase 1 — Script 03: Module 4 Cluster (Lessons 10–13) — Network, kube-proxy, CoreDNS, Conclusion
## Kubernetes Andar Se — Chapter 0 / Bootstrap (Very Hard Way)

**Status:** Draft v1 — Rao performs on his own Proxmox VM, then recording
**Source:** iximiuz lesson pages (Network + CoreDNS + Conclusion full text fetched 2026-11-17; kube-proxy intro + one-page summary) + Notion (Lessons 01–09). Syllabus/reference only — video mein iximiuz ki promotion NAHI, playground NAHI.
**Lab:** Rao's own Proxmox VM — ab 3 machines chahiye: 1 control-plane + 2 workers (worker-1, worker-2). Viewer WSL / kisi bhi OS par Linux VMs par repeat kare.
**Pattern har lesson mein:** Pehle FAIL dikhao → phir fix karo → check karo.
**Script 02 se link:** Lesson 09 mein worker cluster mein join ho gaya, Ready hai (loopback CNI se). Ab asli networking aati hai.

## Version table (lock)
Kubernetes v1.34.0, containerd 2.2.1, runc v1.4.0, nerdctl 2.2.1, CNI plugins v1.9.0, etcd v3.6.4, Flannel v0.27.2 + flannel CNI v1.7.1-flannel2, CoreDNS 1.12.2, crictl v1.34.0, kubeletctl v1.13, auger 1.0.3.

## Pehle — doosra worker (worker-2) join karo
- Lesson 10 se cluster mein 2 workers hain: worker-1 aur worker-2. Lesson 09 wala worker ab **worker-1** hai.
- worker-2 ko Lesson 09 ke steps se join karo (bootstrap token + CA copy + kubelet bootstrap kubeconfig + restart). Dono nodes Ready hone chahiye:
```bash
kubectl get nodes
```
- **Expected:** worker-1 aur worker-2 dono `Ready`.
- Pod CIDR kahan se aaya, check karo (control-plane machine par):
```bash
kubectl get node worker-1 -o jsonpath='{.spec.podCIDR}{"\n"}'
kubectl get node worker-2 -o jsonpath='{.spec.podCIDR}{"\n"}'
```
- **Expected:** worker-1 `10.244.0.0/24`, worker-2 `10.244.1.0/24`.
- **Samjhao — ye CIDR kis ne diya:** Lesson 08 mein controller-manager ko `--cluster-cidr=10.244.0.0/16` aur `--allocate-node-cidrs=true` diye the. Us ne 10.244.0.0/16 ko kaat kar har Node ko ek /24 diya (default /24, `--node-cidr-mask-size` se badal sakte ho). Ye metadata khud networking nahi banata — sirf assignment hai. Networking abhi bhi missing hai.

---

## LESSON 10 — Network (Flannel)

### Overview
- **Network addon** woh missing piece hai jo cluster networking ko nodes ke across kaam karwata hai. Node-level CNI setup akela kaafi nahi hai — cluster-level coordination chahiye: subnet allocation, routing, tunnel setup.
- Kuch addon routes distribute karte hain (BGP — manual `ip route add` ko automate karna). Doosre overlay use karte hain (VXLAN) — jo tab bhi kaam karta hai jab underlying network Pod subnets ke beech direct routing allow na kare. Zyada portable hai.
- **Overlay network kya hai:** Physical network ke upar ek virtual network. worker-1 ka Pod jab worker-2 ke Pod ko packet bhejta hai, overlay us packet ko wrap (encapsulate) karke normal node-to-node network se bhejta hai. Receiving node unwrap karke destination Pod ko de deta hai. Pods aise baat karte hain jaise ek hi network par hon, halanke alag machines par hain.
- **VXLAN** common encapsulation protocol hai, Flannel jaisay addon use karte hain.

### Objectives
- CNI, network addon, aur Pod-to-Pod communication samajhna.
- Pehle manual bridge + manual routes se dikhana ke routing hi missing piece hai.
- Flannel install/configure karna (daemon + CNI plugin).
- Cross-node connectivity verify karna.

### 10.1 Pehle manual bridge — same node chalta hai, cross-node nahi
- Har worker par manual bridge CNI config banao. worker-1 par subnet `10.244.0.0/24`, worker-2 par `10.244.1.0/24`:
  - Path: `/etc/cni/net.d/10-bridge.conf`
```json
{
  "cniVersion": "0.3.1",
  "name": "bridge",
  "type": "bridge",
  "bridge": "cni0",
  "isGateway": true,
  "ipMasq": true,
  "ipam": {
    "type": "host-local",
    "ranges": [[{"subnet": "10.244.0.0/24"}]],
    "routes": [{"dst": "0.0.0.0/0"}]
  }
}
```
- **Note:** worker-2 par subnet `10.244.1.0/24` likhna hai. Perform karte waqt lesson page se verify karo.
- Same-node test karo:
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: podinfo-worker-1
spec:
  nodeName: worker-1
  containers:
    - name: podinfo
      image: ghcr.io/stefanprodan/podinfo:latest
      ports:
        - containerPort: 9898
---
apiVersion: v1
kind: Pod
metadata:
  name: client-worker-1
spec:
  nodeName: worker-1
  containers:
    - name: curl
      image: ghcr.io/stefanprodan/podinfo:latest
      command: ["sh", "-c", "sleep infinity"]
EOF
```
```bash
kubectl wait --for=condition=Ready pod podinfo-worker-1 client-worker-1
PODINFO_IP=$(kubectl get pod podinfo-worker-1 -o jsonpath='{.status.podIP}')
kubectl exec client-worker-1 -- curl -fsS "http://${PODINFO_IP}:9898/version"
```
- **Same node par chalega.** Ab cross-node test karo — pehle FAIL dikhao:
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: podinfo-worker-2
spec:
  nodeName: worker-2
  containers:
    - name: podinfo
      image: ghcr.io/stefanprodan/podinfo:latest
      ports:
        - containerPort: 9898
EOF
```
```bash
kubectl wait --for=condition=Ready pod podinfo-worker-2
PODINFO_IP=$(kubectl get pod podinfo-worker-2 -o jsonpath='{.status.podIP}')
kubectl exec client-worker-1 -- curl -fsS --max-time 5 "http://${PODINFO_IP}:9898/version"
```
- **Timeout hoga.** Kyunki worker-1 ko nahi pata ke `10.244.1.0/24` kahan hai, aur worker-2 ko nahi pata ke `10.244.0.0/24` kahan hai.
- **Samjhao:** Dono machines ke paas apna subnet hai, lekin doosre ka rasta nahi hai. Routing hi missing piece hai.

### 10.2 Manual routes — routing hi missing piece hai, proof
- Dono workers par haath se routes add karo (doosre worker ka IP apne lab se lo, `ip addr` se):
```bash
# worker-1 par:
sudo ip route add 10.244.1.0/24 via <worker-2-IP>
# worker-2 par:
sudo ip route add 10.244.0.0/24 via <worker-1-IP>
```
- Wahi cross-node curl dobara chalao — **ab chalega.**
- **Bolne wali line:** "Sirf 2 routes ne cross-node networking fix kar di. Koi jaadu nahi hua — routing hi missing thi. Ab routes hata do, kyunki ye manual kaam Flannel automate kare ga."
```bash
# worker-1 par:
sudo ip route del 10.244.1.0/24 via <worker-2-IP>
# worker-2 par:
sudo ip route del 10.244.0.0/24 via <worker-1-IP>
```

### 10.3 Flannel kya hai — 2 components
- Flannel sab se purana aur simple network addon hai. Production mein aaj kal kam milta hai, lekin seekhne ke liye perfect hai.
- 2 components hain:
  - **Daemon (flanneld)** — har node par chalta hai, cluster-level coordination karta hai.
  - **CNI plugin** — Pod networking configure karta hai.
- Flannel kaise kaam karta hai:
  1. Per-node Pod CIDR use karta hai (cluster-wide `10.244.0.0/16` se).
  2. flanneld systemd service ke tor par har worker par chalta hai, Node subnet assignments ko Kubernetes API se watch karta hai.
  3. Flannel CNI config install karta hai taake containerd Pod networking ko `flannel` plugin ko delegate kare.
  4. VXLAN overlay set karta hai taake alag nodes ke subnets ke beech traffic tunnel ho.
- **Samjhao — split yaad rakho:** Node-level setup (containerd CNI plugin se Pod ko local network mein wire karta hai) vs Cluster-level coordination (Flannel node-local Pod networks ko cluster bhar mein jorta hai). Yehi split poore course mein hai.

### 10.4 Flannel ke prerequisites — cert + kubeconfig + RBAC (control-plane par)
- Flannel ko API server se baat karni hai, is liye us ki apni identity hai: `system:flannel`.
- Certificate banao — `CN=system:flannel`:
```bash
(
cd /etc/kubernetes/pki
sudo openssl genrsa -out flannel.key 2048
sudo openssl req -new -key flannel.key -out flannel.csr -subj "/CN=system:flannel"
sudo openssl x509 -req -in flannel.csr -out flannel.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365
)
```
- kubeconfig banao — `/etc/kubernetes/flannel.conf`:
```bash
sudo kubectl config set-cluster default \
    --kubeconfig=/etc/kubernetes/flannel.conf \
    --certificate-authority=/etc/kubernetes/pki/ca.crt \
    --embed-certs=true \
    --server=https://control-plane:6443
sudo kubectl config set-credentials default \
    --kubeconfig=/etc/kubernetes/flannel.conf \
    --client-certificate=/etc/kubernetes/pki/flannel.crt \
    --client-key=/etc/kubernetes/pki/flannel.key \
    --embed-certs=true
sudo kubectl config set-context default \
    --kubeconfig=/etc/kubernetes/flannel.conf \
    --cluster=default \
    --user=default
sudo kubectl config use-context default \
    --kubeconfig=/etc/kubernetes/flannel.conf
```
- **Note:** Server `https://control-plane:6443` hai — control-plane machine ka hostname. Apne lab mein hostname ke hisab se likho.
- RBAC banao — Flannel ko sirf woh permissions do jo us ko chahiye (pods get, nodes get/list/watch, nodes/status patch):
```bash
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:flannel
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:flannel
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:flannel
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: system:flannel
EOF
```
- **Samjhao — least privilege:** Flannel ko poora cluster-admin nahi diya. Sirf nodes parhne aur node status patch karne ki permission hai. Yehi RBAC ka asal matlab hai — har component ko sirf woh jo us ko chahiye.

### 10.5 Flannel install karna — worker-1 par
```bash
FLANNEL_VERSION=v0.27.2
FLANNEL_PLUGIN_VERSION=v1.7.1-flannel2
```
```bash
curl -fsSLO "https://github.com/flannel-io/flannel/releases/download/${FLANNEL_VERSION?}/flannel-${FLANNEL_VERSION?}-linux-amd64.tar.gz"
curl -fsSLO "https://github.com/flannel-io/cni-plugin/releases/download/${FLANNEL_PLUGIN_VERSION?}/cni-plugin-flannel-linux-amd64-${FLANNEL_PLUGIN_VERSION?}.tgz"
tar xzvof "flannel-${FLANNEL_VERSION?}-linux-amd64.tar.gz"
tar xzvof "cni-plugin-flannel-linux-amd64-${FLANNEL_PLUGIN_VERSION?}.tgz"
sudo install -m 755 flanneld /usr/local/bin
sudo install -m 755 flannel-amd64 /opt/cni/bin/flannel
```
- systemd unit file:
```bash
sudo wget -O /etc/systemd/system/flanneld.service https://labs.iximiuz.com/content/files/courses/kubernetes-the-very-hard-way-0cbfd997/04-cluster/02-network/__static__/flanneld.service?v=1777378798
```
- **Rao note:** Unit file ki apni copy rakho, video mein URL mat dikhao.
- **Unit file ke flags samjhao:**
  - `--kube-subnet-mgr` — Pod subnet assignments ko Kubernetes API se parho (Node `.spec.podCIDR`), local etcd se nahi.
  - `--kube-api-url` — API server endpoint.
  - `--kubeconfig-file` — Flannel ke client cert wala kubeconfig.
  - `--ip-masq` — Cluster Pod network se bahar jane wale traffic ke liye IP masquerading (SNAT).
  - `--net-config-path` — Network config file (`net-conf.json`).
  - `NODE_NAME=%H` — Flannel ko batata hai ke woh kis node par hai. Pod mein ye downward API se milta, systemd service mein `%H` (hostname) se milta hai.
- Network config banao — `/etc/flannel/net-conf.json`:
```bash
sudo mkdir -p /etc/flannel
sudoedit /etc/flannel/net-conf.json
```
```json
{
    "Network": "10.244.0.0/16",
    "EnableNFTables": false,
    "Backend": {
        "Type": "vxlan"
    }
}
```
- **Note:** `EnableNFTables: false` is liye hai ke environment iptables use karta hai. Flannel newer systems par nftables default karta hai, jo conflict kare ga.
- Flannel CNI plugin config banao — `/etc/cni/net.d/10-flannel.conflist`:
```json
{
  "name": "cbr0",
  "cniVersion": "1.0.0",
  "plugins": [
    {
      "type": "flannel",
      "delegate": {
        "hairpinMode": true,
        "isDefaultGateway": true
      }
    },
    {
      "type": "portmap",
      "capabilities": {
        "portMappings": true
      }
    }
  ]
}
```
- **Samjhao — conflist kya hai:** Ye ek chain hai, single plugin config nahi. `flannel` plugin bridge aur IPAM internally handle karta hai, phir `portmap` us ke baad chalta hai.
  - `name: cbr0` — logical network name, CNI IPAM state track karne ke liye.
  - `delegate.hairpinMode` — Pod apne hi Service IP se khud ko reach kar sakta hai.
  - `delegate.isDefaultGateway` — Bridge Pods ka default gateway banta hai (manual bridge config mein `isGateway` wala role).
  - `type: portmap` — `hostPort` mappings enable karta hai.
- **Samjhao — Flannel CNI plugin andar se bridge hi hai:** Wohi bridge plugin jo tum ne manual configure kiya tha, us ko delegate karta hai. Farq ye hai ke Flannel subnet automatically deta hai, Node ke Pod CIDR assignment se. Aur ab alag `99-loopback.conf` ki zaroorat nahi — conflist use karte waqt containerd Pod ke network namespace mein loopback interface automatically configure karta hai.
- kubeconfig worker par copy karo:
```bash
sudo scp control-plane:/etc/kubernetes/flannel.conf /etc/kubernetes/flannel.conf
```
- Service start karo:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now flanneld
```

### 10.6 Same-node check — Flannel ke baad bhi chalna chahiye
- Test Pods dobara banao (control-plane par):
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: podinfo-worker-1
spec:
  nodeName: worker-1
  containers:
    - name: podinfo
      image: ghcr.io/stefanprodan/podinfo:latest
      ports:
        - containerPort: 9898
---
apiVersion: v1
kind: Pod
metadata:
  name: client-worker-1
spec:
  nodeName: worker-1
  containers:
    - name: curl
      image: ghcr.io/stefanprodan/podinfo:latest
      command: ["sh", "-c", "sleep infinity"]
EOF
```
```bash
kubectl wait --for=condition=Ready pod podinfo-worker-1 client-worker-1
kubectl get pods -o wide
```
```bash
PODINFO_IP=$(kubectl get pod podinfo-worker-1 -o jsonpath='{.status.podIP}')
kubectl exec client-worker-1 -- curl -fsS "http://${PODINFO_IP}:9898/version"
```
- Route table check karo (worker-1 par):
```bash
ip route show
```
- **Expected:** local Pod subnet `10.244.0.0/24 dev cni0` dikhe ga. Abhi Flannel interface nahi dikhe ga — sirf ek worker hai, tunnel karne ke liye doosra node nahi hai. Flannel ne `cni0` interface wahi bridge plugin se dobara banaya jo tum ne pehle manual use kiya tha.

### 10.7 Flannel worker-2 par — same steps, same files
- worker-2 par wahi 6 steps dohraye:
  1. flanneld aur CNI plugin download + install
  2. systemd unit file
  3. `/etc/flannel/net-conf.json`
  4. `/etc/cni/net.d/10-flannel.conflist`
  5. kubeconfig copy (control-plane se)
  6. daemon-reload + flanneld start
- **Samjhao — files identical hain:** Manual bridge setup mein har node ko alag subnet apni CNI config mein likhna parta tha. Flannel mein sab config files dono workers par same hain, kyunki Flannel Node ka assigned Pod CIDR Kubernetes se khud parhta hai.

### 10.8 Cross-node check — ab chalega
```bash
kubectl wait --for=condition=Ready pod podinfo-worker-2
kubectl get pods -o wide
```
```bash
PODINFO_IP=$(kubectl get pod podinfo-worker-2 -o jsonpath='{.status.podIP}')
kubectl exec client-worker-1 -- curl -fsS "http://${PODINFO_IP}:9898/version"
```
- **Ab chalega.** Wahi check jo pehle timeout hota tha, ab jawab deta hai.
- **Bolne wali line:** "client-worker-1 ne 10.244.1.x ko packet bheja, Flannel ne us ko encapsulate karke worker-2 tak pahunchaya, worker-2 ke Flannel ne unwrap karke podinfo ko de diya. Yehi VXLAN hai."

### 10.9 VXLAN in action — route table parho
- worker-1 par:
```bash
ip route show
```
- **Expected:** local `10.244.0.0/24` cni0 par, aur `10.244.1.0/24` flannel.1 par.
- Poori tasveer:
  1. `10.244.0.0/24` ka traffic local rehta hai, cni0 par.
  2. `10.244.1.0/24` ka traffic flannel.1 se jata hai.
  3. Flannel us ko encapsulate karke node network se worker-2 tak le jata hai.
- worker-2 par bhi check karo — wahan `10.244.0.0/24` flannel.1 par dikhe ga.

### 10.10 Summary — Network
- Pod-to-Pod reachability 3 cheezon se banti hai: CNI plugin (local wiring), Pod CIDR assignment (controller-manager), aur cluster coordination (Flannel + VXLAN).
- Manual routes se proof hua ke routing hi missing piece thi. Flannel woh routing automate karta hai.
- **Dependency yaad rakho:** Flannel ke baghair cross-node Pod baat nahi kar sakte. Flannel apiserver par dependent hai (Node Pod CIDR parhne ke liye). Flannel band ho to cross-node traffic band.
- **Security note (Rao ka thread):** VXLAN traffic plain hai — encrypt nahi hota. Production mein WireGuard backend ya NetworkPolicy chahiye hoti hai. Yehi woh jagah hai jahan Cilium/Calico ka zikr aaye ga Phase 2 mein.

---

## LESSON 11 — kube-proxy

### Overview
- Flannel ke baad Pods ek doosre ko IP se reach kar sakte hain. Lekin **Pod IPs ephemeral hain** — ye ek feature hai, lekin client ke liye sar dard hai.
- Har baar jab naya version deploy ho, scale ho, ya Pod reschedule ho, naye Pods naye IP ke sath aate hain. Frontend agar backend ko `10.244.x.x` se connect kare, to woh address Pod replace hote hi invalid ho jata hai.
- 3 replicas hon to kaunsa IP use kare? Load balance kaise ho? Ek Pod mar jaye aur naya alag IP se aaye to kya ho?
- Hardcoding Pod IPs hal nahi hai. **Stable endpoint** chahiye jo Pods ke peeche abstract kare.

### Objectives
- Service kya hai, ClusterIP kya hai, EndpointSlice kya hai — samajhna.
- kube-proxy install/configure karna (iptables mode).
- ClusterIP ko kaam karte hue dekhna (pehle timeout, phir jawab).

### 11.1 Service kya hai — Pod IP vs ClusterIP
| Concept | Address | Lifetime | Example |
|---|---|---|---|
| **Pod IP** | Pod CIDR se | Pod se bandha (ephemeral) | `10.244.0.2` |
| **ClusterIP** | Service CIDR se | Service se bandha (stable) | `10.96.0.10` |
- **Service** ek Kubernetes resource hai jo Pods ke set ko stable network identity deta hai. Service banate hi Kubernetes us ko **ClusterIP** deta hai — virtual IP, dedicated range (service CIDR, `10.96.0.0/12`) se, jo Service ki zindagi bhar kabhi nahi badalta.
- Service **label selector** se target Pods dhoondta hai. Control plane continuously track karta hai ke kaunse Pods selector se match karte hain, aur un ki current IPs **EndpointSlices** mein maintain karta hai.
- Matlab:
  - Clients **ClusterIP** se connect karte hain (kabhi nahi badalta).
  - ClusterIP ek **Pod IP** par route hota hai (jo kabhi bhi badal sakta hai).
  - Pods scale up/down hon to EndpointSlices automatically update hote hain.
- **Samjhao — ClusterIP virtual hai:** Pod IP ki tarah kisi network interface se bandha nahi hai. Us ke peeche koi Pod ya container nahi hai. Ye sirf ek frontend hai jo ek real Pod IP par rewrite hota hai. Kisi ko packets intercept karke destination rewrite karna hai — woh "kisi" **kube-proxy** hai.

### 11.2 Pehle FAIL dikhao — Service banao, ClusterIP timeout
- 3 replicas wala Deployment + Service banao (control-plane par):
```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
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
      containers:
        - name: podinfo
          image: ghcr.io/stefanprodan/podinfo:latest
          ports:
            - containerPort: 9898
---
apiVersion: v1
kind: Service
metadata:
  name: podinfo
spec:
  selector:
    app: podinfo
  ports:
    - port: 80
      targetPort: 9898
EOF
```
```bash
kubectl get svc podinfo
kubectl get endpointslices -l kubernetes.io/service-name=podinfo
kubectl get pods -l app=podinfo -o wide
```
- **Samjhao:** Service ko ClusterIP mil gaya, EndpointSlice mein 3 Pods hain. Lekin ClusterIP par request timeout hogi — kyunki woh address kisi machine par nahi hai. Sirf ek rule hai, jo abhi likha hi nahi gaya.
```bash
CLUSTER_IP=$(kubectl get svc podinfo -o jsonpath='{.spec.clusterIP}')
kubectl exec deploy/podinfo -- curl -fsS --max-time 5 "http://${CLUSTER_IP}:80"
```
- **Timeout hoga.** Yehi woh fail hai jo kube-proxy fix kare ga.

### 11.3 kube-proxy install karna — har worker par
- kube-proxy **har worker** par chalta hai. Woh Services aur EndpointSlices ko watch karta hai, aur iptables rules likhta hai.
- Install karo (har worker par):
```bash
KUBE_VERSION=v1.34.0
```
```bash
curl -fsSLO https://dl.k8s.io/${KUBE_VERSION?}/bin/linux/amd64/kube-proxy
sudo install -m 755 kube-proxy /usr/local/bin
```
```bash
sudo wget -O /etc/systemd/system/kube-proxy.service https://labs.iximiuz.com/content/files/courses/kubernetes-the-very-hard-way-0cbfd997/04-cluster/03-kube-proxy/__static__/kube-proxy.service?v=1777378802
```
- **Rao note:** Unit file ki apni copy rakho, video mein URL mat dikhao.
- Certificate + kubeconfig — scheduler/controller-manager wala hi pattern hai (perform karte waqt lesson page se verify karo):
  - Cert: `CN=system:kube-proxy`, Kubernetes CA se sign.
  - Kubeconfig: `/etc/kubernetes/kube-proxy.conf`, server `https://127.0.0.1:6443` (control-plane par) ya `https://control-plane:6443` (worker par), `--embed-certs=true`.
- Config file banao — `/etc/kubernetes/kube-proxy/config.yaml`:
```yaml
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: iptables
clusterCIDR: 10.244.0.0/16
```
- **Samjhao — 2 fields yaad rakho:** `mode: iptables` (rules kaise likhe jayen) aur `clusterCIDR: 10.244.0.0/16` (Pod network ka range, taake kube-proxy jaan sake ke Pod IPs kahan hain). Baaki fields perform karte waqt lesson page se verify karo — andaza se mat likhna.
- Service start karo (har worker par):
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kube-proxy
```

### 11.4 Rules dekho — KUBE-SERVICES chain
```bash
sudo iptables -t nat -L KUBE-SERVICES -n | grep podinfo
```
- **Samjhao — iptables mode kaise kaam karta hai:** Har Service ko `KUBE-SERVICES` chain mein entry milti hai jo destination ko us ke Pods mein se ek par rewrite karti hai, random pick karke. Ye DNAT hai — packet ka destination address badal jata hai, Pod ko pata bhi nahi chalta ke request ClusterIP par aayi thi.
- **Bolne wali line:** "ClusterIP kisi machine par nahi hai. Woh sirf ek iptables rule hai. kube-proxy woh rule likhta hai, aur har packet us rule se guzarta hai."

### 11.5 Check karo — ClusterIP ab jawab deta hai
```bash
CLUSTER_IP=$(kubectl get svc podinfo -o jsonpath='{.spec.clusterIP}')
kubectl exec deploy/podinfo -- curl -fsS "http://${CLUSTER_IP}:80"
```
- **Ab jawab aaye ga.** Kai baar chalao — har baar alag Pod jawab de sakta hai (random pick). Yehi load balancing hai, iptables mode mein.
- Pod delete karke bhi check karo — EndpointSlice update hoga, aur request ab bhi chalegi:
```bash
kubectl delete pod -l app=podinfo --field-selector=status.phase=Running --grace-period=0 --force 2>/dev/null || kubectl delete pod $(kubectl get pods -l app=podinfo -o jsonpath='{.items[0].metadata.name}')
kubectl get endpointslices -l kubernetes.io/service-name=podinfo
kubectl exec deploy/podinfo -- curl -fsS "http://${CLUSTER_IP}:80"
```

### 11.6 Summary — kube-proxy
- Service stable identity deta hai, ClusterIP virtual hai, EndpointSlice Pods ki current list rakhta hai.
- kube-proxy woh component hai jo ClusterIP ko real banata hai — iptables rules se.
- **Dependency yaad rakho:** kube-proxy ke baghair Service sirf ek naam hai, us par traffic nahi jata. kube-proxy apiserver par dependent hai (Services + EndpointSlices watch karne ke liye). kube-proxy band ho to naye rules nahi likhe jate — purane rules rehte hain, lekin naye Pods/Services kaam nahi karte.
- **Aage ka ishara:** iptables mode simple hai, lekin bade clusters mein IPVS ya eBPF (Cilium) use hota hai. Yehi woh jagah hai jahan Phase 2 mein Cilium ka zikr aaye ga.

---

## LESSON 12 — CoreDNS

### Overview
- Ab tak Pods ko IP se reach kar sakte ho, aur Service ko ClusterIP se. Lekin **naam** se nahi. `curl http://podinfo:80` abhi fail hoga — kyunki koi DNS nahi hai jo `podinfo` ko ClusterIP mein badle.
- **CoreDNS** flexible, extensible DNS server hai, Go mein likha, CNCF graduated project, Kubernetes ka default cluster DNS (Kubernetes 1.13 se, kube-dns ki jagah).
- CoreDNS Kubernetes API ko watch karta hai Services aur Endpoints ke liye, phir current cluster state ke hisab se DNS queries ka jawab deta hai. Jab Pod poochta hai "podinfo.default.svc.cluster.local ka IP kya hai?", CoreDNS matching Service dhoond kar us ka ClusterIP wapas deta hai.
- Zyada tar distributions mein CoreDNS cluster ke andar Deployment ke tor par chalta hai. Is course mein hum us ko bhi haath se install karte hain — **control-plane machine par systemd service** ke tor par, baaki har component ki tarah.

### Objectives
- CoreDNS ka role samajhna.
- Scratch se install/configure karna (Corefile).
- DNS-based service discovery ko action mein dekhna.
- Poora request path samajhna: Pod → DNS → ClusterIP → kube-proxy → Pod.

### 12.1 Pehle FAIL dikhao — naam se nahi chalega
- Client Pod se naam se try karo:
```bash
kubectl exec client -- curl -fsS --max-time 5 "http://podinfo:80"
```
- **Fail hoga.** IP se chalta hai (Lesson 11), naam se nahi. Kyunki Pod ke `/etc/resolv.conf` mein jo DNS hai, woh abhi kuch jawab nahi deta.
- **Samjhao:** IP yaad rakhna insaan ka kaam nahi hai. Naam se kaam karna DNS ka kaam hai. Yehi CoreDNS ka kaam hai.

### 12.2 Prerequisites — cert + kubeconfig + RBAC (control-plane par)
- CoreDNS ko bhi API server se authenticate karna hai. Us ki apni identity hai: `system:coredns`.
- Certificate banao — `CN=system:coredns`:
```bash
(
cd /etc/kubernetes/pki
sudo openssl genrsa -out coredns.key 2048
sudo openssl req -new -key coredns.key -out coredns.csr -subj "/CN=system:coredns"
sudo openssl x509 -req -in coredns.csr -out coredns.crt \
  -CA ca.crt -CAkey ca.key \
  -days 365
)
```
- kubeconfig banao — `/etc/kubernetes/coredns.conf`:
```bash
sudo kubectl config set-cluster default \
    --kubeconfig=/etc/kubernetes/coredns.conf \
    --certificate-authority=/etc/kubernetes/pki/ca.crt \
    --embed-certs=true \
    --server=https://127.0.0.1:6443
sudo kubectl config set-credentials default \
    --kubeconfig=/etc/kubernetes/coredns.conf \
    --client-certificate=/etc/kubernetes/pki/coredns.crt \
    --client-key=/etc/kubernetes/pki/coredns.key \
    --embed-certs=true
sudo kubectl config set-context default \
    --kubeconfig=/etc/kubernetes/coredns.conf \
    --cluster=default \
    --user=default
sudo kubectl config use-context default \
    --kubeconfig=/etc/kubernetes/coredns.conf
```
- RBAC banao — CoreDNS ko sirf parhne ki permission chahiye (list + watch):
```bash
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:coredns
rules:
  - apiGroups: [""]
    resources: ["endpoints", "services", "pods", "namespaces"]
    verbs: ["list", "watch"]
  - apiGroups: ["discovery.k8s.io"]
    resources: ["endpointslices"]
    verbs: ["list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:coredns
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: system:coredns
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:coredns
EOF
```
- **Samjhao — CN username banta hai:** `subjects` mein `kind: User` hai, `ServiceAccount` nahi. Jab component client certificate se authenticate karta hai, to cert ka Common Name (CN) username ban jata hai. Cert mein `CN=system:coredns` hai, is liye ClusterRoleBinding us user ko permissions deta hai. Yehi pattern Flannel, scheduler, controller-manager mein bhi tha.

### 12.3 CoreDNS install karna
```bash
COREDNS_VERSION=1.12.2
```
```bash
curl -fsSLO "https://github.com/coredns/coredns/releases/download/v${COREDNS_VERSION}/coredns_${COREDNS_VERSION}_linux_amd64.tgz"
tar xzf coredns_${COREDNS_VERSION}_linux_amd64.tgz
sudo install -m 755 coredns /usr/local/bin
```
- CoreDNS ke liye alag system user banao:
```bash
sudo adduser \
    --system \
    --group \
    --disabled-login \
    --disabled-password \
    --home /var/lib/coredns \
    coredns
```
- kubeconfig us user ko parhne do:
```bash
sudo chown coredns:coredns /etc/kubernetes/coredns.conf
```

### 12.4 Corefile — CoreDNS ki config
- CoreDNS **Corefile** se configure hota hai. Banao — `/etc/coredns/Corefile`:
```bash
sudo mkdir -p /etc/coredns
sudoedit /etc/coredns/Corefile
```
```
.:53 {
    bind eth0
    errors
    health {
        lameduck 5s
    }
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        kubeconfig /etc/kubernetes/coredns.conf
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
        ttl 30
    }
    forward . /etc/resolv.conf {
        max_concurrent 1000
    }
    cache 30
    loop
    reload
    loadbalance
}
```
- **Corefile breakdown — har line samjhao:**
  - `bind eth0` — sirf node ke network interface par suno (sab interfaces par nahi).
  - `errors` — errors ko stdout par log karo.
  - `health` — health check endpoint expose karo.
  - `ready` — readiness endpoint `:8181/ready` par.
  - `kubernetes cluster.local` — Kubernetes plugin enable karo: `svc.cluster.local` naam API ko watch karke resolve karo.
  - `kubeconfig /etc/kubernetes/coredns.conf` — API server se authenticate karo kubeconfig se (kyunki CoreDNS cluster ke bahar systemd service hai, Pod nahi — Pod hota to ServiceAccount token mount hota).
  - `forward . /etc/resolv.conf` — Non-cluster queries (jaise `google.com`) ko node ke upstream DNS ko forward karo.
  - `cache 30` — DNS responses 30 second cache karo.
  - `loop` — Forwarding loops detect karke roko.
  - `reload` — Corefile change ho to automatically reload karo.
  - `loadbalance` — A records ki order randomize karo (round-robin).
- systemd unit file:
```bash
sudo wget -O /etc/systemd/system/coredns.service https://labs.iximiuz.com/content/files/courses/kubernetes-the-very-hard-way-0cbfd997/04-cluster/04-coredns/__static__/coredns.service?v=1777378804
```
- **Rao note:** Unit file ki apni copy rakho, video mein URL mat dikhao.
- **Unit file mein kya hai:** CoreDNS `coredns` user ke tor par chalta hai, minimum capabilities ke sath. `CAP_NET_BIND_SERVICE` us ko port 53 par sunne deta hai bina root bane. Service kube-apiserver par depend karti hai, aur fail hone par automatically restart hoti hai.
- **Samjhao — least privilege, dobara:** Port 53 privileged hai (1024 se neeche), lekin CoreDNS root nahi hai — sirf ek capability hai. Yehi security ka asal pattern hai.
- Service start karo:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now coredns
```

### 12.5 kubelet ko DNS batana — clusterDNS + clusterDomain
- Pods ko pata hona chahiye ke DNS kahan hai. kubelet har Pod ke `/etc/resolv.conf` mein DNS server likhta hai.
- Kubelet config mein set karo (config.d mein nayi file, 99-cri.conf wale pattern se):
  - `clusterDNS: [172.16.0.2]` — CoreDNS ka address (control-plane machine ka IP).
  - `clusterDomain: cluster.local`
- **Note:** `172.16.0.2` iximiuz lab ka control-plane IP hai. **Rao apne lab mein apne control-plane VM ka asli IP likhe.** Perform karte waqt config file ka exact naam lesson page se verify karo.
- Kubelet restart karo (dono workers par):
```bash
sudo systemctl daemon-reload
sudo systemctl restart kubelet
```
- Check karo — client Pod ke andar:
```bash
kubectl exec client -- cat /etc/resolv.conf
```
- **Expected:** `nameserver 172.16.0.2` (ya apne control-plane ka IP), aur search domains `default.svc.cluster.local svc.cluster.local cluster.local`.

### 12.6 DNS verify karo — 4 naam, ek hi jawab
```bash
kubectl exec client -- curl -fsS --max-time 5 "http://podinfo:80"
```
- **Ab chalega.** Client Pod ne `podinfo` ko Service ke ClusterIP se resolve kiya CoreDNS ke zariye, aur kube-proxy ne traffic ko backend Pod tak pahunchaya.
- Baaki 3 forms bhi try karo:
```bash
# Namespace-qualified
kubectl exec client -- curl -fsS --max-time 5 "http://podinfo.default:80"
# With svc prefix
kubectl exec client -- curl -fsS --max-time 5 "http://podinfo.default.svc:80"
# Fully qualified domain name
kubectl exec client -- curl -fsS --max-time 5 "http://podinfo.default.svc.cluster.local:80"
```
- **Charon forms ek hi ClusterIP par resolve hote hain.** Chhote forms is liye chalte hain ke `/etc/resolv.conf` mein search domains hain jo kubelet ne likhe hain.

### 12.7 Poori tasveer — ek request ka safar
- Client Pod `curl http://podinfo:80` chalata hai:
  1. Client Pod ka resolver `/etc/resolv.conf` parhta hai aur pehla search domain lagata hai: `.default.svc.cluster.local`.
  2. DNS query `172.16.0.2` (control-plane machine ka IP, `clusterDNS` se) ko bhejta hai.
  3. Packet network se control-plane machine tak jata hai.
  4. **CoreDNS** (systemd service) query leta hai aur Kubernetes API se `podinfo` Service dhoondta hai.
  5. CoreDNS Service ka ClusterIP wapas deta hai (masalan `10.96.23.42`).
  6. Client Pod `10.96.23.42:80` se connect karta hai.
  7. **kube-proxy** packet ko intercept karta hai (iptables DNAT) aur us ko podinfo backend Pods mein se ek par bhejta hai.
  8. Request Pod tak pahunchti hai, aur jawab wahi rasta wapas aata hai.
- **Bolne wali line:** "Is module ke har networking component ne apna hissa dala: Flannel Pod network ke liye, kube-proxy Service routing ke liye, CoreDNS naam ke liye. Teenon ke baghair ek bhi request poori nahi hoti."

### 12.8 Summary — CoreDNS
- CoreDNS API ko watch karke DNS jawab deta hai. Service ka naam → ClusterIP.
- Corefile plugin-based hai — har line ka matlab hai.
- **Dependency yaad rakho:** CoreDNS ke baghair naam se kuch nahi chalta, sirf IP se. CoreDNS apiserver par dependent hai. CoreDNS band ho to DNS fail, lekin IP se traffic ab bhi chalega — yehi farq hai jo debug karte waqt kaam aata hai.

---

## LESSON 13 — Conclusion

### Overview
- Mubarak ho! Ab tumhare paas woh cheez hai jo zyada tar Kubernetes users ne kabhi haath se nahi banayi: ek cluster jis ke internals tum asal mein explain kar sakte ho.
- Plain Linux machines se functioning Kubernetes cluster tak, ek component at a time. Koi magic installer nahi, koi invisible defaults nahi. Sirf building blocks aur un ke beech interfaces.

### 13.1 Kya banaya — poora Phase 1 ek nazar mein
| Module | Building blocks | Kyun zaroori hai |
|---|---|---|
| Worker Node | containerd, runc, CNI plugins, kubelet | Yahan Pod YAML se real Linux process banta hai |
| Control Plane | etcd, kube-apiserver, kube-scheduler, kube-controller-manager | Yahan desired state store hoti hai, expose hoti hai, schedule hoti hai, reconcile hoti hai |
| Cluster | TLS bootstrapping, Flannel, kube-proxy, CoreDNS | Yahan alag machines ek Kubernetes cluster ki tarah behave karna shuru karti hain |

### 13.2 Kya seekha — 6 baatein jo yaad rakhni hain
- Kubernetes **monolith nahi hai**. Ye focused components ka set hai jo well-defined APIs se baat karte hain.
- containerd aur kubelet worker-side bridge hain — Kubernetes objects aur real Linux processes ke beech.
- etcd source of truth hai, lekin taqreeban koi us se seedha baat nahi karta. API server us ke samne khara hai aur baaki system ko possible banata hai.
- Scheduler aur controller-manager ka kaam alag hai: ek placement decide karta hai, doosra actual state ko desired state se milata rehta hai.
- Cluster networking ek feature nahi hai. Ye concerns ka stack hai: Pod-to-Pod reachability, Service routing, aur DNS-based discovery.
- Certificates, kubeconfigs, bootstrap tokens, aur RBAC ceremony nahi hain. Yehi woh tareeqa hai jis se components ke beech trust banta hai.
- **Bolne wali line:** "Bohat se 'Kubernetes problems' asal mein bohat aam Linux, networking, ya certificate problems nikaltay hain — jab tumhe pata ho ke kahan dekhna hai. Yehi Very Hard Way ka asal faida hai."

### 13.3 Kya nahi kiya — Phase 1 ki limits (imandari se bolo)
- Course mein **nahi hai:** HA control plane, backups, tighter security, monitoring.
- Ye deliberate hai — Phase 1 ka maqsad foundation tha, production nahi.
- **Phase 2 ka trailer yahin se banao:** HA, multiple etcd, kubeadm/k3s/cloud — wahan production-style cluster deploy karen ge. Conclusion Phase 1 ka end hai, series ka end nahi.

### 13.4 Agle steps — practical exercises (viewer ke liye homework)
1. Cluster dobara banao, is baar memory se ya apni automation se.
2. Apne manual setup ko `kubeadm` se compare karo — pehchano ke woh tumhare liye kya generate karta hai.
3. Ek subsystem replace karo: masalan Flannel + kube-proxy ki jagah Cilium try karo.
4. Production wali cheezein add karo jo course ne chhori hain: highly available control plane, backups, tighter security, monitoring.
- **Aakhri exercise (video mein karo):** Ek ek component band karo, aur logs parhne se pehle symptom predict karo. Har flag ya config file yaad rakhne ki zaroorat nahi hai. Jo zaroori hai woh ye hai ke tumhe pata ho ke har component kis cheez ka zimmedar hai, kis par dependent hai, aur us ke gayab hone par failure kaisa dikhta hai. Ye samajh kubeadm use karo ya managed control plane, har jagah kaam aati hai.

### 13.5 Phase 2 ka trailer — Rao bolen ge
- "Phase 1 mein hum ne Kubernetes ko haath se banaya, ek ek component ko dekh kar. Phase 2 mein hum is ko production ki taraf le jayen ge — multiple etcd, multiple master, multiple worker, kubeadm se, k3s se, aur cloud par. Wahan hum woh sab karen ge jo Phase 1 mein chhora tha: HA, backup, aur security jo hum ne har lesson mein attack karke dikhayi thi."

---

## PHASE 1 — FINAL CHECK (poora Phase 1 complete hone ka checklist)
- [ ] Script 01: Lessons 01–04 — Welcome, Tips, containerd, kubelet (static Pod chal raha hai)
- [ ] Script 02: Lessons 05–09 — etcd, apiserver, scheduler, controller-manager, joining (cluster bana, Node Ready, pehla workload chala)
- [ ] Script 03: Lessons 10–13 — Flannel (cross-node), kube-proxy (ClusterIP), CoreDNS (naam se), Conclusion
- [ ] Har lesson mein fail pehle dikhaya, phir fix, phir check — koi error chhupaya nahi?
- [ ] Har component systemd service hai, koi Pod mein control plane nahi?
- [ ] Koi iximiuz playground / promotion video mein nahi aaya?
- [ ] Har lesson ke end par dependency bola: kaun kis par dependent hai, kiske baghair kya toot-ta hai?
- [ ] Certificates inventory poori hai? (Script 02 Addendum A + Flannel/Coredns/kube-proxy certs)
- [ ] Cluster ka final state: 1 control-plane, 2 workers, Flannel, kube-proxy, CoreDNS — sab systemd services, `kubectl get nodes` Ready, `kubectl get pods -A` chal raha hai?

## Sources for Script 03 (verify karne ke liye)
- iximiuz Network lesson (full text, 2026-11-17): flannel cert/RBAC, install, net-conf.json, 10-flannel.conflist, same-node + cross-node checks, VXLAN route table.
- iximiuz kube-proxy lesson (intro, 2026-11-17) + one-page summary: Service/ClusterIP/EndpointSlice concept, installs v1.34.0, iptables mode, config.yaml clusterCIDR, KUBE-SERVICES check, ClusterIP check.
- iximiuz CoreDNS lesson (full text, 2026-11-17): cert CN=system:coredns, kubeconfig, RBAC, install 1.12.2, coredns user, Corefile full, unit file, 4 name forms, full request path.
- iximiuz Conclusion lesson (full text, 2026-11-17): what built, what learned, where to go next, 4 exercises, stop-one-component exercise.
- One-page summary (kubernetes.tahrioui.de): Lessons 10–12 ke install/check blocks.
- **Honest gaps (perform karke fill karo):** kube-proxy config.yaml ke baaki fields + unit file exact flags (lesson page ke toggles se verify karo — sirf clusterCIDR + iptables mode verified hai); kubelet DNS config file ka exact naam (clusterDNS/clusterDomain values verified, file naam verify karna hai); manual bridge config ka exact JSON (subnet values verified, file verify karni hai); control-plane IP `172.16.0.2` lab-specific hai — apne VM ka IP use karo.
