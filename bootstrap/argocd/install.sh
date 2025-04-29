#!/bin/bash

# helm is required to install argocd

NAMESPACE="argocd"

NAME="argocd"

REPO="https://argoproj.github.io/argo-helm"

CHART="argo-cd"
VALUES_FILE_PATH="values.yaml"

if [[ ! $1 ]];then
  helm upgrade --install $NAME --repo=$REPO $CHART --values $VALUES_FILE_PATH --create-namespace -n $NAMESPACE --wait
  exit 0
fi
