---
theme: ../theme.json
author: Rao Shahzaib
date: MMMM, YYYY
paging: Slide %d / %d
---

# Kubernetes Andar Se

> Kubernetes ko andar se samjho — ek ek component haath se bana kar

---

## Ye Series Kis Ke Liye Hai?

> Tum Pod chala lete ho, lekin nahi jante ke peeche kya hota hai

- `kubectl apply` karte ho — lekin API server se etcd tak kya hota hai, nahi pata
- Pod chalta hai — lekin containerd, runc, CRI kya karte hain, nahi pata
- Network lag jati hai — lekin Flannel, kube-proxy, CoreDNS kaun sa hissa hai, nahi pata
- Ye series un sawalon ke liye hai jo 3-4 saal seekhne ke baad bhi dimagh mein rehte hain

---

## Kubernetes Kya Hai? (Ek Line Mein)

> Kubernetes containers ko chalane, scale karne aur manage karne ka system hai

- Ye **ek program nahi hai** — ye components ka set hai
- Har component ka ek kaam hai, aur woh APIs se baat karta hai
- Is series mein hum har component ko **alag se, haath se** banayen ge

```
User
  │
  ▼
kubectl ──▶ kube-apiserver ──▶ etcd
              │      │      │
              ▼      ▼      ▼
          scheduler  controller  kubelet
                                   │
                                   ▼
                              containerd ──▶ runc ──▶ container
```

---

## Very Hard Way Kyun?

> kubeadm install kar lo — 5 minute mein cluster. To phir Hard Way kyun?

- kubeadm sab kuch chhupa deta hai — tumhe pata hi nahi chalta ke us ne kya banaya
- Hard Way mein **tum** har component download karte ho, configure karte ho, systemd service banate ho
- Jab kuch toot-ta hai, tumhe pata hota hai ke kahan dekhna hai — kyunki tum ne banaya hai
- Future mein jab CRDs aur custom plugins banayen ge, to yehi foundation kaam aaye gi

---

## Course Ka Plan — Phase 1 (Chapter 0 / Bootstrap)

> 13 lessons, 4 modules — poora cluster haath se

- **Module 1 — Introduction:** Welcome, Tips & Tricks
- **Module 2 — Worker Node:** containerd, kubelet
- **Module 3 — Control Plane:** etcd, kube-apiserver, kube-scheduler, kube-controller-manager
- **Module 4 — Cluster:** Joining Nodes, Network (Flannel), kube-proxy, CoreDNS, Conclusion
- Har lesson ka pattern ek hi hai: Download → Configure → systemd → Observe
- Pehle FAIL dikhao, phir fix karo, phir check karo

---

## Lab Kya Hoga?

> Apni Proxmox VM — koi playground nahi, koi promotion nahi

- 1 control-plane + 2 workers — apni VMs par
- Viewer WSL ya kisi bhi OS par Linux VM par repeat kar sakta hai
- Versions lock hain: Kubernetes v1.34.0, containerd 2.2.1, etcd v3.6.4, Flannel v0.27.2, CoreDNS 1.12.2

---

## Is Course Ke Baad Kya Mile Ga?

- Har component ka kaam, us ki dependency, aur us ke baghair kya toot-ta hai — ye sab tum bata sakoge
- `kubectl apply` ke peeche ka poora safar — API server se etcd, scheduler, kubelet, containerd, runc tak
- Security: har lesson mein jo insecure step hai, us ka attack dikhaya, phir fix dikhaya
- Phase 2 mein: HA cluster — multiple etcd, kubeadm, k3s, cloud

---

# Agli Lesson: Tips & Tricks

> Terminal, sudo, systemd aur troubleshooting — woh basics jo har lesson mein kaam aayen ge
