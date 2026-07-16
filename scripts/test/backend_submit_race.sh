#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

PGHOST="${BEAST_TEST_PGHOST:-localhost}"
PGPORT="${BEAST_TEST_PGPORT:-55543}"
PGUSER="${BEAST_TEST_PGUSER:-beasttest}"
PGPASSWORD="${BEAST_TEST_PGPASSWORD:-beasttest}"
PGDATABASE="${BEAST_TEST_PGDATABASE:-beast_backend_test}"
REDIS_HOST="${BEAST_TEST_REDIS_HOST:-localhost}"
REDIS_PORT="${BEAST_TEST_REDIS_PORT:-56380}"
SERVER_PORT="${BEAST_TEST_SERVER_PORT:-5505}"
TEST_HOME="${BEAST_TEST_HOME:-/tmp/beast-backend-submit-race}"
LOG_FILE="$TEST_HOME/beast-api.log"
BASE_URL="http://localhost:$SERVER_PORT"

export PGPASSWORD

cleanup() {
	local status=$?
	if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
		kill "$SERVER_PID" 2>/dev/null || true
		wait "$SERVER_PID" 2>/dev/null || true
	fi
	exit "$status"
}
trap cleanup EXIT

psql_root() {
	PGPASSWORD="$PGPASSWORD" psql -q -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d postgres "$@"
}

psql_test() {
	PGPASSWORD="$PGPASSWORD" psql -q -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" "$@"
}

wait_for_http() {
	for _ in $(seq 1 60); do
		if curl -fsS "$BASE_URL/" >/dev/null 2>&1; then
			return 0
		fi
		sleep 1
	done

	echo "backend did not become ready; last log lines:" >&2
	tail -120 "$LOG_FILE" >&2 || true
	return 1
}

json_field() {
	jq -r "$1"
}

register_user() {
	local username="$1"
	local password="$2"
	curl -fsS -X POST "$BASE_URL/auth/register" \
		-F "name=$username" \
		-F "username=$username" \
		-F "password=$password" \
		-F "email=$username@example.test" >/dev/null
}

login_user() {
	local username="$1"
	local password="$2"
	curl -fsS -X POST "$BASE_URL/auth/login" \
		-F "username=$username" \
		-F "password=$password" | json_field '.token'
}

seed_challenge() {
	local name="$1"
	local flag="$2"
	local max_attempts="$3"
	local dynamic="$4"
	local author_id
	author_id="$(psql_test -Atc "SELECT id FROM users ORDER BY id LIMIT 1")"

	psql_test -Atc "
		INSERT INTO challenges (
			created_at,
			updated_at,
			name,
			dynamic_flag,
			flag,
			type,
			difficulty,
			max_attempt_limit,
			format,
			container_id,
			image_id,
			status,
			deployment_type,
			author_id,
			health_check,
			points,
			max_points,
			min_points,
			server_deployed
		)
		VALUES (
			now(),
			now(),
			'$name',
			$dynamic,
			'$flag',
			'web',
			'easy',
			$max_attempts,
			'web',
			'container-$name',
			'image-$name',
			'Deployed',
			'standard_docker',
			$author_id,
			0,
			500,
			500,
			100,
			'localhost'
		)
		RETURNING id"
}

submit_concurrently() {
	local token="$1"
	local challenge_id="$2"
	local flag="$3"
	local requests="$4"

	python3 - "$BASE_URL" "$token" "$challenge_id" "$flag" "$requests" <<'PY'
import concurrent.futures
import json
import sys
import urllib.error
import urllib.parse
import urllib.request

base_url, token, challenge_id, flag, requests = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5])

def submit(_):
    data = urllib.parse.urlencode({"chall_id": challenge_id, "flag": flag}).encode()
    request = urllib.request.Request(
        base_url + "/api/submit/challenge",
        data=data,
        headers={"Authorization": "Bearer " + token},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            body = response.read().decode()
            return response.status, json.loads(body)
    except urllib.error.HTTPError as error:
        body = error.read().decode()
        try:
            parsed = json.loads(body)
        except json.JSONDecodeError:
            parsed = {"raw": body}
        return error.code, parsed

with concurrent.futures.ThreadPoolExecutor(max_workers=min(32, requests)) as executor:
    results = list(executor.map(submit, range(requests)))

print(json.dumps(results))
PY
}

assert_one_success() {
	local results_json="$1"
	local successes
	successes="$(jq '[.[] | select(.[1].success == true)] | length' <<<"$results_json")"
	if [[ "$successes" != "1" ]]; then
		echo "expected exactly one successful submit response, got $successes" >&2
		jq . <<<"$results_json" >&2
		return 1
	fi
}

assert_zero_successes() {
	local results_json="$1"
	local successes
	successes="$(jq '[.[] | select(.[1].success == true)] | length' <<<"$results_json")"
	if [[ "$successes" != "0" ]]; then
		echo "expected zero successful submit responses, got $successes" >&2
		jq . <<<"$results_json" >&2
		return 1
	fi
}

rm -rf "$TEST_HOME"
mkdir -p "$TEST_HOME/.beast/scripts" "$TEST_HOME/.beast/cache" "$TEST_HOME/.beast/remotes" "$TEST_HOME/.beast/uploads" "$TEST_HOME/.beast/secrets" "$TEST_HOME/.beast/staging" "$TEST_HOME/.beast/assets/logo"

psql_root -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS $PGDATABASE WITH (FORCE)" >/dev/null
psql_root -v ON_ERROR_STOP=1 -c "CREATE DATABASE $PGDATABASE" >/dev/null

cat >"$TEST_HOME/.beast/config.toml" <<EOF
authorized_keys_file = "$TEST_HOME/.beast/beast_authorized_keys"
scripts_dir = "$TEST_HOME/.beast/scripts"
allowed_base_images = ["ubuntu:24.04"]
beast_static_url = "http://localhost:$SERVER_PORT"
jwt_secret = "backend_submit_race_secret"
health_prober = false
ticker_frequency = 3000
default_cpu_shares = 1024
default_memory_limit = 536870912
default_pids_limit = 100
default_cpus_limit = 0.25

[available_servers.localhost]
host = "localhost"
username = ""
ssh_key_path = ""
port_range = "10000:10100"
active = true

[psql_config]
user = "$PGUSER"
password = "$PGPASSWORD"
dbname = "$PGDATABASE"
host = "$PGHOST"
port = "$PGPORT"
sslmode = "disable"

[redis_config]
host = "$REDIS_HOST"
port = "$REDIS_PORT"
password = ""
user = ""

[instance_config]
default_expiration = 300
max_extension = 600
max_instances_per_user = 3

[competition_info]
name = "backend-submit-race"
about = "backend verification"
starting_time = "00:00:00 UTC: +05:30, 1 January 2020, Wednesday"
ending_time = "23:59:59 UTC: +05:30, 1 January 2035, Monday"
timezone = "Asia/Calcutta: UTC +05:30"
prizes = ""
logo_url = ""
dynamic_score = false

[mail_config]
from = ""
password = ""
smtpHost = ""
smtpPort = ""
EOF

(
	cd "$ROOT_DIR"
	HOME="$TEST_HOME" GOCACHE="${GOCACHE:-/tmp/beast-go-cache}" GOMODCACHE="${GOMODCACHE:-/tmp/beast-go-modcache}" \
		go run ./cmd/beast run -p "$SERVER_PORT" -n >"$LOG_FILE" 2>&1
) &
SERVER_PID=$!

wait_for_http

register_user "apiwinner" "pw"
TOKEN_WINNER="$(login_user "apiwinner" "pw")"
CHALLENGE_CORRECT_ID="$(seed_challenge "api-race-correct" "flag{api-correct}" -1 false)"
CORRECT_RESULTS="$(submit_concurrently "$TOKEN_WINNER" "$CHALLENGE_CORRECT_ID" "flag{api-correct}" 64)"
assert_one_success "$CORRECT_RESULTS"

WINNER_SCORE="$(psql_test -Atc "SELECT score FROM users WHERE username = 'apiwinner'")"
if [[ "$WINNER_SCORE" != "500" ]]; then
	echo "expected apiwinner score 500, got $WINNER_SCORE" >&2
	exit 1
fi
SOLVED_ROWS="$(psql_test -Atc "SELECT COUNT(*) FROM user_challenges WHERE user_id = (SELECT id FROM users WHERE username = 'apiwinner') AND challenge_id = $CHALLENGE_CORRECT_ID AND solved = true")"
if [[ "$SOLVED_ROWS" != "1" ]]; then
	echo "expected one solved user_challenges row, got $SOLVED_ROWS" >&2
	exit 1
fi

register_user "apiwrong" "pw"
TOKEN_WRONG="$(login_user "apiwrong" "pw")"
CHALLENGE_WRONG_ID="$(seed_challenge "api-race-wrong" "flag{api-wrong}" 3 false)"
WRONG_RESULTS="$(submit_concurrently "$TOKEN_WRONG" "$CHALLENGE_WRONG_ID" "not-the-flag" 64)"
assert_zero_successes "$WRONG_RESULTS"

WRONG_TRIES="$(psql_test -Atc "SELECT tries FROM user_challenges WHERE user_id = (SELECT id FROM users WHERE username = 'apiwrong') AND challenge_id = $CHALLENGE_WRONG_ID")"
if [[ "$WRONG_TRIES" != "3" ]]; then
	echo "expected apiwrong tries 3, got $WRONG_TRIES" >&2
	exit 1
fi
WRONG_SCORE="$(psql_test -Atc "SELECT score FROM users WHERE username = 'apiwrong'")"
if [[ "$WRONG_SCORE" != "0" ]]; then
	echo "expected apiwrong score 0, got $WRONG_SCORE" >&2
	exit 1
fi

register_user "apidynone" "pw"
register_user "apidyntwo" "pw"
TOKEN_DYN_ONE="$(login_user "apidynone" "pw")"
TOKEN_DYN_TWO="$(login_user "apidyntwo" "pw")"
CHALLENGE_DYNAMIC_ID="$(seed_challenge "api-race-dynamic" "unused-static-flag" -1 true)"
psql_test -v ON_ERROR_STOP=1 -c "INSERT INTO dynamic_flags (created_at, updated_at, name, flag) VALUES (now(), now(), 'api-race-dynamic', 'flag{dynamic-shared}')" >/dev/null

DYNAMIC_RESULTS="$(
	python3 - "$BASE_URL" "$TOKEN_DYN_ONE" "$TOKEN_DYN_TWO" "$CHALLENGE_DYNAMIC_ID" <<'PY'
import concurrent.futures
import json
import sys
import urllib.parse
import urllib.request

base_url, token_one, token_two, challenge_id = sys.argv[1:5]

def submit(token):
    data = urllib.parse.urlencode({"chall_id": challenge_id, "flag": "flag{dynamic-shared}"}).encode()
    request = urllib.request.Request(
        base_url + "/api/submit/challenge",
        data=data,
        headers={"Authorization": "Bearer " + token},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=10) as response:
        return response.status, json.loads(response.read().decode())

with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
    results = list(executor.map(submit, [token_one, token_two]))
print(json.dumps(results))
PY
)"
assert_one_success "$DYNAMIC_RESULTS"

DYNAMIC_CLAIMS="$(psql_test -Atc "SELECT COUNT(*) FROM dynamic_flag_claims WHERE challenge_id = $CHALLENGE_DYNAMIC_ID AND flag = 'flag{dynamic-shared}'")"
if [[ "$DYNAMIC_CLAIMS" != "1" ]]; then
	echo "expected one dynamic flag claim, got $DYNAMIC_CLAIMS" >&2
	exit 1
fi

LEADERBOARD="$(curl -fsS -H "Authorization: Bearer $TOKEN_WINNER" "$BASE_URL/api/info/leaderboard?page=1")"
if ! jq -e '.[] | select(.username == "apiwinner" and .score == 500)' <<<"$LEADERBOARD" >/dev/null; then
	echo "leaderboard did not include apiwinner score 500" >&2
	jq . <<<"$LEADERBOARD" >&2
	exit 1
fi

echo "backend submit race verification passed"
