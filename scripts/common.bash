#!/bin/bash

readonly BASEDIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

readonly OTARUDIR="${OTARUDIR:-$HOME/.otaru}"
if [[ ! -d $OTARUDIR ]]; then
	echo "OTARUDIR=\"$OTARUDIR\" not found or not a dir" >/dev/stderr
	exit 1
fi

readonly OTARUCONF="$OTARUDIR/config.toml"

function otaru::parse_config_toml() {
	if [[ ! -f $OTARUCONF ]]; then
		echo "otaru config file \"$OTARUCONF\" not found!"
	fi

	readonly PROJECT_NAME=$(perl -ne 'print $1 if /^\s*project_name\s*=\s*"?([\w-_]+)"?/' $OTARUCONF)
	if [[ -z $PROJECT_NAME ]]; then
		echo "Failed to find \"project_name\" from $OTARUCONF" >/dev/stderr
		exit 1
	fi

	readonly BUCKET_NAME=$(perl -ne 'print $1 if /^\s*bucket_name\s*=\s*"?([\w-_]+)"?/' $OTARUCONF)
	if [[ -z $BUCKET_NAME ]]; then
		echo "Failed to find \"bucket_name\" from $OTARUCONF" >/dev/stderr
		exit 1
	fi
	readonly META_BUCKET_NAME=$BUCKET_NAME-meta

	readonly USE_SEPARATE_BUCKET_FOR_METADATA=$(perl -ne 'print "true" if /^[^#]*use_separate_bucket_for_metadata\s*=\s*true/' $OTARUCONF)
}

function otaru::gcloud_setup_datastore() {
	read -p "Setup gcloud datastore indices for otaru (y/N)? " answer
	case ${answer:0:1} in
		y|Y)
			;;
		*)
			return
			;;
	esac
	gcloud preview datastore create-indexes --project $PROJECT_NAME $BASEDIR/resources/index.yaml
}

function otaru::gcloud_verify_storage() {
	echo "Checking primary blobstore bucket: $BUCKET_NAME"
	gsutil ls -L -b -p $PROJECT_NAME gs://$BUCKET_NAME | tee /tmp/bucketinfo || {
		echo "Failed to find primary blobstore bucket: $BUCKET_NAME" >/dev/stderr
		rm /tmp/bucketinfo
		exit 1
	}
	META_STORAGE_CLASS=$(grep "Storage class" /tmp/bucketinfo | awk '{ print $3 }')
	rm /tmp/bucketinfo

	if [[ $USE_SEPARATE_BUCKET_FOR_METADATA == true ]]; then
		echo "INFO: configured to use separate bucket for metadata"
		echo "Checking metadata blobstore bucket: $BUCKET_NAME"
		gsutil ls -L -b -p $PROJECT_NAME gs://$META_BUCKET_NAME | tee /tmp/metabucketinfo || {
			echo "Failed to find metadata blobstore bucket: $META_BUCKET_NAME" >/dev/stderr
			exit 1
		}
		META_STORAGE_CLASS=$(grep "Storage class" /tmp/metabucketinfo | awk '{ print $3 }')
	fi

	if [[ $META_STORAGE_CLASS == NEARLINE ]]; then
		echo "WARNING: Storing metadata on nearline storage is highly discouraged.\n- Metadata for otaru is flushed frequently, and you will get charged for its historical data too." >/dev/stderr
	else
		echo "Detected $META_STORAGE_CLASS as metadata bucket storage class. Good!"
	fi
}

function otaru::gcloud_verify_credentials_json() {
	if [[ -f $OTARUDIR/credentials.json ]]; then
		echo "credentials.json found. Good!"
		(grep -q client_secret $OTARUDIR/credentials.json) || {
			echo "client_secret not found in credentials.json.">/dev/stderr
			exit 1
		}
	else
		echo "credentials.json not found. Please create one at https://console.developers.google.com/project/$PROJECT_NAME/apiui/credential and place it as $OTARUDIR/credentials.json.">/dev/stderr
		exit 1
	fi
}

function otaru::gcloud_setup() {
	otaru::parse_config_toml

	gcloud version >/dev/null || {
		echo "Google Cloud SDK not found in \$PATH. Please install: https://cloud.google.com/sdk/"
		exit 1
	}

	otaru::gcloud_setup_datastore
	otaru::gcloud_verify_storage
	otaru::gcloud_verify_credentials_json

	echo "Please manually make sure that the below APIs are enabled:"
	echo "- Google Datastore API: https://console.developers.google.com/project/$PROJECT_NAME/apiui/apiview/datastore/overview"
	echo "- Google Cloud Storage API : https://console.developers.google.com/project/$PROJECT_NAME/apiui/apiview/storage_component/overview"
}

function otaru::update_version() {
	(
		gitcommit=`git rev-parse HEAD`
		buildhost=`hostname -f`
		unixtime=`date +%s`
		echo "package version"
		echo
		echo "const GIT_COMMIT = \"$gitcommit\""
		echo "const BUILD_HOST = \"$buildhost\""
		echo "const BUILD_TIME = $unixtime"
	)>/tmp/consts.go || {
		echo "Failed to generate version/consts/go"
		exit 1
	}
	cp /tmp/consts.go $BASEDIR/../version/consts.go
}
