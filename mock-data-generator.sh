#!/bin/bash

# Mock Data Generator for MimirInsights Testing
# This script creates mock Mimir-like deployments and data sources for testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

NAMESPACE="mimir-test"
CLUSTER_NAME="mimirInsights-test"

echo -e "${BLUE}🎭 Setting up Mock Data for MimirInsights Testing${NC}"

# Function to print step headers
print_step() {
    echo -e "${YELLOW}===================================================${NC}"
    echo -e "${YELLOW}📦 $1${NC}"
    echo -e "${YELLOW}===================================================${NC}"
}

# Create mock Mimir namespace and deployments
print_step "Creating Mock Mimir Environment"

# Create mimir namespace
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Create mock Mimir components
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mimir-distributor
  namespace: ${NAMESPACE}
  labels:
    app: mimir
    component: distributor
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mimir
      component: distributor
  template:
    metadata:
      labels:
        app: mimir
        component: distributor
    spec:
      containers:
      - name: distributor
        image: grafana/mimir:latest
        ports:
        - containerPort: 9090
        - containerPort: 9095
        env:
        - name: MIMIR_COMPONENT
          value: "distributor"
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: mimir-distributor
  namespace: ${NAMESPACE}
  labels:
    app: mimir
    component: distributor
spec:
  ports:
  - name: http
    port: 9090
    targetPort: 9090
  - name: grpc
    port: 9095
    targetPort: 9095
  selector:
    app: mimir
    component: distributor
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mimir-ingester
  namespace: ${NAMESPACE}
  labels:
    app: mimir
    component: ingester
spec:
  replicas: 3
  selector:
    matchLabels:
      app: mimir
      component: ingester
  template:
    metadata:
      labels:
        app: mimir
        component: ingester
    spec:
      containers:
      - name: ingester
        image: grafana/mimir:latest
        ports:
        - containerPort: 9090
        - containerPort: 9095
        env:
        - name: MIMIR_COMPONENT
          value: "ingester"
        resources:
          requests:
            cpu: 200m
            memory: 1Gi
          limits:
            cpu: 1000m
            memory: 2Gi
---
apiVersion: v1
kind: Service
metadata:
  name: mimir-ingester
  namespace: ${NAMESPACE}
  labels:
    app: mimir
    component: ingester
spec:
  ports:
  - name: http
    port: 9090
    targetPort: 9090
  - name: grpc
    port: 9095
    targetPort: 9095
  selector:
    app: mimir
    component: ingester
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mimir-querier
  namespace: ${NAMESPACE}
  labels:
    app: mimir
    component: querier
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mimir
      component: querier
  template:
    metadata:
      labels:
        app: mimir
        component: querier
    spec:
      containers:
      - name: querier
        image: grafana/mimir:latest
        ports:
        - containerPort: 9090
        env:
        - name: MIMIR_COMPONENT
          value: "querier"
        resources:
          requests:
            cpu: 150m
            memory: 512Mi
          limits:
            cpu: 750m
            memory: 1Gi
---
apiVersion: v1
kind: Service
metadata:
  name: mimir-querier
  namespace: ${NAMESPACE}
  labels:
    app: mimir
    component: querier
spec:
  ports:
  - name: http
    port: 9090
    targetPort: 9090
  selector:
    app: mimir
    component: querier
EOF

# Create mock tenant namespaces
print_step "Creating Mock Tenant Namespaces"

for tenant in team-a team-b team-c; do
    kubectl create namespace ${tenant} --dry-run=client -o yaml | kubectl apply -f -
    
    # Add tenant label
    kubectl label namespace ${tenant} team=${tenant} --overwrite
    
    # Create mock Alloy agent in each tenant namespace
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alloy-agent
  namespace: ${tenant}
  labels:
    app: alloy
    tenant: ${tenant}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: alloy
      tenant: ${tenant}
  template:
    metadata:
      labels:
        app: alloy
        tenant: ${tenant}
    spec:
      containers:
      - name: alloy
        image: grafana/alloy:latest
        ports:
        - containerPort: 12345
        env:
        - name: TENANT_ID
          value: "${tenant}"
        resources:
          requests:
            cpu: 50m
            memory: 128Mi
          limits:
            cpu: 200m
            memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: alloy-agent
  namespace: ${tenant}
  labels:
    app: alloy
    tenant: ${tenant}
spec:
  ports:
  - name: http
    port: 12345
    targetPort: 12345
  selector:
    app: alloy
    tenant: ${tenant}
EOF
done

# Create mock Prometheus instance to simulate metrics
print_step "Creating Mock Prometheus for Metrics"

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: ${NAMESPACE}
  labels:
    app: prometheus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports:
        - containerPort: 9090
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus/'
        - '--web.console.libraries=/etc/prometheus/console_libraries'
        - '--web.console.templates=/etc/prometheus/consoles'
        - '--storage.tsdb.retention.time=200h'
        - '--web.enable-lifecycle'
        volumeMounts:
        - name: prometheus-config
          mountPath: /etc/prometheus/
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
      volumes:
      - name: prometheus-config
        configMap:
          name: prometheus-config
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: ${NAMESPACE}
  labels:
    app: prometheus
spec:
  ports:
  - name: http
    port: 9090
    targetPort: 9090
  selector:
    app: prometheus
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: ${NAMESPACE}
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      
    scrape_configs:
    - job_name: 'mimir-components'
      static_configs:
      - targets: ['mimir-distributor:9090', 'mimir-ingester:9090', 'mimir-querier:9090']
      
    - job_name: 'alloy-agents'
      kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
          - team-a
          - team-b
          - team-c
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: alloy-agent
EOF

# Wait for deployments to be ready
print_step "Waiting for Mock Services to be Ready"

echo -e "${BLUE}Waiting for Mimir components...${NC}"
kubectl wait --for=condition=available deployment/mimir-distributor -n ${NAMESPACE} --timeout=120s
kubectl wait --for=condition=available deployment/mimir-ingester -n ${NAMESPACE} --timeout=120s
kubectl wait --for=condition=available deployment/mimir-querier -n ${NAMESPACE} --timeout=120s

echo -e "${BLUE}Waiting for Prometheus...${NC}"
kubectl wait --for=condition=available deployment/prometheus -n ${NAMESPACE} --timeout=120s

echo -e "${BLUE}Waiting for Alloy agents...${NC}"
for tenant in team-a team-b team-c; do
    kubectl wait --for=condition=available deployment/alloy-agent -n ${tenant} --timeout=120s
done

# Create test script for API endpoints
print_step "Creating Test Scripts"

cat > test-api.sh <<'EOF'
#!/bin/bash

# Test script for MimirInsights API endpoints
set -e

API_BASE="http://localhost:8081/api"

echo "🧪 Testing MimirInsights API Endpoints"
echo "======================================"

# Health check
echo "1. Testing health endpoint..."
curl -s "${API_BASE}/health" | jq . || echo "Health endpoint not ready yet"

# Discovery endpoint
echo "2. Testing discovery endpoint..."
curl -s "${API_BASE}/discovery" | jq . || echo "Discovery endpoint not ready yet"

# Tenants endpoint
echo "3. Testing tenants endpoint..."
curl -s "${API_BASE}/tenants" | jq . || echo "Tenants endpoint not ready yet"

# Metrics endpoint
echo "4. Testing metrics endpoint..."
curl -s "${API_BASE}/metrics" | head -20 || echo "Metrics endpoint not ready yet"

# Limits endpoint
echo "5. Testing limits endpoint..."
curl -s "${API_BASE}/limits" | jq . || echo "Limits endpoint not ready yet"

echo "✅ API tests completed!"
EOF

chmod +x test-api.sh

# Create data simulation script
cat > simulate-data.sh <<'EOF'
#!/bin/bash

# Simulate metric data for testing
set -e

PROMETHEUS_URL="http://localhost:9090"
NAMESPACE="mimir-test"

echo "📊 Simulating Metric Data"
echo "========================"

# Port forward to Prometheus for data simulation
kubectl port-forward -n ${NAMESPACE} service/prometheus 9090:9090 &
PROMETHEUS_PID=$!

sleep 5

# Generate some mock metrics by calling the services
echo "Generating mock traffic..."
for i in {1..10}; do
    kubectl exec -n ${NAMESPACE} deployment/mimir-distributor -- wget -q -O- http://localhost:9090/ready >/dev/null 2>&1 || true
    kubectl exec -n ${NAMESPACE} deployment/mimir-ingester -- wget -q -O- http://localhost:9090/ready >/dev/null 2>&1 || true
    kubectl exec -n ${NAMESPACE} deployment/mimir-querier -- wget -q -O- http://localhost:9090/ready >/dev/null 2>&1 || true
    sleep 2
done

# Stop port forward
kill $PROMETHEUS_PID 2>/dev/null || true

echo "✅ Mock data simulation completed!"
EOF

chmod +x simulate-data.sh

# Display status
print_step "Mock Environment Status"

echo -e "${BLUE}Mimir components:${NC}"
kubectl get pods -n ${NAMESPACE} -l app=mimir

echo -e "${BLUE}Tenant namespaces:${NC}"
kubectl get namespaces -l team

echo -e "${BLUE}Alloy agents:${NC}"
for tenant in team-a team-b team-c; do
    echo "  ${tenant}:"
    kubectl get pods -n ${tenant} -l app=alloy
done

echo -e "${GREEN}✅ Mock environment setup completed!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "1. Deploy MimirInsights: ${GREEN}./deploy.sh${NC}"
echo -e "2. Run port forwarding: ${GREEN}./port-forward.sh${NC}"
echo -e "3. Test API endpoints: ${GREEN}./test-api.sh${NC}"
echo -e "4. Simulate data: ${GREEN}./simulate-data.sh${NC}"
echo ""
echo -e "${BLUE}🎉 Mock data environment is ready for testing!${NC}"
EOF