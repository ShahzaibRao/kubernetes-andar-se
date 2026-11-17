---
theme: ../theme.json
author: Rao Shahzaib
date: MMMM, YYYY
paging: Slide %d / %d
---

# containerd — Lesson 03

> Worker node ka pehla component: container ko chalane wala engine

```bash
echo "containerd" | figlet -f small -w 90
```

---

## containerd Kya Hai?

> containerd images pull karta hai aur containers manage karta hai — lekin process khud start nahi karta

- containerd **runtime nahi hai** — woh OCI runtime (runc) ko kaam deta hai
- kubelet containerd se CRI (Container Runtime Interface) se baat karta hai
- `nerdctl` containerd ka CLI hai — Docker jaisa experience, containerd ke upar

```
kubelet ──▶ CRI ──▶ containerd ──▶ containerd-shim ──▶ runc ──▶ container
              │              │
              │              └── image pull + store
              └── CNI plugins (/opt/cni/bin) — network
```

---

## Objectives — Is Lesson Ke Baad

- containerd install aur configure kar sako ge
- Samajh sako ge ke containerd, shim aur runc mein farq kya hai
- `ctr` aur `nerdctl` se container chala sako ge
- CNI plugins kyun chahiye, ye bata sako ge

---

## Pehle FAIL Dikhao — containerd Akela

> containerd install karo, runc ke baghair container chalao — fail hoga

```bash
curl -fsSLO https://github.com/containerd/containerd/releases/download/v2.2.1/containerd-2.2.1-linux-amd64.tar.gz
```

---

```bash
tar xzvof containerd-2.2.1-linux-amd64.tar.gz
sudo install -m 755 bin/* /usr/local/bin
```

---

```bash
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml
```

---

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now containerd
sudo systemctl status containerd --no-pager
```

---

```bash
sudo ctr run --rm ghcr.io/sagikazarmark/docker-hello-world:latest hello
```

---

## Fail Ka Matlab — runc Missing Hai

> Error parho: `exec: "runc": executable file not found in $PATH`

- containerd ne container banane ki koshish ki
- Process start karne ke liye us ko **runc** chahiye — jo hai hi nahi
- containerd manager hai, executor nahi — yehi farq hai jo log bhool jate hain

---

## Fix — runc Install Karo (v1.4.0)

```bash
curl -fsSLO https://github.com/opencontainers/runc/releases/download/v1.4.0/runc.amd64
sudo install -m 755 runc.amd64 /usr/local/bin/runc
runc --version
```

---

## Config — SystemdCgroup = true

> containerd aur kubelet dono ko ek hi cgroup driver chahiye — warna do manager ek hi container ko khinchte hain

```bash
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
grep -A2 'runc.options' /etc/containerd/config.toml
```

---

```bash
sudo systemctl restart containerd
sudo systemctl status containerd --no-pager
```

---

## nerdctl + CNI Plugins — Network Ke Baghair Container Adhoora

> nerdctl `bridge` CNI plugin maangta hai — `/opt/cni/bin` mein kuch nahi to container network nahi mile ga

```bash
curl -fsSLO https://github.com/containerd/nerdctl/releases/download/v2.2.1/nerdctl-2.2.1-linux-amd64.tar.gz
tar xzvof nerdctl-2.2.1-linux-amd64.tar.gz
sudo install -m 755 nerdctl /usr/local/bin
```

---

```bash
curl -fsSLO https://github.com/containernetworking/plugins/releases/download/v1.9.0/cni-plugins-linux-amd64-v1.9.0.tgz
sudo mkdir -p /opt/cni/bin
sudo tar xzvof cni-plugins-linux-amd64-v1.9.0.tgz -C /opt/cni/bin
ls /opt/cni/bin | head
```

---

## Check Karo — podinfo Jawab De Raha Hai?

```bash
sudo nerdctl run -d --name podinfo -p 9898:9898 ghcr.io/stefanprodan/podinfo:latest
```

---

```bash
PODINFO_IP=$(sudo nerdctl inspect --format '{{ .NetworkSettings.IPAddress }}' podinfo)
curl -f "http://${PODINFO_IP}:9898"
```

---

```bash
sudo nerdctl ps
sudo nerdctl inspect podinfo --format '{{ .NetworkSettings.IPAddress }}'
```

---

## Summary — Kya Seekha

- containerd manager hai — images, containers, snapshots sambhalta hai
- runc executor hai — asal process woh start karta hai, OCI runtime spec ke mutabiq
- containerd-shim beech mein hai — containerd aur runc ke darmiyan
- CNI plugins network dete hain — `/opt/cni/bin` ke baghair container ko IP nahi milta
- **Dependency:** containerd ke baghair kubelet Pod chala nahi sakta. runc ke baghair containerd container create nahi kar sakta

---

# Agli Lesson: kubelet — Lesson 04

> kubelet Pod ko containerd se kaise chalata hai — aur static Pods kya hote hain
