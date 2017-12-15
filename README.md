# otaru
[![Circle CI](https://circleci.com/gh/nyaxt/otaru/tree/master.svg?style=shield&circle-token=99fc14b26125325054679985cf796989fcc1b8be)](https://circleci.com/gh/nyaxt/otaru/tree/master)

Otaru is a cloud-backed filesystem for archiving your files. Otaru is optimized for storing personal collection of 10MB-10GB-ish files, such as book scan pdfs and disk image dumps.

For more details, see [Design Doc](https://docs.google.com/document/d/1j57oi9LrB8Viycwx3a9B5_Bgc9tzRir3RyBr6gwBu5g/edit?usp=sharing)

## Quick Start

### Build otaru inside Docker container
Building otaru takes a bit of time (Approx 6 min with decent internet connection). You may want to start building while doing other setup.

    $ git clone https://github.com/nyaxt/otaru && cd otaru
    $ docker build -t otaru .
    $ docker run -ti --rm -v `pwd`/out:/out otaru

### Configure Google Cloud Platform
- Access [Google Cloud Console](https://console.developers.google.com), and have a project ready (preferrably not the default "API Project") with Cloud Datastore and Cloud Storage enabled.
- Allow the following API usage from "APIs & auth" -> "APIs":
  - Google Cloud Datastore API
  - Google Cloud Storage API
  - Google Cloud Storage JSON API
- Create two new buckets to store Otaru blobs/metadata. You can create new bucket from "Storage" -> "Cloud Storage" -> "Browser".
  - Otaru supports using separate bucket for storing metadata, which is accessed more frequently compared to blobs.
  - Blob bucket may be any of Standard, Durable Reduced Availability, or Nearline. However, it is recommended to store metadata in Standard class bucket.
  - Metadata bucket name must be blob bucket name + "-meta" suffix. For example, if your bucket for storing blobs is named "otaru-foobar", metadata bucket must be named "otaru-foobar-meta"
- Issue OAuth 2.0 client ID. "APIs & auth" -> "Credentials" -> "Add credentials" -> "API key"
  - Choose "OAuth 2.0 client ID", Application type "Other".
  - Name the ID "Otaru client" or something distinguishable.
  - Download the client secret by clicking on the "Download JSON" icon button located on the left of the table.

### Create {config,password} file & place OAuth 2.0 credentials

    $ mkdir ~/.otaru # You may change this dir to any dir you want, but a new directory is needed as otaru has multiple config files to keep.
    $ cp doc/config.toml.example ~/.otaru/config.toml
    $ $EDITOR ~/.otaru/config.toml # replace placeholders
    $ cp [downloaded-client-secret-json] ~/.otaru/credentials.json
    $ echo [your-password] > ~/.otaru/password.txt # configure encryption key

### Setup Google Cloud SDK

#### Install Google Cloud SDK
Install `gcloud` command per instructions: https://cloud.google.com/sdk/

#### Enable Alpha/Beta commands
    $ gcloud components update alpha beta

#### Authenticate gcloud tool
    $ gcloud auth login

### Authorize Google Cloud Storage / Datastore access to Otaru
Complete the step "Build otaru inside Docker container" before executing below.

    $ OTARUDIR=~/.otaru out/otaru-gcloudauthcli

Visit displayed url, and paste response code. Check that `~/.otaru/tokencache.json` is correctly generated.

### Generate self-signed cert+key for quick testing.
Otaru requires TLS for its WebUI and api server.
Below command will generate self-signed X.509 certificate and key pair at ~/.otaru/cert{,-key}.pem :

    $ go get -u -v github.com/cloudflare/cfssl/cmd/cfssl{,json}
    $ OTARUDIR=~/.otaru scripts/gen_self_signed_cert.bash

**Warning:** Use the generated self-signed certificates only for quick testing purposes, not for production.

### Setup Google Cloud Datastore index, and verify Google Cloud Storage settings

    $ OTARUDIR=~/.otaru scripts/gcloud_setup.bash

### Mount!
    
    $ mkdir -p /otaru/foobar && sudo chown `whoami` /otaru/foobar
    $ OTARUDIR=~/.otaru out/otaru-mkfs /otaru/foobar
    $ OTARUDIR=~/.otaru out/otaru-server /otaru/foobar

Enjoy using /otaru/foobar. Press Ctrl-C to start unmount sequence. Navigate to http://localhost:10246 for webui.
