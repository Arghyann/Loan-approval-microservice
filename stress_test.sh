#!/usr/bin/env bash
set -e

NAMESPACE="loan-app"
JOB_NAME="c-powered-wrk-swarm"
ATTACK_DURATION="30s"
THREADS="4"
CONCURRENCY="1000"
PARALLEL_PODS=24
BACKEND_REPLICAS=40

echo "================================================================="
echo " 🔥 MAXIMUM AGGRESSION (24 C MISSILES / 24,000 CONNECTIONS) 🔥"
echo "================================================================="
echo ""

# 1. Clean up previous test runs
echo "🧹 [1/4] Cleaning up previous test runs..."
kubectl delete job $JOB_NAME -n $NAMESPACE 2>/dev/null || true
echo "✅ Cleaned."
echo ""

# 2. Scale Backend Fleet to 40 Pods
echo "🛡️  [2/4] Scaling Backend Fleet to ${BACKEND_REPLICAS} pods across private-nodes..."
kubectl scale deployment iam-service -n $NAMESPACE --replicas=$BACKEND_REPLICAS >/dev/null
kubectl rollout status deployment/iam-service -n $NAMESPACE --timeout=60s >/dev/null
echo "✅ ${BACKEND_REPLICAS} Backend Pods are active on private-nodes."
echo ""

# 3. Launch 24 C Wrk Missiles
echo "🚀 [3/4] Launching ${PARALLEL_PODS} C-Powered (wrk) Attack Missiles..."
echo "       • Engine:                C (epoll non-blocking event-loop)"
echo "       • Attack Duration:       ${ATTACK_DURATION}"
echo "       • Threads Per Pod:       ${THREADS} C threads"
echo "       • Concurrency Per Pod:   ${CONCURRENCY} connections"
echo "       • Total Cluster Sockets: $((PARALLEL_PODS * CONCURRENCY)) concurrent streams!"
echo "       • Target URL:            http://iam-service.loan-app.svc.cluster.local:8080/health"
echo "       • Attacker Nodes:        attack-nodes (8 Dedicated servers)"
echo ""

cat << YAML | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: ${NAMESPACE}
spec:
  parallelism: ${PARALLEL_PODS}
  completions: ${PARALLEL_PODS}
  template:
    spec:
      restartPolicy: Never
      nodeSelector:
        eks.amazonaws.com/nodegroup: attack-nodes
      containers:
      - name: c-missile
        image: skandyla/wrk
        command: ["wrk"]
        args:
        - "-t${THREADS}"
        - "-c${CONCURRENCY}"
        - "-d${ATTACK_DURATION}"
        - "--latency"
        - "http://iam-service.loan-app.svc.cluster.local:8080/health"
YAML

echo ""
echo "⏳ [4/4] Attack in progress for ${ATTACK_DURATION}... Watch Grafana (http://localhost:3000/d/loan-stress-center)"
echo "-----------------------------------------------------------------"

# Wait for completion
START_TIME=$(date +%s)
kubectl wait --for=condition=complete job/${JOB_NAME} -n ${NAMESPACE} --timeout=150s >/dev/null
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "================================================================="
echo " 🏆 MAXIMUM AGGRESSION BENCHMARK COMPLETED IN ${DURATION} SECONDS! 🏆"
echo "================================================================="
echo ""

# Aggregate stats from all 24 C wrk pods
echo "📊 AGGREGATE PERFORMANCE SCORECARD:"
echo "-----------------------------------------------------------------"

kubectl logs -n ${NAMESPACE} -l job-name=${JOB_NAME} | awk -v DURATION=$DURATION '
  /requests in/ {
    total_req += $1
  }
  /Requests\/sec:/ {
    total_rps += $2
    pods++
  }
  END {
    printf "  • Total Attack Pods in C:    %d pods\n", pods
    printf "  • Total Requests Processed:  %d requests\n", total_req
    printf "  • Total Duration:            %d seconds\n", DURATION
    printf "  • 🔥 TOTAL COMBINED THROUGHPUT: \033[1;32m%.2f Requests/Sec\033[0m\n", total_rps
  }
'

echo "-----------------------------------------------------------------"
echo "✨ Benchmark finished! Open Grafana to view the peak!"
