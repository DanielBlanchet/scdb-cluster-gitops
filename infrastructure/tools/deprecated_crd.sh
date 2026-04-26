#!/usr/bin/env bash

contexts=$(kubectl config get-contexts -o name)

for ctx in $contexts; do
  echo "=============================="
  echo "Cluster: $ctx"
  echo "=============================="

  kubectl --context "$ctx" get crd -o json | jq -r '
  [.items[].spec.versions[] | select(.deprecated==true)] | length as $count |
  if $count == 0 then
    "No deprecated CRDs ✅"
  else
    "Deprecated CRDs: \($count)"
  end
  '

  kubectl --context "$ctx" get crd -o json | jq -r '
  .items[] as $crd |
  $crd.spec.versions[] |
  select(.deprecated == true) |
  "\($crd.metadata.name) \(.name)"
  '

  echo ""
done
