#!/bin/bash
set -e

echo "=== 1. Provisioning AWS Infrastructure ==="
cd terraform
terraform init
terraform apply -auto-approve
cd ..

echo "=== 2. Authenticating to ECR ==="
aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin <YOUR_ACCOUNT_ID>.dkr.ecr.us-west-2.amazonaws.com

echo "=== 3. Building & Pushing Docker Image ==="
docker build -t distributed-kv:latest .
docker tag distributed-kv:latest <YOUR_ACCOUNT_ID>.dkr.ecr.us-west-2.amazonaws.com/distributed-kv:latest
docker push <YOUR_ACCOUNT_ID>.dkr.ecr.us-west-2.amazonaws.com/distributed-kv:latest

echo "=== 4. Deploying to Kubernetes ==="
kubectl apply -f k8s/aws-ebs-storageclass.yaml
kubectl apply -f k8s/node-statefulset.yaml
kubectl apply -f k8s/gateway-deployment.yaml

echo "Deployment Complete!"
