# OpenFaaS Slack Function

This repository creates and deploys an [OpenFaaS] function that uses Golang to
send messages to Slack channels.

## Installing OpenFaaS

I'm not going to go into any great detail for installing and deploying OpenFaaS,
I'll do that as a separate set of instructions later on. I essentially followed
the directions from [OpenFaas Deployment] and used the awesome [Arkade] CLI
installer for Kubernetes applications, plus some of the linked blog posts.

## Slack App

See [Slack Apps] for your list of available apps, and where you can create a new
one.

_TODO:_ Provide some docs on creating a Slack app.

Ensure that your Slack app has the [chat:write] scope.

### OpenFaaS Slack API Token Secret

When creating the Slack application it will provide you with a Slack Bot token
that is required for authentication and sending messages. This needs to be added
as secret in the `openfaas-fn` namespace so that it is available for use by the
function. When the function is deployed the secret will be mounted as a file to
`/var/openfaas/secrets/slack-api-token` and the value can be read by the
function. See [OpenFaas Using secrets] for more information.

```sh
kubectl create secret generic slack-api-token \
  --from-literal slack-api-token="xoxb-1234-17401234-30xxxxxxx" \
  --namespace openfaas-fn
```

## Private Docker Registry

When deploying functions from a private registry OpenFaaS needs the credentials
to be able to authenticate to it when pulling images. See [Use a private
registry with Kubernetes] for more information on this.

Run the below command to create the Docker registry credentials secret in the
`openfaas-fn` namespace.

```sh
kubectl create secret docker-registry homelab-docker-registry \
  --docker-username=homelab-user \
  --docker-password=homelab-password \
  --docker-email=homelab@example.com \
  --docker-server=https://registry.mydomain.io \
  --namespace openfaas-fn
```

Add the below yaml to the `default` service account in the `openfaas-fn`
namespace so that it has the credentials to authenticate with the registry when
pulling images.

```sh
kubectl edit serviceaccount default -n openfaas-fn
```

```yaml
imagePullSecrets:
  - name: homelab-docker-registry
```

## Creating the Function

The below steps were followed to create a new function and handler.

Run the command below to pull the `golang-http` template that creates an HTTP
request handler for Golang.

```sh
faas-cli template store pull golang-http
```

Run the command below to create the function definition files and empty function
handler.

```sh
$ faas new --lang golang-http slack
Folder: slack created.
  ___                   _____           ____
 / _ \ _ __   ___ _ __ |  ___|_ _  __ _/ ___|
| | | | '_ \ / _ \ '_ \| |_ / _` |/ _` \___ \
| |_| | |_) |  __/ | | |  _| (_| | (_| |___) |
 \___/| .__/ \___|_| |_|_|  \__,_|\__,_|____/
      |_|


Function created in folder: slack
Stack file written: slack.yml
```

## Building the Function

The OpenFaaS documentation and [Simple Serverless with Golang Functions and
Microservices] provide instruction on how to develop and build OpenFaaS
functions.

### Golang Dependencies

This function uses additional Go libraries that need to be included as
dependencies when building. See [GO - Dependencies] for options on including
these dependencies. This repository uses the [Dep] dependency management tool
for this.

```sh
brew install dep
```

```sh
dep init
```

I found that as my code already had the imports it automatically created the
`Gopkg.toml` and `Gopkg.lock` files with those dependencies. The
`dep ensure -add github.com/slack-go/slack` command for example returned an
error saying this dependency had already been added. Running the command for
imports that have not yet been added to the handler Go code succeeded without
errors and updated the `Gopkg.toml` and `Gopkg.lock` files as expected.

### ARM64 Image Builds

This function is going to be deployed onto a Raspberry Pi using ARM64 so the
build and deploy process is slightly different than a basic `faas-cli up`
command. The below command will create a new directory containing the
`Dockerfile` and artifacts that will be used to build the container image.

```sh
faas-cli build --shrinkwrap -f slack.yml
```

The below commands should only need to be run once but will create a new Docker
build context for using with [Docker Buildx] to create images for multiple
platforms.

```sh
export DOCKER_CLI_EXPERIMENTAL=enabled
docker buildx create --use --name=multiarch
docker buildx inspect --bootstrap
```

Run the below command to use Buildx to create an image that supports both amd64
and arm64 architectures, and push it to the registry.

```sh
$ docker buildx build \
 --push \
 --tag registry.mydomain.io/openfaas/slack:latest \
 --platform=linux/amd64,linux/arm64 \
 build/slack/
```

## Deploying the Function

Run the below commands to point to the OpenFaaS gateway and authenticate.

```sh
$ export OPENFAAS_URL=https://gateway.mydomain.io
$ export PASSWORD=$(kubectl get secret -n openfaas basic-auth -o jsonpath="{.data.basic-auth-password}" | base64 --decode; echo)

$ echo -n $PASSWORD | faas-cli login --username admin --password-stdin
Calling the OpenFaaS server to validate the credentials...
credentials saved for admin https://gateway.mydomain.io
```

Run the below command to deploy the function. The provides the three environment
variables used by this function, and also specifies the `slack-api-token` secret
so that the function has access to this. Because access is granted to this
secret it will be mounted as a file to `/var/openfaas/secrets/slack-api-token`.

```sh
$ faas-cli deploy \
  --image registry.mydomain.io/openfaas/slack:latest \
  --name slack \
  --env SLACK_CHANNEL=general \
  --env SLACK_DEBUG=false \
  --env SLACK_LOGLEVEL=info \
  --secret slack-api-token

Deployed. 202 Accepted.
URL: https://gateway.mydomain.io/function/slack
```

Run the below command, using the awesome [HTTPie] command-line utility as a
replacement for cURL, to send a message to the Slack channel.

```sh
echo 'HTTPie is Awesome! :heart:' | http POST https://gateway.mydomain.io/function/slack
```

Run the below command to remove the function.

```sh
faas-cli remove slack
```

[arkade]: https://github.com/alexellis/arkade
[chat:write]: https://api.slack.com/scopes/chat:write
[dep]: https://github.com/golang/dep
[docker buildx]:
  https://docs.docker.com/engine/reference/commandline/buildx_build/
[go - dependencies]: https://docs.openfaas.com/cli/templates/#go-go-dependencies
[httpie]: https://httpie.io/
[openfaas]: https://www.openfaas.com/
[openfaas deployment]: https://docs.openfaas.com/deployment/
[openfaas using secrets]: https://docs.openfaas.com/reference/secrets/
[simple serverless with golang functions and microservices]:
  https://www.openfaas.com/blog/golang-serverless/
[slack apps]: https://api.slack.com/apps
[use a private registry with kubernetes]:
  https://docs.openfaas.com/deployment/kubernetes/#use-a-private-registry-with-kubernetes
