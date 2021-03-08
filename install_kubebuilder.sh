#!/bin/bash
os=$(go env GOOS)
arch=$(go env GOARCH)
version=$(curl -s https://api.github.com/repos/kubernetes-sigs/kubebuilder/releases/latest | grep "tag_name" | cut -d '"' -f 4 | cut -c 2-)

echo "install kubebuilder version", $version

# download latest release kubebuilder and extract it to tmp
curl -L https://go.kubebuilder.io/dl/${version}/${os}/${arch} | tar -xz -C /tmp/

# move to a long-term location and put it on your path
sudo mv /tmp/kubebuilder_${version}_${os}_${arch} /usr/local/kubebuilder

export PATH=$PATH:/usr/local/kubebuilder/bin