# Lesson 03 — containerd

## containerd Kya Hai?
> containerd images pull karta hai aur containers manage karta hai — lekin process khud start nahi karta
- containerd **runtime nahi hai** — woh OCI runtime (runc) ko kaam deta hai
- kubelet containerd se CRI (Container Runtime Interface) se baat karta hai
- `nerdctl` containerd ka CLI hai — Docker jaisa experience, containerd ke upar
```text
kubelet ──▶ CRI ──▶ containerd ──▶ containerd-shim ──▶ runc ──▶ container
              │              │
              │              └── image pull + store
              └── CNI plugins (/opt/cni/bin) — network
```

## Objectives — Is Lesson Ke Baad
- containerd install aur configure kar sako ge
- Samajh sako ge ke containerd, shim aur runc mein farq kya hai
- `ctr` aur `nerdctl` se container chala sako ge
- CNI plugins kyun chahiye, ye bata sako ge

## Pehle FAIL Dikhao — containerd Akela
> containerd install karo, runc ke baghair container chalao — fail hoga
```sh
curl -fsSLO https://github.com/containerd/containerd/releases/download/v2.2.1/containerd-2.2.1-linux-amd64.tar.gz
```

## Step 1 — containerd Install Karo
> Tarball se binary nikalo aur /usr/local/bin mein daalo
```sh
tar xzvof containerd-2.2.1-linux-amd64.tar.gz
sudo install -m 755 bin/* /usr/local/bin
```

## Step 2 — Config File Banao
> Default config banao — output ko chhupao, warna poora TOML screen bhar de ga
```sh
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml > /dev/null
wc -l /etc/containerd/config.toml
```

## Step 3 — systemd Unit File Lagao
> Tarball mein service file NAHI hoti — is ke baghair systemctl containerd ko dhoond hi nahi sakta
```sh
sudo wget -P /etc/systemd/system "https://raw.githubusercontent.com/containerd/containerd/v2.2.1/containerd.service"
```

## Step 4 — containerd Service Start Karo
> Ab systemd ko unit file ka pata hai — service enable aur start karo
```sh
sudo systemctl daemon-reload
sudo systemctl enable --now containerd
sudo systemctl status containerd --no-pager
```

## Step 5 — Pehla Container Chalao (FAIL Hoga)
> Pehle image pull karo — ctr run khud pull NAHI karta
```sh
sudo ctr images pull ghcr.io/sagikazarmark/docker-hello-world:latest
sudo ctr run --rm ghcr.io/sagikazarmark/docker-hello-world:latest hello
```

## Fail Ka Matlab — runc Missing Hai
> Error parho: `exec: "runc": executable file not found in $PATH`
- containerd ne container banane ki koshish ki
- Process start karne ke liye us ko **runc** chahiye — jo hai hi nahi
- containerd manager hai, executor nahi — yehi farq hai jo log bhool jate hain
- Note: agar tumhari machine par runc pehle se install hai to Step 5 fail nahi hoga — ye theek hai, fail sirf fresh machine par dikhta hai

## Fix — runc Install Karo (v1.4.0)
```sh
curl -fsSLO https://github.com/opencontainers/runc/releases/download/v1.4.0/runc.amd64
sudo install -m 755 runc.amd64 /usr/local/bin/runc
runc --version
```

## Config — SystemdCgroup = true
> containerd aur kubelet dono ko ek hi cgroup driver chahiye — warna do manager ek hi container ko khinchte hain
```sh
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
grep 'SystemdCgroup' /etc/containerd/config.toml
```

## Step 6 — containerd Restart Karo
> Config badli hai — service restart karo aur check karo
```sh
sudo systemctl restart containerd
sudo systemctl status containerd --no-pager
```

## nerdctl + CNI Plugins — Network Ke Baghair Container Adhoora
> nerdctl `bridge` CNI plugin maangta hai — /opt/cni/bin mein kuch nahi to container network nahi mile ga
```sh
curl -fsSLO https://github.com/containerd/nerdctl/releases/download/v2.2.1/nerdctl-2.2.1-linux-amd64.tar.gz
tar xzvof nerdctl-2.2.1-linux-amd64.tar.gz
sudo install -m 755 nerdctl /usr/local/bin
```

## Step 7 — CNI Plugins Install Karo
> Plugins /opt/cni/bin mein aate hain — bridge, host-local, loopback sab yahin hote hain
```sh
curl -fsSLO https://github.com/containernetworking/plugins/releases/download/v1.9.0/cni-plugins-linux-amd64-v1.9.0.tgz
sudo mkdir -p /opt/cni/bin
sudo tar xzvof cni-plugins-linux-amd64-v1.9.0.tgz -C /opt/cni/bin
ls /opt/cni/bin | head
```

## Check Karo — podinfo Jawab De Raha Hai?
```sh
sudo nerdctl run -d --name podinfo -p 9898:9898 ghcr.io/stefanprodan/podinfo:latest
```

## Check — podinfo Ka IP Nikalo
> Container ka IP lo aur us par curl karo — jawab aana chahiye
```sh
PODINFO_IP=$(sudo nerdctl inspect --format '{{ .NetworkSettings.IPAddress }}' podinfo); curl -f "http://${PODINFO_IP}:9898"
```

## Check — Container List Dekho
> ps se dekho podinfo Up hai ya nahi
```sh
sudo nerdctl ps
sudo nerdctl inspect podinfo --format '{{ .NetworkSettings.IPAddress }}'
```

## Summary — Kya Seekha
- containerd manager hai — images, containers, snapshots sambhalta hai
- runc executor hai — asal process woh start karta hai, OCI runtime spec ke mutabiq
- containerd-shim beech mein hai — containerd aur runc ke darmiyan
- CNI plugins network dete hain — /opt/cni/bin ke baghair container ko IP nahi milta
- Dependency: containerd ke baghair kubelet Pod chala nahi sakta. runc ke baghair containerd container create nahi kar sakta

## Agli Lesson: kubelet — Lesson 04
> kubelet Pod ko containerd se kaise chalata hai — aur static Pods kya hote hain
