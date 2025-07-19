#!/bin/bash

# ==============================================================================
# Update Values Script for MimirInsights
# Updates values-production-final.yaml with the latest timestamp tag from CI/CD
# ==============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
VALUES_FILE="deployments/helm-chart/values-production-final.yaml"
REGISTRY="ghcr.io/akshaydubey29"

# Check if timestamp is provided as argument
if [ $# -eq 0 ]; then
    print_error "Usage: $0 <timestamp>"
    print_error "Example: $0 20250719-141012"
    exit 1
fi

TIMESTAMP=$1

# Validate timestamp format (YYYYMMDD-HHMMSS)
if [[ ! $TIMESTAMP =~ ^[0-9]{8}-[0-9]{6}$ ]]; then
    print_error "Invalid timestamp format. Expected: YYYYMMDD-HHMMSS"
    print_error "Example: 20250719-141012"
    exit 1
fi

print_status "Updating values file with timestamp: $TIMESTAMP"

# Check if values file exists
if [ ! -f "$VALUES_FILE" ]; then
    print_error "Values file not found: $VALUES_FILE"
    exit 1
fi

# Create backup
BACKUP_FILE="${VALUES_FILE}.backup.$(date +%Y%m%d-%H%M%S)"
cp "$VALUES_FILE" "$BACKUP_FILE"
print_status "Created backup: $BACKUP_FILE"

# Update backend image
print_status "Updating backend image..."
sed -i.bak "s|image: ghcr.io/akshaydubey29/mimir-insights-backend-[0-9]\{8\}-[0-9]\{6\}|image: ghcr.io/akshaydubey29/mimir-insights-backend-$TIMESTAMP|g" "$VALUES_FILE"

# Update frontend image
print_status "Updating frontend image..."
sed -i.bak "s|image: ghcr.io/akshaydubey29/mimir-insights-frontend-[0-9]\{8\}-[0-9]\{6\}|image: ghcr.io/akshaydubey29/mimir-insights-frontend-$TIMESTAMP|g" "$VALUES_FILE"

# Remove temporary backup files
rm -f "${VALUES_FILE}.bak"

# Verify the changes
print_status "Verifying changes..."

BACKEND_IMAGE=$(grep -A 1 "backend:" "$VALUES_FILE" | grep "image:" | sed 's/.*image: //')
FRONTEND_IMAGE=$(grep -A 1 "frontend:" "$VALUES_FILE" | grep "image:" | sed 's/.*image: //')

print_success "Values file updated successfully!"
echo ""
echo "📦 Updated Images:"
echo "  Backend:  $BACKEND_IMAGE"
echo "  Frontend: $FRONTEND_IMAGE"
echo ""
echo "🚀 Next steps:"
echo "1. Deploy to your local kind cluster:"
echo "   helm upgrade --install mimir-insights ./deployments/helm-chart \\"
echo "     --namespace mimir-insights \\"
echo "     --values ./deployments/helm-chart/values-production-final.yaml \\"
echo "     --wait"
echo ""
echo "2. Setup port forwarding:"
echo "   kubectl port-forward -n mimir-insights svc/mimir-insights-backend 8080:8080 &"
echo "   kubectl port-forward -n mimir-insights svc/mimir-insights-frontend 8081:80 &"
echo ""
echo "3. Access the application:"
echo "   Frontend: http://localhost:8081"
echo "   Backend:  http://localhost:8080/api/tenants" 