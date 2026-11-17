#!/usr/bin/env bash
# Usage: [GO_VERSION=1.18] scripts/smoke.sh
set -euo pipefail

cd "$(dirname "$0")/.."

image="task-api:smoke-go${GO_VERSION:-default}"
container="task-api-smoke-$$"

trap 'docker rm -f "$container" >/dev/null 2>&1 || true' EXIT

docker build ${GO_VERSION:+--build-arg "GO_VERSION=$GO_VERSION"} -t "$image" .
docker run -d --name "$container" -p 127.0.0.1::8080 "$image" >/dev/null
base="http://$(docker port "$container" 8080/tcp | head -n 1)"

for _ in $(seq 1 50); do
	curl -s -o /dev/null "$base/tasks" && break
	sleep 0.2
done

fail=0
check() {
	local want=$1 got
	shift
	got=$(curl -s -o /dev/null -w '%{http_code}' "$@" || true)
	if [ "$got" = "$want" ]; then
		echo "ok   $want $*"
	else
		echo "FAIL $got, want $want: $*"
		fail=1
	fi
}

check 201 -X POST "$base/tasks" -d '{"name":"smoke"}'
check 200 "$base/tasks"
check 200 -X PUT "$base/tasks/1" -d '{"name":"smoke","status":1,"version":1}'
check 409 -X PUT "$base/tasks/1" -d '{"name":"smoke","status":0,"version":1}'
check 204 -X DELETE "$base/tasks/1"
check 404 "$base/tasks/1"

docker stop -t 10 "$container" >/dev/null
code=$(docker inspect -f '{{.State.ExitCode}}' "$container")
if [ "$code" = 0 ] && docker logs "$container" 2>&1 | grep -q "shutting down"; then
	echo "ok   graceful shutdown"
else
	echo "FAIL graceful shutdown: exit code $code"
	docker logs "$container" 2>&1 | tail -n 5
	fail=1
fi

exit "$fail"
