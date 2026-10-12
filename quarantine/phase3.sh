#!/bin/bash
PASS="1234"
PROJ="/mnt/c/Users/mishr/OneDrive/Documents/Distributed-values"
export PATH="/usr/local/go/bin:$PATH"

echo "=== TASK 3.1 - ENVIRONMENT CHECK ==="
which tc iptables nc
tc -Version 2>&1
iptables --version 2>&1
uname -r

echo ""
echo "=== TASK 3.2 - BOOT 3-NODE CLUSTER ==="
rm -rf /tmp/d1 /tmp/d2 /tmp/d3
mkdir -p /tmp/d1 /tmp/d2 /tmp/d3

$PROJ/node_linux -port=50051 -dir=/tmp/d1 &
N1=$!
echo "Node 1 PID=$N1 on :50051"

$PROJ/node_linux -port=50052 -dir=/tmp/d2 &
N2=$!
echo "Node 2 PID=$N2 on :50052"

$PROJ/node_linux -port=50053 -dir=/tmp/d3 &
N3=$!
echo "Node 3 PID=$N3 on :50053"

sleep 2
nc -zv 127.0.0.1 50051 2>&1 && echo "50051 OPEN"
nc -zv 127.0.0.1 50052 2>&1 && echo "50052 OPEN"
nc -zv 127.0.0.1 50053 2>&1 && echo "50053 OPEN"

echo ""
echo "=== TASK 3.2 - SANITY PUT/GET ==="
$PROJ/script_linux

echo ""
echo "=== TASK 3.3 - INDUCE PARTITION: DROP port 50053 ==="
echo "$PASS" | sudo -S iptables -A OUTPUT -p tcp --dport 50053 -j DROP 2>&1
echo "$PASS" | sudo -S iptables -A INPUT  -p tcp --sport 50053 -j DROP 2>&1
echo "Rules applied:"
echo "$PASS" | sudo -S iptables -L -n --line-numbers 2>&1 | grep -E "50053|^Chain"

echo ""
echo "=== TASK 3.3 - CONFIRM PARTITION IS REAL ==="
timeout 3 nc -zv 127.0.0.1 50053 2>&1 || echo "CONFIRMED: 50053 unreachable (iptables DROP active)"

echo ""
echo "=== TASK 3.4 - START CLIENT (keeps detector alive for 20s) ==="
$PROJ/phase3test_linux &
CLIENT_PID=$!
echo "Client PID=$CLIENT_PID started"

# Let the partition sit for 7s (enough for detector to miss 3×2s beats = mark dead, hint queued)
sleep 7

echo ""
echo "=== TASK 3.5 - HEAL PARTITION at t=7s ==="
echo "$PASS" | sudo -S iptables -D OUTPUT -p tcp --dport 50053 -j DROP 2>&1
echo "$PASS" | sudo -S iptables -D INPUT  -p tcp --sport 50053 -j DROP 2>&1
echo "Rules after removal:"
echo "$PASS" | sudo -S iptables -L -n --line-numbers 2>&1 | grep -E "50053|^Chain"

echo ""
echo "=== TASK 3.5 - CONFIRM CONNECTIVITY RESTORED ==="
timeout 3 nc -zv 127.0.0.1 50053 2>&1 && echo "CONFIRMED: 50053 reachable again"

echo "Waiting for client to finish and handoff to replay..."
wait $CLIENT_PID
echo "Client exited."

echo ""
echo "=== TASK 3.5 - CHECK NODE 3 WAL FOR HANDOFF ==="
echo "Raw bytes in /tmp/d3/active.wal containing partition_key:"
if grep -aq "partition_key" /tmp/d3/active.wal 2>/dev/null; then
    echo "SUCCESS: partition_key found in Node 3's WAL — hinted handoff replayed!"
    ls -la /tmp/d3/
else
    echo "FAIL: partition_key NOT in /tmp/d3/active.wal"
    ls -la /tmp/d3/
fi

echo ""
echo "=== CLEANUP ==="
kill $N1 $N2 $N3 2>/dev/null
echo "Phase 3 done."
