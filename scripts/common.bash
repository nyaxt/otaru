#!/bin/bash

readonly BASEDIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

readonly OTARUDIR="${OTARUDIR:-$HOME/.otaru}"
if [ ! -d $OTARUDIR ]; then
	echo "OTARUDIR=\"$OTARUDIR\" not found or not a dir" >/dev/stderr
	exit 1
fi

readonly OTARUCONF="$OTARUDIR/config.toml"

function otaru::parse_config_toml() {
	if [ ! -f $OTARUCONF ]; then
		echo "otaru config file \"$OTARUCONF\" not found!"
	fi

	readonly PROJECT_NAME=$(perl -ne 'print $1 if /^[^#]*project_name\s*=\s*"?([\w-_]+)"?/' $OTARUCONF)
	if [[ -z $PROJECT_NAME ]]; then
		echo "Failed to find \"project_name\" from $OTARUCONF" >/dev/stderr
		exit 1
	fi
}

function otaru::gcloud_setup_datastore() {
	gcloud preview datastore create-indexes --project $PROJECT_NAME $BASEDIR/resources/index.yaml
}

function otaru::gcloud_setup() {
	otaru::parse_config_toml

	gcloud version >/dev/null || {
		echo "Google Cloud SDK not found in \$PATH. Please install: https://cloud.google.com/sdk/"
		exit 1
	}

	otaru::gcloud_setup_datastore
}
