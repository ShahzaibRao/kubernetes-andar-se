# Kubernetes Andar Se — Phase 1 Roadmap (Draft v1)
Source: iximiuz Labs "Kubernetes the (Very) Hard Way" + Rao's Notion copy
Date: 2026-11-17 | Status: DRAFT — awaiting Rao's answers on lab, versions, security/HA placement

## Source verified
- iximiuz course: https://labs.iximiuz.com/courses/kubernetes-the-very-hard-way-0cbfd997
- Notion: https://app.notion.com/p/Kubernetes-the-Very-Hard-Way-3065ecbeb7378046a5e6cf22c42e9c2f
- Inspired by Kelsey Hightower's Kubernetes the Hard Way. Bottom-up: Worker Node first, then Control Plane, then Cluster.
- Pattern per lesson: Download → Configure manually → Run as systemd service → Observe behaviour.
- Control plane runs as systemd services, NOT as Pods. That keeps each component separate and visible.
- Versions in course: Kubernetes v1.34.0, containerd 2.2.1, runc v1.4.0, CNI plugins v1.9.0, etcd v3.6.4, Flannel v0.27.2 + flannel CNI v1.7.1-flannel2, CoreDNS 1.12.2.
- Course is for learning, NOT production. Some steps deliberately use simplified / insecure configs.

## Lab (LOCKED 2026-11-17)
- NO iximiuz Labs playground in videos. Rao creates a separate dedicated VM on his own Proxmox and configures everything there.
- iximiuz course + Notion are syllabus/reference only — NOT to be promoted (not a promoter, not paid).
- Goal: viewers can reproduce every step themselves on WSL or any OS, not just watch for understanding.
- VM count: TBD / flexible — add VMs as content design requires, no fixed number now.

## Phase 1 purpose (LOCKED 2026-11-17 — Rao's words)
Phase 1 is NOT "etcd hota hai, kubelet hota hai" theory. Each component is manually configured + demonstrated live so the student SEES how they integrate:
- Kaun kis ke sath integrated hai, kaun kis par dependent hai, kaun independent hai
- Kiske hone se kya farq padta hai, kiske na hone se kya toot-ta hai
- kubeadm / k3s / cloud jo systemd mein khud configure karke chala dete hain, woh backend mein kya kar rahe hain — taake student ke dimagh ke peeche wale sawal clear ho jayen
- Ye foundation future ke liye hai: jab CRDs aur custom plugins design karna sikhayenge, to student ko pata ho ke kin cheezon ko dimagh mein rakh kar design karna hai
- Chapter 0 / Bootstrap = yehi Phase 1 hai

## Phase 2 (separate, after Phase 1)
- HA cluster: multiple etcd on different nodes, different VMs on Rao's Proxmox
- Multiple master + worker nodes, configured with kubeadm, with k3s, and on cloud
- Production-style cluster deployment lives HERE, not in Phase 1

## The 13 lessons — exact mapping

### Module 1: Introduction
| # | Lesson | What it installs / does | Key check |
|---|--------|-------------------------|-----------|
| 01 | Welcome | Course intro, bottom-up approach, playground has prior lessons pre-built | — |
| 02 | Tips and Tricks | systemd workflow: unit in /etc/systemd/system/, daemon-reload, systemctl status, journalctl -u | `systemctl status kubelet`, `journalctl -u kubelet` |

### Module 2: Worker Node
| # | Lesson | What it installs / does | Key check |
|---|--------|-------------------------|-----------|
| 03 | containerd | containerd 2.2.1, runc v1.4.0, nerdctl 2.2.1, CNI plugins v1.9.0. Config /etc/containerd/config.toml, SystemdCgroup=true. OCI runtime + CNI concepts, containerd-shim, ctr vs nerdctl, namespaces (k8s.io vs default) | `ctr run` fails without runc, then succeeds; nerdctl fails without CNI bridge plugin, then podinfo answers on :9898 |
| 04 | kubelet | kubelet v1.34.0, crictl v1.34.0, kubeletctl v1.13. Config split in /var/lib/kubelet/config.d/ (50-static-pods, 70-authnz, 99-cri). CRI, RunPodSandbox, pause container, static Pods from /etc/kubernetes/manifests | `curl -sfk https://localhost:10250/pods`, podinfo on localhost:9898. NOTE: this lesson sets anonymous=true + AlwaysAllow — insecure on purpose, fixed in Lesson 09 |

### Module 3: Control Plane
| # | Lesson | What it installs / does | Key check |
|---|--------|-------------------------|-----------|
| 05 | etcd | etcd/etcdctl/etcdutl v3.6.4. Starts plain HTTP first, then TLS. Own CA separate from K8s CA. Only apiserver talks to etcd. Single member, Raft quorum explained (odd number, majority write) | `etcdctl endpoint health` over TLS, client-cert auth |
| 06 | kube-apiserver | kube-apiserver + kubectl v1.34.0, auger 1.0.3. Port :6443, --service-cluster-ip-range=10.96.0.0/12, --authorization-mode=Node,RBAC. Static token file first, then K8s CA + client certs. Admin cert O=system:masters. Data in etcd under /registry as protobuf. curl directly to API | `kubectl cluster-info`, curl namespace create/patch, read back from etcd |
| 07 | kube-scheduler | kube-scheduler v1.34.0, CN=system:kube-scheduler, kubeconfig /etc/kubernetes/scheduler.conf. Manual Binding demo first: Pod with empty spec.nodeName, post Binding via curl | New Pod gets nodeName; stays Pending (no kubelet yet) |
| 08 | kube-controller-manager | kube-controller-manager v1.34.0, CN=system:kube-controller-manager. Reconciliation loops. Without it ReplicaSet creates nothing; with it 3 Pods appear + self-healing. Also --allocate-node-cidrs (10.244.0.0/16, /24 per node) + signs CSRs | `kubectl get pods -l app=podinfo` → 3 Pods |

### Module 4: Cluster
| # | Lesson | What it installs / does | Key check |
|---|--------|-------------------------|-----------|
| 09 | Joining Nodes to the Cluster | TLS bootstrapping: worker gets only CA + bootstrap token (abcdef.0123456789abcdef, Secret type bootstrap.kubernetes.io/token). kubelet sends CSR, RBAC auto-approves, controller-manager signs. Node registers NotReady (no CNI) → loopback CNI makes it Ready. Apiserver flags: --enable-bootstrap-token-auth=true | `kubectl wait --for=condition=Ready node worker`, `kubectl get nodes` |
| 10 | Network | 2 workers (worker-1 10.244.0.0/24, worker-2 10.244.1.0/24). Bridge CNI first (same-node OK, cross-node fails), manual routes by hand, then Flannel v0.27.2 VXLAN (flannel.1), net-conf.json 10.244.0.0/16 | Cross-node curl podinfo-worker-2 :9898/version — times out before, works after |
| 11 | kube-proxy | kube-proxy v1.34.0, iptables mode, config /etc/kubernetes/kube-proxy/config.yaml, clusterCIDR 10.244.0.0/16. Service ClusterIP exists only as iptables rule in KUBE-SERVICES chain, random Pod pick | `iptables -t nat -L KUBE-SERVICES`, curl ClusterIP:80 |
| 12 | CoreDNS | CoreDNS 1.12.2 as systemd service on control-plane, listens 172.16.0.2:53. kubelet clusterDNS=[172.16.0.2], clusterDomain=cluster.local. Watches Services via API, answers cluster.local | `kubectl exec client -- cat /etc/resolv.conf`, `curl http://podinfo:80` |
| 13 | Conclusion | What course leaves out + next steps: rebuild from memory, compare with kubeadm, replace Flannel/kube-proxy with Cilium, stop one component at a time and predict symptom | — |

## What the course DELIBERATELY leaves out (Lesson 13 says so)
- Highly available control plane (single apiserver, single etcd member in course)
- Backups (no etcd backup/restore lesson)
- Tighter security (insecure steps on purpose: kubelet anonymous, etcd plain HTTP first, token auth first)
- Monitoring (no Prometheus/Grafana)

## Rao's additions — to be placed in Phase 1 (his call, 2026-11-17)
These are NOT in iximiuz. Placement TBD after his answers:
1. **Security thread per lesson (5-level depth):** Kya hai → Kaise kaam karta hai → Andar kya hai (live proof) → Kaise tootta hai (attack demo) → Kaise secure karein + tradeoff. Examples already named: kubelet :10250 anonymous attack, privileged pod escape, hostPath misuse, etcd without TLS, token auth vs client certs.
2. **HA:** multiple etcd (3/5, Raft quorum, leader loss demo), 3 control-plane nodes + load balancer, apiserver HA.
3. **PKI deep:** CA per component (etcd CA separate, K8s CA), cert generation, rotation, kubeconfig per component.
4. **Storage deep (Pod lesson):** OverlayFS lowerdir/upperdir/merged/workdir, image layer caching, 100 containers sharing one read layer.
5. **Future phases only:** kubeadm / EKS / GKE comparison — NOT Phase 1.

## Questions for Rao (before finalising roadmap v1)
1. Lab: viewers will follow on iximiuz playground, or on Rao's own VMs (OCI free / Proxmox / local)? Course topology is 1 control-plane + 2 workers — same for videos?
2. Versions: stick to course versions (K8s v1.34.0 etc.) or use latest stable at recording time?
3. Mapping: 1 lesson = 1 video, or split deep lessons (containerd, apiserver, network) into 2 videos? Target length per video?
4. Security + HA placement: (a) attack demo inside each lesson right after the insecure step, or (b) separate security/HA block after Lesson 13? Rao's earlier call was (a) — confirm.
5. Language: video in Roman Urdu, notes/blog in English (like his articles)? Scripts written in which language?
6. Notion notes: are they a straight copy of iximiuz, or do they already contain Rao's own additions? Should roadmap v1 be built from Notion toggles (full text fetched via Notion API — verified containerd + kubelet full content exists) or from the iximiuz one-page summary?
7. First recording: start with Module 1 (Welcome + Tips) or jump straight to Lesson 03 containerd as the first real technical video?

## Status (2026-11-17)
- **Phase 1 COMPLETE — 13/13 lessons scripted.** Script 01 (Lessons 01–04), Script 02 (Lessons 05–09), Script 03 (Lessons 10–13). Rao approved Script 01 as correct/valid ("bilkul sahi hai, valid hai, pasand aaya").
- Gaps filled 2026-11-17 with verified sources: kubeletctl v1.13 install (CyberArk GitHub releases), etcd client-cert commands (course cert pattern, CN=client). Addenda added to Scripts 01–02: dependency maps, cert inventory, troubleshooting, security attack demo placements.
- Remaining honest gaps (perform karke fill karo, Script 03 Sources section mein listed): kube-proxy config.yaml ke baaki fields, kubelet DNS config file ka exact naam, manual bridge JSON verify, control-plane IP lab-specific.
