# Phase 1 — Script 01: Module 1 Introduction + Module 2 Worker Node
## Kubernetes Andar Se — Chapter 0 / Bootstrap (Very Hard Way)

**Status:** Draft v1 — Rao performs this script on his own Proxmox VM, then recording
**Source (syllabus/reference only, NOT to be promoted in video):** iximiuz Labs "Kubernetes the (Very) Hard Way" + Rao's Notion copy (same 13 lessons, 4 modules). Kelsey Hightower's Kubernetes the Hard Way is the original inspiration.
**Lab:** Rao's own dedicated Proxmox VM. NO iximiuz playground in videos. Viewers reproduce every step on WSL or any OS with a Linux VM — hands-on, not watch-only.
**Pattern har lesson mein:** Download → Haath se Configure → systemd service ki tarah Run → Observe karo ke kya hua.
**Phase 1 ka maksad:** Har component ko theory mein "hota hai" keh kar nahi, haath se configure karke dikhana — kaun kis se jura hai, kaun kis par dependent hai, kaun independent hai, kiske na hone se kya toot-ta hai. Taake kubeadm / k3s / cloud jo peeche systemd mein khud kar dete hain, woh backend sawaal clear ho jayen. Ye foundation aage CRDs / custom plugin design ke liye hai.
**Phase 2 (alag, baad mein):** HA cluster — multiple etcd, multiple master/worker, kubeadm, k3s, cloud, production-style deploy. Phase 1 mein NAHI.

---

## Version table — poori Phase 1 ke liye (course versions, lock kar lo)

| Component | Version |
|---|---|
| Kubernetes (kubelet, kube-apiserver, kube-scheduler, kube-controller-manager, kube-proxy, kubectl) | v1.34.0 |
| containerd | 2.2.1 |
| runc (OCI runtime) | v1.4.0 |
| nerdctl | 2.2.1 |
| CNI plugins | v1.9.0 |
| etcd / etcdctl / etcdutl | v3.6.4 |
| Flannel (flanneld) | v0.27.2 |
| Flannel CNI plugin | v1.7.1-flannel2 |
| CoreDNS | 1.12.2 |
| crictl | v1.34.0 |
| kubeletctl | v1.13 |
| auger (etcd data dekhne ke liye, Lesson 06) | 1.0.3 |

**Note:** Video mein versions screen par dikhao. Agar recording ke waqt naya stable aa jaye to version change mat karo beech mein — ek series mein ek version set rakho, warna viewer confuse hoga.

## Lab environment — Script 01 ke liye

- Ek dedicated Linux VM (Ubuntu) Proxmox par. Viewer ke liye: WSL ya kisi bhi OS par ek Linux VM.
- Har lesson ke baad VM snapshot / note: har lesson pichle lesson ki state par banta hai, dobara zero se shuru mat karo.
- Control plane systemd services ki tarah chalega, Pod ki tarah NAHI — taake har component alag dikhe.
- Har lesson mein ek cheez pehle FAIL karke dikhao, phir fix karo. Ye course ka style hai: error pehle, samajh baad mein.

## Prerequisites — course shuru karne se pehle viewer ko kya aana chahiye

1. Linux basics: file system, `ls`, `cat`, `mkdir`, `curl`, `tar`, `jq`.
2. `sudo` use karna aata ho, root aur non-root ka farq pata ho.
3. Container ka basic idea: image kya hai, container kya hai (deep abhi nahi, wo isi course mein aaye ga).
4. `kubectl` ka basic use (pod chalana aata ho) — lekin andar kya hota hai, ye nahi pata hona chahiye, wahi to seekhna hai.
5. Ek Linux VM + internet. Koi cloud account zaroori nahi, koi kubeadm / EKS zaroori nahi.

**Ye course kis ke liye NAHI hai:** Jo sirf `kubectl apply` karke cluster chalana chahta hai, us ke liye kubeadm / managed K8s kaafi hai. Ye un ke liye hai jo andar dekhna chahte hain.

---

## LESSON 01 — Welcome / Introduction (bolne ka script)

### 1.1 Kubernetes apne aap mein kya hai (2–3 minute, theory short)
- Kubernetes ek container orchestrator hai: woh decide karta hai ke container kahan chalega, kitne chalenge, mar jaye to dobara kaise chalega, aur un tak network kaise pahunche ga.
- Lekin is course mein hum Kubernetes ko "use" nahi karenge, hum usay "khol kar" dekhen ge.
- Kubernetes ek single program nahi hai. Ye kai components ka system hai: containerd, kubelet, etcd, kube-apiserver, kube-scheduler, kube-controller-manager, kube-proxy, CoreDNS, CNI. Har ek ka apna kaam hai.

### 1.2 Very Hard Way kyun? (yehi tumhari pehchan hai)
- Aam tutorial kehta hai: ye Pod hai, ye Deployment hai, `kubectl apply` karo, ho gaya.
- Yahan hum ulta chalenge — **bottom-up**: pehle worker node ke components, phir control plane, phir dono ko jor kar cluster.
- Har component ko pehle akela install karenge, test karenge, phir agle se jodenge. Is se pata chalta hai ke kaun kya karta hai, aur kaun kiske baghair kaam nahi karta.
- Jaise containerd akela container chala sakta hai, lekin Pod nahi bana sakta. kubelet akela static Pod chala sakta hai, lekin API server ke baghair cluster nahi bana sakta. Ye farq sirf karke samajh aata hai, parh kar nahi.

### 1.3 Course ka purpose — saaf lafzon mein
- Purpose ye nahi ke end mein production-ready cluster mil jaye. Course khud kehta hai: ye learning ke liye hai, production ke liye nahi.
- Kuch steps jaan boojh kar simple / insecure hain taake concept saaf dikhe. Production mein humesha security best practices, sahi certificate management, network policies, aur kubeadm / kops / managed service use karo.
- Course khatam karne ke baad viewer ko ye aana chahiye:
  - `kubectl run nginx` ke peeche kya hota hai, step by step.
  - Har component kahan rehta hai, kis port par bolta hai, kis se baat karta hai.
  - Ek component band ho jaye to kya symptom aaye ga, aur log kahan dekhen ge.
  - kubeadm / k3s / cloud ne peeche kya configure kiya hoga.

### 1.4 Course khatam karne ke baad kya fayda hoga
- Interview mein "Pod kya hai" ka jawab nahi, "Pod banta kaise hai" ka jawab de sako ge.
- Production mein Pod Pending hai, CrashLoop hai, Node NotReady hai — to guess nahi, component by component debug kar sako ge.
- Aage CRD aur custom plugin design karna ho to pata hoga ke kin cheezon ko zehen mein rakhna hai.
- Ye skill har us DevOps engineer ko alag karta hai jo sirf YAML apply karta hai.

### 1.5 Har bande ko ye course kyun karna chahiye (1 minute)
- Kyunki Kubernetes duniya mein sab se zyada use hone wala orchestrator hai, lekin zyada log usay black box ki tarah chalate hain.
- Jo andar samajhta hai, wohi tootne par theek kar sakta hai, aur wohi aage design kar sakta hai.

---

## LESSON 02 — Tips & Tricks (ye lesson chota hai, lekin har video mein kaam aaye ga)

### 2.1 Config files ko edit karna — sudo, sudoedit, sudo su
- Config files root ki hoti hain. Seedha `nano /etc/containerd/config.toml` likho ge to permission error aaye ga.
- Sahi tareeqa:
```bash
sudoedit /etc/containerd/config.toml
# ya
sudo mkdir -p /etc/containerd
sudoedit /etc/containerd/config.toml
```
- `sudoedit` editor kholta hai, save karte hi file root ki ban jati hai, safe hai.
- `sudo su` se poora shell root ban jata hai — course mein kabhi kabhi zaroorat padti hai, lekin har command mein `sudo` lagana behtar aadat hai. Video mein batao ke kab `sudo su` use kar rahe ho aur kyun.
- `sudo` ke baghair system files mat chhedo, aur root shell ko aise hi khula mat chhoro.

### 2.2 systemd basics — har component isi tarah chalega
- Har component ki unit file is jagah hoti hai:
```bash
/etc/systemd/system/<service>.service
# masalan: /etc/systemd/system/containerd.service, kubelet.service
```
- Unit file change karo to pehle reload, phir restart:
```bash
sudo systemctl daemon-reload
sudo systemctl restart containerd
```
- Ek hi command mein enable + start:
```bash
sudo systemctl enable --now containerd
sudo systemctl enable --now kubelet
```
- `enable` ka matlab hai: boot par khud start ho. `--now` ka matlab hai: abhi bhi start kar do.

### 2.3 Troubleshooting — service na chale to kya karna hai
- Pehle status dekho:
```bash
sudo systemctl status containerd
sudo systemctl status kubelet
sudo systemctl status etcd
```
- Phir logs dekho (yehi asal debugging hai):
```bash
sudo journalctl -u containerd.service
sudo journalctl -u kubelet.service
sudo journalctl -u kubelet.service -f
```
- `-f` ka matlab hai live logs dekhna (follow).
- Rule: error ko chhupao mat. Video mein error dikhao, phir log parho, phir fix karo. Viewer ko yehi seekhna hai.
- Service edit karni ho to unit file edit karo, phir `daemon-reload` + `restart` — reload bhool jao ge to purani config chalti rahe gi.

### 2.4 Is lesson ka check
- Viewer `systemctl status` aur `journalctl -u` chala sake, aur `sudoedit` se ek test file edit karke save kar sake.
- Agar ye aata hai, to agle lessons mein hum command par nahi, concept par focus karenge.

---

## LESSON 03 — containerd (Module 2: Worker Node — pehla asli component)

### Overview
- Is lesson mein hum **containerd** ko dekhen ge — industry-standard container runtime jo Kubernetes cluster mein container chalata hai.
- Ye worker node ka pehla component hai. Is ke baghair node par koi container nahi chal sakta.

### Objectives (lesson ke end par viewer ko ye aana chahiye)
- containerd ka Kubernetes cluster mein role samajhna.
- containerd ko scratch se install aur configure karna.
- containerd ke aas paas ke tools aur ecosystem ko dekhna.
- Samajhna ke containerd **CNI (Container Network Interface)** aur **OCI (Open Container Initiative)** jaise open standards par kaise kaam karta hai.

### 3.1 containerd kya hai?
- containerd ek **high-level container runtime** hai. Ye Docker se nikla tha, ab CNCF maintain karta hai.
- Ye Kubernetes mein ye 5 kaam karta hai:
  1. **Container Lifecycle Management:** container banana, start, stop, delete karna.
  2. **Image Management:** registry se image pull karna, store karna, manage karna.
  3. **Storage Management:** container filesystem aur volume mounts sambhalna.
  4. **Network Management:** CNI plugins ke sath mil kar container networking karna.
  5. **Runtime Management:** low-level runtime (runc) se baat karna.
- **Diagram bol kar samjhao (video mein diagram dikhao):**
  - Upar kubelet → neeche containerd → neeche containerd-shim → neeche runc → neeche Linux kernel (namespaces + cgroups).
  - containerd "kya" decide karta hai (kaunsa container, kaunsi image, kaunsa network, kaunsa storage). runc "kaise" karta hai (process kaise banega, isolation kaise hogi).
- containerd khud container process start NAHI karta. Woh kaam woh OCI runtime ko deta hai.

### 3.2 containerd install karna — step by step
```bash
CONTAINERD_VERSION=2.2.1
```
```bash
curl -fsSLO "https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz"
sudo tar xzvfC "containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz" /usr/local
```
- systemd unit file download karo:
```bash
sudo wget -P /etc/systemd/system "https://raw.githubusercontent.com/containerd/containerd/v${CONTAINERD_VERSION}/containerd.service"
```
- Service start karo:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now containerd
```
- Check karo:
```bash
sudo systemctl status containerd
sudo journalctl -u containerd.service
```
- **Bolne wali line:** "Mubarak ho, containerd chal raha hai — lekin abhi is se container nahi chalega, kyunki OCI runtime abhi nahi hai. Yehi hum agle step mein dekhte hain."

### 3.3 containerd ke sath interact karna — `ctr`
- containerd ka apna CLI hai `ctr`. Ye Docker CLI jaisa nahi, ye low-level hai.
- `ctr` ko pehle image **manually pull** karni padti hai, phir run karna padta hai:
```bash
sudo ctr images pull ghcr.io/sagikazarmark/docker-hello-world:latest
```
```bash
sudo ctr run --rm ghcr.io/sagikazarmark/docker-hello-world:latest hello
```
- **Ye command FAIL hogi — aur fail honi chahiye, video mein dikhao:**
```
ctr: failed to create shim task: OCI runtime create failed:
unable to retrieve OCI runtime error
(open /run/containerd/io.containerd.runtime.v2.task/default/hello/log.json: no such file or directory):
exec: "runc": executable file not found in $PATH
```
- Samjhao: containerd ne image pull kar li, lekin process banane ke liye runc chahiye, jo abhi nahi hai. Is liye fail hua.

### 3.4 OCI runtime install karna — runc, aur containerd-shim kya hai?
- **OCI Runtime Specification:** ye standard batata hai ke low-level par container kaise banta, start hota, manage hota hai. Is se containerd kisi bhi OCI-compliant runtime ke sath kaam kar sakta hai, bina code change kiye.
- Pehle Docker Engine monolithic tha — sab kuch ek jagah. Phir ecosystem ne modular tareeqa apnaya: high-level runtime (containerd) + low-level runtime (runc) + network (CNI).
- **containerd-shim kya hai?** Ye containerd aur OCI runtime ke beech halka sa bridge process hai. Ye container ko chalata rehta hai agar containerd crash bhi ho jaye, I/O sambhalta hai, aur process khatam hone par usay reap karta hai.
- Doosre OCI runtimes: `crun` (cgroups/namespaces, behtar performance) aur `gVisor` (user-space kernel, VM jaisi isolation). Reference runtime `runc` hai, sab se zyada use aur test hua hua.
- runc install karo:
```bash
RUNC_VERSION=v1.4.0
```
```bash
curl -fsSLO "https://github.com/opencontainers/runc/releases/download/${RUNC_VERSION}/runc.amd64"
sudo install -m 755 runc.amd64 /usr/local/sbin/runc
```
- containerd ko systemd cgroup driver use karne ke liye configure karo. File:
  - Path: `/etc/containerd/config.toml`
```toml
version = 3

[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.runc.options]
  SystemdCgroup = true
```
- **Samjhao — SystemdCgroup = true kyun zaroori hai:** cgroups CPU, memory, I/O ko limit aur isolate karte hain. Agar systemd aur runtime alag alag cgroup tree manage karenge to memory / CPU pressure mein instability aaye gi. systemd init hai, to dono ko ek hi tree use karna chahiye.
- File banane ka command:
```bash
sudo mkdir -p /etc/containerd
sudoedit /etc/containerd/config.toml
```
- Restart karo:
```bash
sudo systemctl restart containerd
```
- Ab wahi command dobara chalao — ab chalegi:
```bash
sudo ctr run --rm ghcr.io/sagikazarmark/docker-hello-world:latest hello
```
- **Bolne wali line:** "Pehle fail hua tha kyunki runc nahi tha. Ab runc hai, to container chal gaya. Yehi farq hai high-level aur low-level runtime ka."

### 3.5 nerdctl install karna — aur kyun?
- `ctr` low-level hai, rozana use ke liye dostana nahi hai, khaas tor par jise Docker CLI aata hai.
- `nerdctl` (contaiNERD CTL) Docker CLI jaisa interface deta hai containerd ke upar.
- Install karo:
```bash
NERDCTL_VERSION=2.2.1
```
```bash
curl -fsSLO "https://github.com/containerd/nerdctl/releases/download/v${NERDCTL_VERSION}/nerdctl-${NERDCTL_VERSION}-linux-amd64.tar.gz"
tar xzvof "nerdctl-${NERDCTL_VERSION}-linux-amd64.tar.gz"
sudo install -m 755 nerdctl /usr/local/bin
nerdctl completion bash | sudo tee /etc/bash_completion.d/nerdctl
```
- `ctr` ke muqable mein `nerdctl` image pull aur network setup khud kar leta hai. Test karo:
```bash
sudo nerdctl run --rm ghcr.io/stefanprodan/podinfo:latest /home/app/podinfo --version
```
- **Ye bhi FAIL hogi — dikhao:**
```
FATA failed to create shim task: OCI runtime create failed:
runc create failed: unable to start container process:
error during container init: error running createRuntime hook #0:
failed to call cni.Setup: plugin type="bridge" failed (add): failed to find plugin "bridge" in path [/opt/cni/bin]
```
- Samjhao: `nerdctl` by default bridge network banana chahta hai. Bridge plugin `/opt/cni/bin` mein nahi hai, is liye fail hua. `ctr` ye koshish karta hi nahi, is liye woh pehle chal gaya tha.
- Temporary workaround (sirf samajhne ke liye, production mein nahi):
```bash
sudo nerdctl run --net host --rm ghcr.io/stefanprodan/podinfo:latest /home/app/podinfo --version
```
- **Bolne wali line:** "Ye sirf workaround hai. Production mein CNI plugins install karo ge, jo agle step mein karte hain."

### 3.6 CNI plugins install karna
- containerd process execution OCI runtime ko deta hai, waise hi network configuration CNI ko deta hai.
- **CNI (Container Network Interface)** ek specification hai: container runtime network namespace kaise banaye, IP kaise de, container aur host ke beech connectivity kaise kare.
- Plugins kahan hote hain: `/opt/cni/bin` — ye convention hai, runtime aur network add-on dono yahin dekhte hain.
- Install karo:
```bash
CNI_PLUGINS_VERSION=v1.9.0
```
```bash
curl -fsSLO "https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_VERSION}/cni-plugins-linux-amd64-${CNI_PLUGINS_VERSION}.tgz"
sudo mkdir -p /opt/cni/bin
sudo tar xzvofC "cni-plugins-linux-amd64-${CNI_PLUGINS_VERSION}.tgz" /opt/cni/bin
```
- Ab podinfo chalao:
```bash
sudo nerdctl run -d --name podinfo ghcr.io/stefanprodan/podinfo:latest
sudo nerdctl ps
```
- Verify karo:
```bash
PODINFO_IP=$(sudo nerdctl inspect --format '{{ .NetworkSettings.IPAddress }}' podinfo)
curl -f "http://${PODINFO_IP}:9898"
```
- **Check pass hona chahiye:** podinfo jawab deta hai port 9898 par.

### 3.7 Summary — containerd lesson
- containerd CNCF ka industry-standard high-level runtime hai, Docker se nikla.
- Ye image, storage, network, lifecycle manage karta hai, lekin process runc se banwata hai.
- OCI standard se koi bhi compliant runtime use ho sakta hai. CNI standard se network plugin alag hota hai.
- `ctr` low-level hai, `nerdctl` Docker jaisa hai. Dono containerd se baat karte hain, lekin alag namespace mein containers dekhte hain (ye kubelet lesson mein aaye ga).
- **Dependency yaad rakho:** kubelet ke baghair containerd akela container chala sakta hai, lekin Pod nahi bana sakta. Yehi agle lesson ki taraf ishara hai.

### References (video mein link mat dikhao, sirf notes mein)
- containerd.io, CNCF containerd, OCI Runtime Spec (opencontainers.org), CNI (cni.dev), containernetworking/plugins, nerdctl, runc, Kelsey Hightower Kubernetes the Hard Way (inspiration).

## LESSON 04 — kubelet (Module 2: Worker Node — doosra component)

### Overview
- Is lesson mein hum **kubelet** ko dekhen ge — node agent jo har worker node par chalta hai.
- kubelet control plane aur container runtime ke beech bridge hai.

### Objectives
- kubelet ka role samajhna.
- kubelet ko scratch se install aur configure karna.
- Dekhna ke kubelet containerd se **CRI (Container Runtime Interface)** ke zariye kaise baat karta hai.
- Pod aur container ka rishta samajhna.
- kubelet se API server ke baghair Pod chalana.
- kubelet ki basic debugging seekhna.

### 4.1 kubelet kya hai?
- kubelet har node par chalne wala **primary node agent** hai. Ye Kubernetes ke "haath aur paon" hai: control plane se Pod specification leta hai, aur node par usay asal mein chala kar dikhata hai.
- kubelet ek continuous reconciliation loop mein kaam karta hai:
  1. **Watches:** API server se Pod specifications dekhta hai.
  2. **Compares:** desired state (kya chalna chahiye) vs actual state (kya chal raha hai).
  3. **Takes action:** farq khatam karne ke liye container banata / rokta hai.
  4. **Reports:** status wapas control plane ko bhejta hai.
- kubelet khud container nahi chalata. Woh containerd ya CRI-O jaise runtime ko kehta hai. Is ke liye woh **CRI** use karta hai, taake runtime badla bhi ja sake bina kubelet ko dobara compile kiye.

### 4.2 kubelet install karna — step by step
```bash
KUBE_VERSION=v1.34.0
```
```bash
curl -fsSLO "https://dl.k8s.io/${KUBE_VERSION}/bin/linux/amd64/kubelet"
sudo install -m 755 kubelet /usr/local/bin
```
- systemd unit file download karo:
```bash
sudo wget -O /etc/systemd/system/kubelet.service "https://labs.iximiuz.com/content/files/courses/kubernetes-the-very-hard-way-0cbfd997/02-worker-node/02-kubelet/__static__/kubelet.service?v=1770643331"
```
- **Rao ke liye note (video mein ye URL promotion ke tor par mat dikhao):** Apne lab mein is unit file ki apni copy rakho (apne Proxmox VM / apne notes se). Source reference sirf is script mein hai, video mein iximiuz ka zikr nahi karna.
- kubelet multiple config files ko merge kar sakta hai. Files alphabetical order mein merge hoti hain, is liye number prefix (50-, 70-, 99-) se order control hota hai.
- Pehle sirf runtime kaise baat kare, ye configure karo. File:
  - Path: `/var/lib/kubelet/config.d/99-cri.conf`
```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration

containerRuntimeEndpoint: unix:///var/run/containerd/containerd.sock
cgroupDriver: systemd
```
- File banane ka command:
```bash
sudo mkdir -p /var/lib/kubelet/config.d
sudoedit /var/lib/kubelet/config.d/99-cri.conf
```
- Service start karo:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now kubelet
```
- Check karo:
```bash
sudo systemctl status kubelet
sudo journalctl -u kubelet.service
```

### 4.3 kubelet API — port 10250
- kubelet ek HTTP API deta hai, aam tor par port **10250** par: Pod logs, exec sessions, node metrics, health.
- Aam tor par ye TLS + authentication / authorization se secure hota hai. Is lesson mein hum seekhne ke liye usay disable karenge — **production mein KABHI mat karna.**
- Config file:
  - Path: `/var/lib/kubelet/config.d/70-authnz.conf`
```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration

authentication:
  anonymous:
    enabled: true
  webhook:
    enabled: false

authorization:
  mode: AlwaysAllow
```
- **Samjhao — ye insecure kyun hai, aur kyun kar rahe hain:** Abhi API server nahi hai, is liye hum dekhna chahte hain ke kubelet akela kya karta hai. Lesson 09 (Joining Nodes) mein anonymous access band kar den ge. Har insecure step ke sath ye bolna zaroori hai: "Ye sirf seekhne ke liye hai."
- Restart karo, phir check karo:
```bash
sudo systemctl restart kubelet
curl -k https://localhost:10250/healthz
```

### 4.4 Static Pods — API server ke baghair Pod
- **Static Pod** woh Pod hai jo kubelet seedha manage karta hai, API server ke zariye nahi.
- kubelet ek directory ko monitor karta hai. Us mein Pod manifest rakho to kubelet khud us Pod ko bana aur restart karta hai.
- kubelet API server mein us ka **mirror Pod** bhi banata hai, taake `kubectl get pods` mein dikhe — lekin us mirror Pod ko API se control nahi kar sakte, sirf kubelet kar sakta hai.
- **Static Pod kyun zaroori hai?** Chicken-and-egg problem: control plane chalane ke liye Pod chahiye, Pod chalane ke liye control plane chahiye. Static Pod se control plane ke components ko kubelet se chala sakte hain, bina API server ke.
- Static Pod by default band hota hai, enable karna padta hai:
```bash
sudo mkdir -p /etc/kubernetes/manifests
```
- Config file:
  - Path: `/var/lib/kubelet/config.d/50-static-pods.conf`
```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration

staticPodPath: /etc/kubernetes/manifests
```
- Pehla static Pod — podinfo (yehi app poore course mein use hogi):
  - Path: `/etc/kubernetes/manifests/podinfo.yaml`
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: podinfo
spec:
  hostNetwork: true
  containers:
    - name: podinfo
      image: ghcr.io/stefanprodan/podinfo:latest
      ports:
        - containerPort: 9898
```
- **Samjhao — `hostNetwork: true` kya hai:** Pod host ka network namespace use karta hai, apna alag nahi banata. Pod host ka IP share karta hai, to port 9898 seedha localhost par milta hai. Static Pods jo host se seedha baat karna chahte hain, un ke liye ye aam hai.
- **Note:** Static Pod Kubernetes API objects jaise Secrets, ConfigMaps, ServiceAccounts ko reference NAHI kar sakta. Sirf node par maujood cheezein, jaise hostPath ya emptyDir, use kar sakta hai.
- Restart + verify:
```bash
sudo systemctl restart kubelet
curl -sfk https://localhost:10250/pods | jq '.items[0].metadata'
curl http://localhost:9898 | jq
```
- Pod ka naam `podinfo-worker` dikhe ga — static Pod ke naam ke sath node ka naam lag jata hai taake cluster mein unique rahe.

### 4.5 Containerd namespace ka farq — container dikhe ga kyun nahi?
- Pod chal raha hai, lekin ye command khaali dikhaye gi:
```bash
sudo nerdctl ps
sudo ctr container ls
```
- Wajah: containerd containers ko namespaces mein organize karta hai (Kubernetes jaisa). Kubernetes `k8s.io` namespace use karta hai, `nerdctl` / `ctr` by default `default` namespace dekhte hain.
- Check karo:
```bash
sudo ctr namespace ls
sudo nerdctl ps --namespace k8s.io
sudo ctr --namespace k8s.io container ls
```
- **Bolne wali line:** "Container gayab nahi hua, bas doosre namespace mein hai. Yehi woh choti cheez hai jo debugging mein ghanton zaya karti hai agar pata na ho."

### 4.6 CRI — Container Runtime Interface (deep, yehi Pod ka asal raaz hai)
- **CRI** ek gRPC API hai jo kubelet ko container runtime se baat karne ka standard tareeqa deta hai. Ye Kubernetes ka apna hissa hai (OCI aur CNI Kubernetes-agnostic hain, CRI nahi).
- Pehle Kubernetes mein sirf Docker tha, koi CRI nahi tha, kubelet Docker se tightly coupled tha. Phir CRI aaya, Docker ka implementation shim layer (dockershim) mein gaya. CRI ko Kubernetes v1.20 mein deprecate, v1.24 mein remove kar diya gaya. Docker use karna ho to `cri-dockerd` use hota hai (wahi dockershim, doosre naam se).
- CRI API ki 2 services hain:
  - **RuntimeService:** Pod sandbox aur containers manage karta hai (create, start, stop, remove).
  - **ImageService:** images manage karta hai (pull, list, remove).
- kubelet jab Pod banata hai to ye calls karta hai:
  1. **Create Pod:** `RunPodSandbox`
  2. **Start container:** `CreateContainer` + `StartContainer`
  3. **Pull image:** `PullImage`
  4. **Pod status:** `PodSandboxStatus`
- **Pod sandbox kya hai?** Ye woh shared environment hai jis mein Pod ke containers chalte hain. kubelet pehle sandbox banata hai, phir us mein containers launch karta hai.
- **Pause container kyun hai?** Linux namespaces ko zinda rehne ke liye ek running process chahiye. Traditional runtimes (Docker, containerd) sandbox set up karne ke liye ek `pause` container banate hain — ye container kuch karta nahi, bas namespaces ko hold karta hai, taake Pod ke saare containers wahi network, IPC, UTS namespace share kar saken. Pause container mar jaye to Pod ka shared environment toot jata hai.
- **Diagram bol kar samjhao:** Pod → 1 pause container (sandbox) + N app containers, sab ek hi network namespace mein, alag alag mount / process namespace mein (jo share karna ho wo share, baaki alag).

### 4.7 kubeletctl aur crictl install karna
- Versions: crictl v1.34.0, kubeletctl v1.13.
- `crictl` containerd se usi CRI interface ke zariye baat karta hai jis se kubelet baat karta hai, is liye woh `k8s.io` namespace ke containers dekh sakta hai.
- Install + basic use (verified: CyberArk kubeletctl GitHub releases, v1.13 — `kubeletctl_linux_amd64` binary):
```bash
curl -fsSLO https://github.com/cyberark/kubeletctl/releases/download/v1.13/kubeletctl_linux_amd64
sudo install -m 755 kubeletctl_linux_amd64 /usr/local/bin/kubeletctl
kubeletctl version
```
```bash
kubeletctl pods --server https://localhost:10250 --ignoreconfig
```
- **Security demo (Rao ka attack thread, Lesson 04 mein yahin karo):** kubelet API abhi anonymous + AlwaysAllow hai. Kisi bhi machine se `curl -k https://<worker-ip>:10250/pods` se Pod list mil jati hai, aur `kubeletctl exec` se container ke andar command chal jata hai — bina kisi credential ke. Yehi woh attack hai jo Lesson 09 mein x509 auth se band hota hai. Video mein attack dikhao, phir fix dikhao.
- crictl install (kubelet ke sath aata hai — `crictl` v1.34.0, CRI endpoint `unix:///var/run/containerd/containerd.sock`):
```bash
crictl pods
crictl ps
```

### 4.8 Summary — kubelet lesson
- kubelet har node ka agent hai, control plane aur runtime ke beech bridge hai.
- Ye API server se Pod spec leta hai, containerd se CRI ke zariye container banwata hai, aur status wapas bhejta hai.
- Static Pod API server ke baghair chalta hai — yehi control plane ko Pod ki tarah chalane ka raasta hai.
- kubelet API :10250 par hai. Is lesson mein anonymous + AlwaysAllow hai — insecure, Lesson 09 mein band hoga.
- **Dependency yaad rakho:** containerd ke baghair kubelet kuch nahi kar sakta. API server ke baghair kubelet static Pod to chala sakta hai, lekin cluster ka hissa nahi ban sakta. Yehi agle module ki taraf ishara hai.

### References (notes mein, video mein link mat dikhao)
- Kubernetes CRI docs, cri-api (RuntimeService / ImageService), Kelsey Hightower Kubernetes the Hard Way (inspiration), iximiuz one-page summary Lessons 03–04.

---

## SCRIPT 01 ka final check — perform karne se pehle

- [ ] Lesson 01: Kubernetes kya hai, Very Hard Way kyun, purpose, fayda — bola?
- [ ] Lesson 02: sudoedit, systemd (daemon-reload, enable --now, status, restart), journalctl -u — dikhaya?
- [ ] Lesson 03: containerd 2.2.1 install, ctr fail (no runc), runc 1.4.0, SystemdCgroup=true, nerdctl 2.2.1, CNI 1.9.0, podinfo :9898 — sab perform hua?
- [ ] Lesson 04: kubelet 1.34.0 install, 99-cri.conf, 70-authnz.conf (insecure note bola?), 50-static-pods.conf, podinfo static Pod, :10250/pods, k8s.io namespace, CRI + pause container — sab perform hua?
- [ ] Har fail step video mein dikhaya, error chhupaya nahi?
- [ ] Koi iximiuz playground / promotion video mein nahi aaya?
- [ ] Har lesson ke end par "ye component kis par dependent hai, kiske baghair kya toot-ta hai" bola?

## Agla script (Script 02 — Module 3 Control Plane, abhi outline, detail alag script mein)
- Lesson 05 etcd — kya hai, install, interact, data kaise jata hai, key-value, Raft, TLS, single member vs quorum. (Subagent se full Notion extraction aa raha hai, us ke baad detail script likha jaye ga.)
- Lesson 06 kube-apiserver — kya hai, install, auth + authz, curl se API, kubectl, TLS, etcd mein data /registry mein protobuf.
- Lesson 07 kube-scheduler — Pod ko Node kaise deta hai, Binding, nodeName.
- Lesson 08 kube-controller-manager — reconciliation loops, ReplicaSet se Pod banana, self-healing, --allocate-node-cidrs.
- Lesson 09 Joining Nodes — TLS bootstrapping, CSR, RBAC auto-approve, Node Ready.
- Lesson 10 Network — Flannel VXLAN, cross-node Pod networking.
- Lesson 11 kube-proxy — Service ClusterIP, iptables KUBE-SERVICES chain.
- Lesson 12 CoreDNS — cluster.local, clusterDNS 172.16.0.2:53.
- Lesson 13 Conclusion — kya chhoota (HA, backup, monitoring, tighter security), aage kya try karein.

**Rao ke liye note:** Script 02 ke liye control-plane pages ki full extraction background mein chal rahi hai. Script 01 perform karo, us ke baad Script 02 us extraction se likha jaye ga — us mein etcd ke har toggle ki har line hogi, koi cheez miss nahi hogi.

---

## ADDENDUM (2026-11-17) — Rao ki ijazat se jo missing tha, add kiya gaya

### A. Dependency map — kaun kis par dependent hai, kiske baghair kya toot-ta hai
| Component | Dependent hai | Baghair is ke kya toot-ta hai |
|---|---|---|
| runc | containerd | containerd container create nahi kar sakta — "runc not found" |
| CNI plugins (/opt/cni/bin) | containerd / nerdctl | Pod/container ko network nahi milta |
| containerd | kubelet | kubelet Pod chala nahi sakta |
| kubelet | containerd + CRI | Pod nahi banta, Node register nahi hota |
| crictl | containerd CRI socket | kubelet ke Pods `nerdctl ps` mein nahi dikhte — crictl se dekho |
| kubeletctl | kubelet API :10250 | kubelet API ko debug nahi kar sakte |

### B. Troubleshooting — har fail par yehi order
```bash
sudo systemctl status <service>
sudo journalctl -u <service> -n 100 --no-pager
sudo systemctl daemon-reload
sudo systemctl restart <service>
ss -tlnp | grep -E '10250|2379|6443'
```
- Config file ka syntax check karo pehle, phir service restart karo. Log parhne se pehle symptom predict karo — yehi Conclusion (Lesson 13) ki exercise hai.
- Certificate expiry check: `openssl x509 -in <cert>.crt -noout -dates -subject`

### C. Security attack demos — kahan karna hai (Rao ka thread)
- **Lesson 04 (kubelet anonymous):** `curl -k https://<worker-ip>:10250/pods` — bina credential ke Pod list. `kubeletctl exec` se container ke andar command. Attack dikhao, phir Lesson 09 mein x509 fix dikhao.
- **Lesson 03 (containerd socket):** `/run/containerd/containerd.sock` par jis ka access hai, woh root ke barabar hai — socket permissions dikhao.
- **Lesson 05 (etcd plain HTTP):** TLS se pehle `etcdctl get --prefix /` se poora data parh lo — yehi wajah hai ke mTLS lagta hai.
- **Lesson 06 (static token):** `tokens.csv` mein token plain text hai, aur `iximiuz` token har jagah same hai — leak ho to koi bhi admin ban jata hai. Is liye TLS + client cert par jate hain.
