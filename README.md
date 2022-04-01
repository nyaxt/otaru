# otaru
Otaru is a cloud-backed filesystem for archiving your files. Otaru is optimized for storing personal collection of 10MB-10GB-ish files, such as book scan pdfs and disk image dumps.

For more details, see [Design Doc](https://docs.google.com/document/d/1j57oi9LrB8Viycwx3a9B5_Bgc9tzRir3RyBr6gwBu5g/edit?usp=sharing)

## Quick Start

### Build otaru inside Docker container
Building otaru takes a bit of time (Approx 6 min. with decent internet connection). You may want to start building while doing other setup.

    $ git clone https://github.com/nyaxt/otaru && cd otaru
    $ docker build -t otaru .
    $ docker run -ti --rm -v `pwd`/out:/out otaru

### Configure Google Cloud Platform
- Access [Google Cloud Console](https://console.cloud.google.com), and have a project ready (preferrably not the default "API Project").
- Enable Cloud Datastore and Cloud Storage for the project.
- Allow the following API usage from "APIs & auth" -> "APIs":
  - Google Cloud Datastore API
  - Google Cloud Storage API
  - Google Cloud Storage JSON API
- Create two new buckets to store Otaru blobs/metadata. You can create new bucket from "Storage" -> "Cloud Storage" -> "Browser".
  - Otaru supports using separate bucket for storing metadata, which is accessed more frequently compared to blobs.
  - Blob bucket may be any of Standard, Durable Reduced Availability, or Nearline. However, it is recommended to store metadata in Standard class bucket.
  - Metadata bucket name must be blob bucket name + "-meta" suffix. For example, if your bucket for storing blobs is named `otaru-foobar`, metadata bucket must be named `otaru-foobar-meta`.

### Create {config,password} file

    $ mkdir ~/.otaru # You may change this dir to any dir you want, but a new directory is needed as otaru has multiple config files to keep.
    $ cp doc/config.toml.example ~/.otaru/config.toml
    $ $EDITOR ~/.otaru/config.toml # replace placeholders
    $ echo [your-password] > ~/.otaru/password.txt # configure encryption key

### Setup Google Cloud SDK

#### Install Google Cloud SDK
Install `gcloud` command per instructions: https://cloud.google.com/sdk/

#### Enable Alpha/Beta commands
    $ gcloud components update alpha beta

#### Authenticate gcloud tool
    $ gcloud auth login

### Create and configure service account

    $ export account=otaru-user
    $ export project=$(gcloud config get-value core/project)
    $ gcloud iam service-accounts create ${account}
    $ gcloud iam service-accounts keys create ~/.otaru/credentials.json --iam-account ${account}@${project}.iam.gserviceaccount.com
    $ gcloud projects add-iam-policy-binding ${project} --member serviceAccount:${account}@${project}.iam.gserviceaccount.com --role "roles/datastore.owner"
    $ gsutil -m acl ch -r -u ${account}@${project}.iam.gserviceaccount.com:O gs://${bucket}
    $ gsutil -m acl ch -r -u ${account}@${project}.iam.gserviceaccount.com:O gs://${bucket}-meta

### Generate self-signed cert+key for quick testing.
Otaru requires TLS for its WebUI and api server.
Below command will generate self-signed X.509 certificate and key pair at `~/.otaru/cert{,-key}.pem`:

    $ go get -u -v github.com/cloudflare/cfssl/cmd/cfssl{,json}
    $ OTARUDIR=~/.otaru scripts/gen_self_signed_cert.bash

**Warning:** Use the generated self-signed certificates only for quick testing purposes, not for production.

### Setup Google Cloud Datastore index, and verify Google Cloud Storage settings

    $ OTARUDIR=~/.otaru scripts/gcloud_setup.bash

### Mount!

    $ OTARUDIR=~/.otaru out/otaru-mkfs
    $ OTARUDIR=~/.otaru out/otaru-server

Navigate to http://localhost:10246 for webui.
