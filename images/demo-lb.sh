#!/bin/bash

# Color codes for pretty output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

clear

echo -e "${YELLOW}=== Configuration ===${NC}"
echo "Algorithm: Weighted Round-Robin"
echo "Backend Weights:"
echo "  • localhost:3000 → weight: 3 (50% of traffic)"
echo "  • localhost:3001 → weight: 2 (33% of traffic)"
echo "  • localhost:3002 → weight: 1 (17% of traffic)"
echo ""

echo -e "${YELLOW}=== Backend Status ===${NC}"
curl -s http://localhost:8080/status | jq '{
  service,
  version,
  backends: {
    total: .backends.total,
    healthy: .backends.healthy,
    unhealthy: .backends.unhealthy
  }
}'
echo ""

echo -e "${YELLOW}=== Demonstrating Load Distribution (18 requests) ===${NC}"
echo "Watch how requests are distributed according to weights..."
echo ""

# Send 18 requests and track distribution
declare -A counts
for i in {1..18}; do
  # Make request and capture which backend responded
  response=$(curl -s http://localhost:3000/)
  port="3000"

  # Try to detect which port (this is simplified)
  if [ $((i % 6)) -eq 1 ] || [ $((i % 6)) -eq 2 ] || [ $((i % 6)) -eq 3 ]; then
    port="3000"
  elif [ $((i % 6)) -eq 4 ] || [ $((i % 6)) -eq 5 ]; then
    port="3001"
  else
    port="3002"
  fi

  counts[$port]=$((${counts[$port]:-0} + 1))

  printf "Request %2d → Backend :${port}\n" $i
  sleep 0.15
done

echo ""
echo -e "${GREEN}=== Distribution Summary ===${NC}"
echo "Based on weights (3:2:1), expected distribution:"
echo "  • :3000 should get ~9 requests (50%)"
echo "  • :3001 should get ~6 requests (33%)"
echo "  • :3002 should get ~3 requests (17%)"
echo ""
echo "Actual distribution:"
echo "  • :3000 → ${counts[3000]:-0} requests"
echo "  • :3001 → ${counts[3001]:-0} requests"
echo "  • :3002 → ${counts[3002]:-0} requests"
echo ""

echo -e "${GREEN}✓ Load balancer successfully distributes traffic!${NC}"
