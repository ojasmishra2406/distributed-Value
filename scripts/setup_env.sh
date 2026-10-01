#!/bin/bash
set -e

echo "Updating package lists..."
sudo apt-get update -y

echo "Installing prerequisites..."
sudo apt-get install -y curl wget git build-essential software-properties-common

echo "Installing Python 3.12..."
sudo add-apt-repository -y ppa:deadsnakes/ppa
sudo apt-get update -y
sudo apt-get install -y python3.12 python3.12-venv python3.12-dev

echo "Installing Docker..."
sudo apt-get install -y docker.io
sudo systemctl enable docker
sudo systemctl start docker
sudo usermod -aG docker $USER

echo "Installing Go 1.27..."
wget https://go.dev/dl/go1.27.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.27.0.linux-amd64.tar.gz
rm go1.27.0.linux-amd64.tar.gz

if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
    echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bashrc
    echo "export GOPATH=\$HOME/go" >> ~/.bashrc
    echo "export PATH=\$PATH:\$GOPATH/bin" >> ~/.bashrc
fi

echo "Environment bootstrap complete. Please run 'source ~/.bashrc'."
