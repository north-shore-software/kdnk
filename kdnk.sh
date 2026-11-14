#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$REPO_ROOT/examples/config.env"

# Every relative path below (./examples/..., go.mod) is resolved against the
# repo, so the script works when invoked from anywhere.
cd "$REPO_ROOT"

GEMINI_MODELS_URL="https://generativelanguage.googleapis.com/v1beta/models"
OLLAMA_DEFAULT_HOST="http://localhost:11434"
LOCAL_MODEL="gemma4"
DEFAULT_PORT=8090

if [ -t 1 ]; then
	B=$'\033[1m'; DIM=$'\033[2m'; RED=$'\033[31m'; YEL=$'\033[33m'; GRN=$'\033[32m'; R=$'\033[0m'
else
	B=""; DIM=""; RED=""; YEL=""; GRN=""; R=""
fi

say()  { printf '%s\n' "$*"; }
step() { printf '\n%s%s%s\n' "$B" "$*" "$R"; }
note() { printf '%s%s%s\n' "$DIM" "$*" "$R"; }
warn() { printf '%swarning:%s %s\n' "$YEL" "$R" "$*" >&2; }
die()  { printf '%serror:%s %s\n' "$RED" "$R" "$*" >&2; exit 1; }

run_cmd() {
	printf '%s$ %s%s\n' "$DIM" "$*" "$R"
	"$@"
}

ask() {
	local prompt="$1" default="$2" reply
	read -r -p "$prompt [$default]: " reply || reply=""
	printf '%s' "${reply:-$default}"
}

ask_yn() {
	local prompt="$1" default="$2" reply
	read -r -p "$prompt " reply || reply=""
	reply="${reply:-$default}"
	case "$reply" in
		[yY]*) return 0 ;;
		*) return 1 ;;
	esac
}

version_ge() {
	local IFS=.
	local -a have=($1) want=($2)
	local i x y
	for ((i = 0; i < 3; i++)); do
		x="${have[i]:-0}"
		y="${want[i]:-0}"
		if ((10#$x > 10#$y)); then return 0; fi
		if ((10#$x < 10#$y)); then return 1; fi
	done
	return 0
}

describe() {
	case "$1" in
		todo) printf 'a task list. Static system prompt; tools that write and read a growing collection.' ;;
		rpg)  printf 'a short game about building an AI app. Per-conversation state, injected into the prompt every turn.' ;;
		*)    printf 'a kdnk example.' ;;
	esac
}

first_message() {
	case "$1" in
		todo) printf 'add: file taxes, urgency 8, value 9' ;;
		rpg)  printf "let's start" ;;
		*)    printf 'hello' ;;
	esac
}

config_value() {
	[ -f "$CONFIG_FILE" ] || return 0
	sed -n "s/^$1=\(.*\)$/\1/p" "$CONFIG_FILE" | head -1
}

port_free() {
	! (exec 3<>"/dev/tcp/127.0.0.1/$1") >/dev/null 2>&1
}

normalise_host() {
	case "$1" in
		http://*|https://*) printf '%s' "$1" ;;
		*) printf 'http://%s' "$1" ;;
	esac
}

list_examples() {
	local examples=()
	for dir in "$REPO_ROOT"/examples/*/; do
		[ -f "$dir/main.go" ] || continue
		examples+=("$(basename "$dir")")
	done
	# printf with an empty array still runs the format once; guard so callers
	# see no lines at all rather than one empty name.
	[ "${#examples[@]}" -gt 0 ] || return 0
	printf '%s\n' "${examples[@]}"
}

# Callers capture stdout to get the answer, so the menu and everything else
# meant for the human goes to stderr. Only the final name is printed to stdout.
pick_example() {
	local given="${1:-}"
	local examples=()
	while IFS= read -r line; do
		[ -n "$line" ] && examples+=("$line")
	done < <(list_examples)
	[ "${#examples[@]}" -gt 0 ] || die "no examples found under $REPO_ROOT/examples"

	if [ -n "$given" ]; then
		for name in "${examples[@]}"; do
			if [ "$name" = "$given" ]; then
				printf '%s' "$given"
				return 0
			fi
		done
		warn "unknown example '$given'."
	fi

	local default_example="${examples[0]}"
	for name in "${examples[@]}"; do
		if [ "$name" = "todo" ]; then default_example="todo"; fi
	done

	say "" >&2
	for i in "${!examples[@]}"; do
		printf '  %d) %s%s%s — %s\n' "$((i + 1))" "$B" "${examples[i]}" "$R" "$(describe "${examples[i]}")" >&2
	done
	say "" >&2

	local choice selected=""
	while [ -z "$selected" ]; do
		choice="$(ask "  example (number or name)" "$default_example")"
		if [ "$choice" -eq "$choice" ] 2>/dev/null; then
			if [ "$choice" -ge 1 ] && [ "$choice" -le "${#examples[@]}" ]; then
				selected="${examples[$((choice - 1))]}"
			fi
		else
			for name in "${examples[@]}"; do
				if [ "$name" = "$choice" ]; then selected="$name"; fi
			done
		fi
		[ -n "$selected" ] || warn "pick a number from 1 to ${#examples[@]}, or an example name."
	done
	printf '%s' "$selected"
}

configure_cloud() {
	local existing_key="${1:-}"
	local key="" attempts=0 code suggestion=""

	if [ -n "$existing_key" ]; then
		suggestion="the existing saved key"
	elif [ -n "${GEMINI_API_KEY:-}" ]; then
		suggestion="the GEMINI_API_KEY already in your environment"
	fi

	if [ -n "$suggestion" ] && ask_yn "  Reuse $suggestion? [Y/n]" "y"; then
		key="${existing_key:-${GEMINI_API_KEY:-}}"
	fi

	say ""

	while true; do
		while [ -z "$key" ]; do
			attempts=$((attempts + 1))
			[ "$attempts" -le 5 ] || die "no usable API key after $((attempts - 1)) attempts. Get one at https://aistudio.google.com/apikey and re-run setup."
			read -r -s -p "  Gemini API key (not echoed): " key || key=""
			printf '\n'
			[ -n "$key" ] || warn "empty. Get a key at https://aistudio.google.com/apikey"
		done

		note "checking the key against $GEMINI_MODELS_URL"
		code="$(curl -sS -m 10 -o /dev/null -w '%{http_code}' -H "x-goog-api-key: $key" "$GEMINI_MODELS_URL" 2>/dev/null)" || code=""

		case "$code" in
			200)
				say "key accepted. ${GRN}ok${R}"
				break
				;;
			400|401|403)
				warn "Google rejected that key (HTTP $code)."
				key=""
				;;
			*)
				warn "could not reach Google to check the key${code:+ (HTTP $code)}. Continuing with key as given."
				break
				;;
		esac
	done

	GEMINI_API_KEY_VALUE="$key"
}

configure_local() {
	local host tags

	# An explicit OLLAMA_HOST wins, then whatever the last setup saved.
	host="$(normalise_host "${OLLAMA_HOST:-${EXISTING_OLLAMA_HOST:-$OLLAMA_DEFAULT_HOST}}")"
	if [ -n "${OLLAMA_HOST:-}" ] && [ "$host" != "$OLLAMA_HOST" ]; then
		warn "OLLAMA_HOST is '$OLLAMA_HOST' with no scheme. Using '$host' instead."
	fi

	while true; do
		note "probing $host/api/tags"
		if tags="$(curl -sf -m 5 "$host/api/tags" 2>/dev/null)"; then
			say "Ollama is up at $host. ${GRN}ok${R}"
			break
		fi

		warn "no Ollama at $host. Start it with 'ollama serve', or set OLLAMA_HOST."
		say ""
		say "  1) retry"
		say "  2) switch to cloud (Gemini)"
		say "  3) continue anyway"
		say ""
		case "$(ask "  what now" "1")" in
			2) MODEL_SET="cloud"; configure_cloud "$EXISTING_GEMINI_KEY"; return ;;
			3) tags=""; break ;;
			*) ;;
		esac
	done

	if printf '%s' "$tags" | grep -q "\"$LOCAL_MODEL"; then
		say "$LOCAL_MODEL is present. ${GRN}ok${R}"
	else
		warn "$LOCAL_MODEL is not pulled on that host."
		if ask_yn "  Pull $LOCAL_MODEL now? [y/N]" "n"; then
			if command -v ollama >/dev/null 2>&1; then
				run_cmd ollama pull "$LOCAL_MODEL"
			else
				note "pulling over API"
				curl -N -X POST "$host/api/pull" -d "{\"model\":\"$LOCAL_MODEL\"}"
				printf '\n'
			fi
		else
			warn "skipping pull. Run 'ollama pull $LOCAL_MODEL' manually if needed."
		fi
	fi

	OLLAMA_HOST_VALUE="$host"
}

check_go() {
	command -v go >/dev/null 2>&1 || die "go is not on PATH. Install it from https://go.dev/dl/"

	local want_go have_go
	want_go="$(sed -n 's/^go[[:space:]]\{1,\}\([0-9][0-9.]*\).*/\1/p' "$REPO_ROOT/go.mod" | head -1)"
	have_go="$(go env GOVERSION)"
	have_go="${have_go#go}"

	if [ -n "$want_go" ] && ! version_ge "$have_go" "$want_go"; then
		die "go $have_go is too old; go.mod needs $want_go or newer."
	fi

	say "go $have_go at $(command -v go), go.mod wants $want_go. ${GRN}ok${R}"
}

check_node() {
	command -v node >/dev/null 2>&1 || die "node is not on PATH. Install it from https://nodejs.org/"
	command -v pnpm >/dev/null 2>&1 || die "pnpm is not on PATH. Install it from https://pnpm.io/installation"
	say "node $(node -v) at $(command -v node), pnpm $(pnpm -v) at $(command -v pnpm). ${GRN}ok${R}"
}

pnpm_install() {
	(cd "$1" && run_cmd pnpm install) || die "pnpm install failed in ${1#"$REPO_ROOT"/}."
}

web_dirs() {
	printf '%s\n' "$REPO_ROOT/docs/site"
	local dir
	for dir in "$REPO_ROOT"/examples/*/web/; do
		[ -f "$dir/package.json" ] && printf '%s\n' "${dir%/}"
	done
	return 0
}

cmd_setup() {
	step "1. Checking tools"
	check_go
	check_node

	step "2. Installing JavaScript dependencies"
	local dir
	while IFS= read -r dir; do
		pnpm_install "$dir"
	done < <(web_dirs)

	step "3. Config setup"

	cat <<EOF

  1) ${B}cloud${R} — Gemini. Needs an API key. Nothing to download.
  2) ${B}local${R} — Ollama serving $LOCAL_MODEL. No key, but a multi-GB model pull.

EOF

	# configure_local can hand control back to configure_cloud, so the saved key
	# and host have to outlive this function's locals.
	local existing_model_set="" existing_port=""
	EXISTING_GEMINI_KEY=""
	EXISTING_OLLAMA_HOST=""
	if [ -f "$CONFIG_FILE" ]; then
		existing_model_set="$(config_value MODEL_SET)"
		existing_port="$(config_value PORT)"
		EXISTING_GEMINI_KEY="$(config_value GEMINI_API_KEY_VALUE)"
		EXISTING_OLLAMA_HOST="$(config_value OLLAMA_HOST_VALUE)"
	fi

	local choice
	MODEL_SET=""
	while [ -z "$MODEL_SET" ]; do
		choice="$(ask "  model set" "${existing_model_set:-cloud}")"
		case "$choice" in
			1|cloud) MODEL_SET="cloud" ;;
			2|local) MODEL_SET="local" ;;
			*) warn "expected 'cloud' or 'local'." ;;
		esac
	done

	GEMINI_API_KEY_VALUE=""
	OLLAMA_HOST_VALUE=""

	if [ "$MODEL_SET" = "cloud" ]; then
		configure_cloud "$EXISTING_GEMINI_KEY"
	else
		configure_local
	fi

	step "4. Which port?"

	PORT=""
	while [ -z "$PORT" ]; do
		choice="$(ask "  port" "${existing_port:-$DEFAULT_PORT}")"
		if ! [ "$choice" -eq "$choice" ] 2>/dev/null || [ "$choice" -lt 1 ] || [ "$choice" -gt 65535 ]; then
			warn "'$choice' is not a port number."
			continue
		fi
		if port_free "$choice"; then
			PORT="$choice"
		else
			warn "something is already listening on 127.0.0.1:$choice.
If that is an older copy of this app, stop it first — otherwise the new build
will die on bind while the old one keeps answering, which looks exactly like
your rebuild having no effect."
		fi
	done

	say "using port ${B}${PORT}${R}"

	step "5. Writing configuration"

	# The API key lands in this file, so create it private and fill it after —
	# chmod'ing post-write leaves a window at the ambient umask.
	umask 077
	: > "$CONFIG_FILE"
	chmod 600 "$CONFIG_FILE"

	{
		printf 'MODEL_SET=%s\n' "$MODEL_SET"
		printf 'GEMINI_API_KEY_VALUE=%s\n' "$GEMINI_API_KEY_VALUE"
		printf 'OLLAMA_HOST_VALUE=%s\n' "$OLLAMA_HOST_VALUE"
		printf 'PORT=%s\n' "$PORT"
	} > "$CONFIG_FILE"
	say "saved configuration to ${B}examples/config.env${R}. ${GRN}ok${R}"
}

cmd_run() {
	local mode="$1"
	shift
	[ -f "$CONFIG_FILE" ] || {
		warn "no configuration found at examples/config.env. Running setup first..."
		cmd_setup
	}

	MODEL_SET="$(config_value MODEL_SET)"
	GEMINI_API_KEY_VALUE="$(config_value GEMINI_API_KEY_VALUE)"
	OLLAMA_HOST_VALUE="$(config_value OLLAMA_HOST_VALUE)"
	PORT="$(config_value PORT)"

	# A truncated or hand-edited config otherwise surfaces as an empty
	# KDNK_MODEL_SET and a confusing failure at the first message.
	case "$MODEL_SET" in
		cloud)
			[ -n "$GEMINI_API_KEY_VALUE" ] || die "config has MODEL_SET=cloud but no API key. Re-run: ./kdnk.sh setup"
			;;
		local)
			[ -n "$OLLAMA_HOST_VALUE" ] || die "config has MODEL_SET=local but no Ollama host. Re-run: ./kdnk.sh setup"
			;;
		*)
			die "config has an unusable MODEL_SET ('$MODEL_SET'). Re-run: ./kdnk.sh setup"
			;;
	esac

	if ! [ "$PORT" -eq "$PORT" ] 2>/dev/null || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
		die "config has an unusable PORT ('$PORT'). Re-run: ./kdnk.sh setup"
	fi

	local example_arg="${1:-}"
	local EXAMPLE
	EXAMPLE="$(pick_example "$example_arg")"

	local web_dir="$REPO_ROOT/examples/$EXAMPLE/web"
	local has_web=""
	[ -f "$web_dir/package.json" ] && has_web=1
	if [ "$mode" = "dev" ] && [ -z "$has_web" ]; then
		die "$EXAMPLE has no Vite app (examples/$EXAMPLE/web/package.json). Use: ./kdnk.sh run $EXAMPLE"
	fi

	step "1. Checking tools"
	check_go
	[ -z "$has_web" ] || check_node

	step "2. Building $EXAMPLE"
	if [ -n "$has_web" ]; then
		[ -d "$web_dir/node_modules" ] || pnpm_install "$web_dir"
		if [ "$mode" = "run" ]; then
			(cd "$web_dir" && run_cmd pnpm build) || die "web build failed."
		fi
	fi
	mkdir -p "$REPO_ROOT/examples/$EXAMPLE/bin"
	run_cmd env CGO_ENABLED=0 go build -o "$REPO_ROOT/examples/$EXAMPLE/bin/$EXAMPLE" "./examples/$EXAMPLE" || die "build failed."

	say "built ${B}examples/$EXAMPLE/bin/$EXAMPLE${R}. ${GRN}ok${R}"

	step "3. Running $EXAMPLE"

	port_free "$PORT" || die "something is already listening on 127.0.0.1:$PORT.
If that is an older copy of this app, stop it first — otherwise this build dies
on bind while the old one keeps answering, which looks exactly like your rebuild
having no effect."

	mkdir -p "$REPO_ROOT/examples/$EXAMPLE/pb_data"

	export KDNK_MODEL_SET="$MODEL_SET"
	if [ "$MODEL_SET" = "cloud" ]; then
		export GEMINI_API_KEY="$GEMINI_API_KEY_VALUE"
	else
		export OLLAMA_HOST="$OLLAMA_HOST_VALUE"
	fi

	local server=("$REPO_ROOT/examples/$EXAMPLE/bin/$EXAMPLE" serve --http=127.0.0.1:"$PORT" --dir "$REPO_ROOT/examples/$EXAMPLE/pb_data")

	if [ "$mode" = "run" ]; then
		cat <<EOF

Starting ${B}$EXAMPLE${R} at ${B}http://127.0.0.1:$PORT/${R}

Dashboard: http://127.0.0.1:$PORT/_/
First message suggestion: ${B}$(first_message "$EXAMPLE")${R}

EOF
		exec "${server[@]}"
	fi

	cat <<EOF

Starting ${B}$EXAMPLE${R} API at ${B}http://127.0.0.1:$PORT/${R} and the Vite dev server below.
Open the Vite URL; it forwards /api to port $PORT.

Dashboard: http://127.0.0.1:$PORT/_/
First message suggestion: ${B}$(first_message "$EXAMPLE")${R}

EOF

	"${server[@]}" &
	SERVER_PID=$!
	trap 'kill "$SERVER_PID" 2>/dev/null; wait "$SERVER_PID" 2>/dev/null' EXIT
	trap 'exit 130' INT TERM

	cd "$web_dir"
	KDNK_PORT="$PORT" pnpm dev
}

cmd_clear() {
	local example_arg="${1:-}"
	local EXAMPLE
	EXAMPLE="$(pick_example "$example_arg")"

	step "Clearing $EXAMPLE data"
	local data_dir="$REPO_ROOT/examples/$EXAMPLE/pb_data"
	if [ -d "$data_dir" ]; then
		rm -rf "$data_dir"
		say "cleared ${B}examples/$EXAMPLE/pb_data${R}. ${GRN}ok${R}"
	else
		say "no data directory found at ${B}examples/$EXAMPLE/pb_data${R}."
	fi
}

cmd_docs() {
	local site_dir="$REPO_ROOT/docs/site"
	command -v pnpm >/dev/null 2>&1 || die "pnpm is not on PATH. Install it from https://pnpm.io/installation"

	cd "$site_dir"
	if [ ! -d "$site_dir/node_modules" ]; then
		step "Installing docs dependencies"
		run_cmd pnpm install || die "pnpm install failed."
	fi

	step "Starting docs server"
	exec pnpm dev
}

usage() {
	local names
	names="$(list_examples | paste -sd, - | sed 's/,/, /g')"
	say "Usage: ./kdnk.sh <command> [example]"
	say ""
	say "Commands:"
	say "  setup            Check tools, install JavaScript dependencies, and configure"
	say "                   model set, API key, Ollama host, and port"
	say "  run [example]    Build and run an example${names:+ ($names)}"
	say "  dev [example]    Run an example's server and its Vite dev server"
	say "  clear [example]  Delete an example's local data"
	say "  docs             Run the docs site dev server"
	say ""
	say "Omit [example] to pick one from a menu."
	say ""
	exit "${1:-1}"
}

[ $# -gt 0 ] || usage

subcmd="$1"
shift || true

case "$subcmd" in
	setup) cmd_setup ;;
	run) cmd_run run "${1:-}" ;;
	dev) cmd_run dev "${1:-}" ;;
	clear) cmd_clear "${1:-}" ;;
	docs) cmd_docs ;;
	-h|--help|help) usage 0 ;;
	*) warn "unknown command '$subcmd'."; usage ;;
esac
