# Lesson 03 — containerd

## containerd — Kubernetes ka Engine Room
> Aaj hum containerd ko haath se install karenge — aur samjhenge ke container asal mein chalta kaise hai.
- Aaj ka plan: install → config → service → ctr FAIL → runc → nerdctl → CNI → podinfo live
- Har step pehle haath se hoga — koi kubeadm, koi shortcut nahi
- End mein tum khud bata sako ge: containerd kya karta hai, aur runc kya karta hai

## containerd manager hai — executor nahi
> containerd decide karta hai **KYA** karna hai — kaunsa container, kaunsi image, kaunsa network. runc karta hai **KAISE** — process kaise banega, isolation kaise hogi.
- containerd **runtime nahi hai** — woh OCI runtime (runc) ko kaam deta hai
- Kubernetes mein containerd ke 5 kaam: lifecycle, image, storage, network, runtime management
- kubelet containerd se CRI (Container Runtime Interface) se baat karta hai
- `nerdctl` containerd ka CLI hai — Docker jaisa experience, containerd ke upar
```text
kubelet
  │ CRI
  ▼
containerd ── image pull + store
  │
  ▼
containerd-shim
  │
  ▼
runc ──▶ container

CNI plugins (/opt/cni/bin) — network
```

## Is lesson ke baad tum ye kar sako ge
> Ye lesson khatam ho to tum containerd ko haath se khada kar sako ge — kubeadm ke baghair.
- containerd install aur configure kar sako ge
- Samajh sako ge ke containerd, shim aur runc mein farq kya hai
- `ctr` aur `nerdctl` se container chala sako ge
- CNI plugins kyun chahiye, ye bata sako ge

## Pehle FAIL dikhao — containerd akela
> containerd install karo, runc ke baghair container chalao — fail hoga. Ye fail hi is lesson ka sab se bara sabak hai.
- Pehle tarball download karo — install abhi nahi, sirf file lo
- Yehi file agle step mein install hogi
! Tarball maujood: containerd-2.2.1-linux-amd64.tar.gz
```sh
# tarball download karo
curl -fsSLO https://github.com/containerd/containerd/releases/download/v2.2.1/containerd-2.2.1-linux-amd64.tar.gz
```

## Install: pehle iptables, phir binaries
> CNI bridge plugin ko NAT ke liye iptables chahiye — ye step miss hua to aakhir mein `nerdctl run` fail hoga.
- `iptables` host tool hai — containerd ka hissa nahi, lekin CNI ke baghair kaam nahi
- Tarball se binaries nikalo aur `/usr/local/bin` mein daalo
! bin/ folder nikla — containerd, ctr, containerd-shim-runc-v2
```sh
# host tool — CNI bridge ke liye zaroori
sudo apt install -y iptables
# tarball se binaries nikalo
tar xzvof containerd-2.2.1-linux-amd64.tar.gz
# /usr/local/bin mein daalo
sudo install -m 755 bin/* /usr/local/bin
```

## Config: default file banao, flood se bacho
> `containerd config default` poora TOML dump karta hai — is ko file mein daalo, screen par nahi.
- Config yahan rehti hai: `/etc/containerd/config.toml`
- Isi file mein aage SystemdCgroup = true karenge
! `wc -l` bataye ga ke config.toml ban gayi
```sh
# config folder banao
sudo mkdir -p /etc/containerd
# default config file mein likho — screen par mat dikhao
containerd config default | sudo tee /etc/containerd/config.toml > /dev/null
# lines gin lo — file bani ya nahi
wc -l /etc/containerd/config.toml
```

## Tarball mein service file NAHI hoti
> Tarball mein binaries hain, service file nahi. `systemctl` containerd ko dhoond hi nahi sakta jab tak unit file `/etc/systemd/system` mein na ho.
- Yehi woh step hai jo sab se zyada log bhool jate hain — phir kehte hain "service nahi mil rahi"
- Unit file containerd ke GitHub se lo — official, version ke hisab se
! /etc/systemd/system/containerd.service maujood
```sh
# official unit file download karo
sudo wget -P /etc/systemd/system "https://raw.githubusercontent.com/containerd/containerd/v2.2.1/containerd.service"
```

## Service start — containerd zinda hai
> Mubarak ho, containerd chal raha hai — lekin abhi is se container nahi chalega, kyunki OCI runtime abhi nahi hai. Yehi hum agle step mein dekhte hain.
- `daemon-reload` — systemd ko nayi unit file ka pata do
- `enable --now` — boot par bhi chale, aur abhi bhi chale
! Active: active (running)
```sh
# systemd ko unit file ka pata do
sudo systemctl daemon-reload
# enable + start — ek hi command
sudo systemctl enable --now containerd
# check karo — running hona chahiye
sudo systemctl status containerd --no-pager
```

## Pehla container — FAIL hona chahiye
> `ctr` image khud pull nahi karta — pehle pull karo, phir run karo. Aur run fail hoga — yehi dikhana hai.
- `ctr` containerd ka apna CLI hai — low-level, Docker CLI jaisa nahi
- Image pull ho jaye gi — process nahi banega
! exec: "runc": executable file not found in $PATH
```sh
# pehle image pull karo — ctr run khud pull NAHI karta
sudo ctr images pull ghcr.io/sagikazarmark/docker-hello-world:latest
# ab run karo — FAIL hoga
sudo ctr run --rm ghcr.io/sagikazarmark/docker-hello-world:latest hello
```

## Fail ka matlab: runc missing hai
> containerd ne image pull kar li — lekin process banane ke liye runc chahiye, jo hai hi nahi.
- containerd **manager** hai, executor nahi — yehi farq hai jo log bhool jate hain
- Beech mein **containerd-shim** hota hai — containerd aur runc ke darmiyan halka sa bridge
- Shim container ko chalata rehta hai agar containerd crash bhi ho jaye
- Note: agar tumhari machine par runc pehle se install hai to Step 5 fail nahi hoga — ye theek hai, fail sirf fresh machine par dikhta hai

## Fix: runc install karo — executor aa gaya
> runc OCI runtime hai — asal process wohi start karta hai, OCI runtime spec ke mutabiq.
- **OCI Runtime Specification** — standard batata hai ke low-level par container kaise banta hai
- Is standard ki wajah se containerd kisi bhi OCI-compliant runtime ke sath kaam kar sakta hai
- Doosre OCI runtimes: `crun` (behtar performance) aur `gVisor` (VM jaisi isolation) — reference runtime `runc` hai
! runc version 1.4.0
```sh
# runc binary download karo
curl -fsSLO https://github.com/opencontainers/runc/releases/download/v1.4.0/runc.amd64
# /usr/local/bin mein daalo
sudo install -m 755 runc.amd64 /usr/local/bin/runc
# version check karo
runc --version
```

## Ek hi cgroup driver — warna do manager ladenge
> systemd aur runtime alag cgroup tree manage karen to memory/CPU pressure mein instability aati hai. systemd init hai — dono ko ek hi tree use karna chahiye.
- cgroups CPU, memory, I/O ko limit aur isolate karte hain
- containerd aur kubelet dono ko ek hi cgroup driver chahiye — warna do manager ek hi container ko khinchte hain
! SystemdCgroup = true
```sh
# config mein SystemdCgroup true karo
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
# verify karo
grep 'SystemdCgroup' /etc/containerd/config.toml
```

## Restart karo — ab wohi command chalegi
> Pehle fail hua tha kyunki runc nahi tha. Ab runc hai, to container chal gaya. Yehi farq hai high-level aur low-level runtime ka.
- Config badli hai — service restart karo aur check karo
- Ab wohi `ctr run` dobara chalao ge to chalega — agle slide mein nahi, khud try karo
! Active: active (running)
```sh
# config change ke baad restart zaroori
sudo systemctl restart containerd
# check karo
sudo systemctl status containerd --no-pager
```

## ctr low-level hai — rozana ke liye nerdctl
> `ctr` containerd ka apna CLI hai — low-level, dostana nahi. `nerdctl` Docker jaisa interface deta hai, containerd ke upar.
- `nerdctl` ka matlab: contaiNERD CTL — Docker CLI jaisa experience
- `ctr` ke muqable mein `nerdctl` image pull aur network setup **khud** kar leta hai
- Farq yaad rakho: dono containerd se baat karte hain, lekin alag namespace mein containers dekhte hain — ye kubelet lesson mein aaye ga

## nerdctl install — Docker jaisa CLI
> nerdctl containerd ke upar baithta hai — is ke baghair tum `ctr` se hi kaam chala rahe hote.
- Tarball se binary nikalo, `/usr/local/bin` mein daalo
- Is ke baad `nerdctl run` Docker jaisa lage ga — lekin andar containerd hi hai
! nerdctl version 2.2.1
```sh
# nerdctl tarball download karo
curl -fsSLO https://github.com/containerd/nerdctl/releases/download/v2.2.1/nerdctl-2.2.1-linux-amd64.tar.gz
# extract karo
tar xzvof nerdctl-2.2.1-linux-amd64.tar.gz
# /usr/local/bin mein daalo
sudo install -m 755 nerdctl /usr/local/bin
```

## Network: CNI plugins ke baghair IP nahi milta
> containerd process OCI runtime ko deta hai, waise hi network CNI ko deta hai.
- **CNI (Container Network Interface)** — specification: network namespace kaise banaye, IP kaise de, container aur host ke beech connectivity kaise kare
- Plugins yahan hote hain: `/opt/cni/bin` — ye convention hai, runtime aur network add-on dono yahin dekhte hain
- `nerdctl` by default bridge network banana chahta hai — bridge plugin na ho to fail hota hai
! List mein bridge, host-local, loopback nazar aayen
```sh
# CNI plugins tarball download karo
curl -fsSLO https://github.com/containernetworking/plugins/releases/download/v1.9.0/cni-plugins-linux-amd64-v1.9.0.tgz
# folder banao
sudo mkdir -p /opt/cni/bin
# /opt/cni/bin mein extract karo
sudo tar xzvof cni-plugins-linux-amd64-v1.9.0.tgz -C /opt/cni/bin
# check karo — plugins aaye ya nahi
ls /opt/cni/bin | head
```

## Proof: podinfo jawab de raha hai
> Ab sab kuch jagah par hai — containerd, runc, nerdctl, CNI. Podinfo chalao aur dekho.
- `-d` — detached, background mein chale
- `-p 9898:9898` — host port 9898, container port 9898
> Agar iptables na hota to yahi command is error par fail hoti: "failed to call cni.Setup: plugin type=\"bridge\" failed (add): failed to locate iptables"
> Agar podinfo pehle se chal raha ho to port 9898 busy hoga ("bind for :9898 failed: port is already allocated") — pehle `sudo nerdctl rm -f podinfo` chala lo, phir ye command
! Container Up — 0.0.0.0:9898->9898/tcp
```sh
# podinfo chalao — detached, port map ke sath
sudo nerdctl run -d --name podinfo -p 9898:9898 ghcr.io/stefanprodan/podinfo:latest
```

## Container ka IP nikalo — curl karo
> Container ka IP lo aur us par curl karo — jawab aana chahiye. Yehi proof hai ke network kaam kar raha hai.
- `nerdctl inspect` se IP nikalo, variable mein daalo
- Phir us IP par curl karo — podinfo ka JSON jawab aaye ga
! {"version": "6.17.0"...} — JSON jawab
```sh
# IP nikalo aur curl karo — ek hi command
PODINFO_IP=$(sudo nerdctl inspect --format '{{ .NetworkSettings.IPAddress }}' podinfo); curl -f "http://${PODINFO_IP}:9898"
```

## List dekho — podinfo Up hai
> `ps` se dekho podinfo Up hai ya nahi — yehi aakhri check hai.
- STATUS mein Up hona chahiye — agar Exited ho to kuch ghalat hai
- IP dobara bhi dekh sakte ho — wahi IP jo pichle slide mein mila
! STATUS: Up
```sh
# containers ki list dekho
sudo nerdctl ps
# podinfo ka IP dobara dekho
sudo nerdctl inspect podinfo --format '{{ .NetworkSettings.IPAddress }}'
```

## Kya seekha — manager, executor, network
> Aaj tum ne containerd ko haath se khada kiya — aur har piece ka kaam alag alag dekha.
- containerd **manager** hai — images, containers, snapshots sambhalta hai
- runc **executor** hai — asal process woh start karta hai, OCI runtime spec ke mutabiq
- containerd-shim beech mein hai — containerd aur runc ke darmiyan
- CNI plugins network dete hain — /opt/cni/bin ke baghair container ko IP nahi milta
- Dependency yaad rakho: containerd ke baghair kubelet Pod chala nahi sakta. runc ke baghair containerd container create nahi kar sakta.

## Agli lesson: kubelet — Pod kaise chalta hai
> containerd akela container chala sakta hai — lekin Pod nahi bana sakta. Yehi agle lesson ki taraf ishara hai.
- kubelet Pod ko containerd se kaise chalata hai — aur static Pods kya hote hain
- Agli baar: kubelet install, TLS bootstrapping, aur pehla Pod
- Parhne ke liye: containerd docs — containerd.io · OCI runtime spec — github.com/opencontainers/runtime-spec · CNI — github.com/containernetworking/cni
