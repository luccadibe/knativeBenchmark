initial readme



Knative requirements:
For production purposes, it is recommended that:

If you have only one node in your cluster, you need at least 6 CPUs, 6 GB of memory, and 30 GB of disk storage.
If you have multiple nodes in your cluster, for each node you need at least 2 CPUs, 4 GB of memory, and 20 GB of disk storage.
You have a cluster that uses Kubernetes v1.28 or newer.
You have installed the kubectl CLI.
Your Kubernetes cluster must have access to the internet, because Kubernetes needs to be able to fetch images. To pull from a private registry, see Deploying images from a private container registry.

The system requirements provided are recommendations only. The requirements for your installation might vary, depending on whether you use optional components, such as a networking layer.

Knative supports encryption features through cert-manager. Follow the documentation in Serving encryption for more information.
maybe benchmark with encryption vs no encryption

func languages
go
node
python
quarkus
rust
springboot
typescript

func templates
LANGUAGE     TEMPLATE
go           cloudevents
go           http
node         cloudevents
node         http
python       cloudevents
python       flask
python       http
python       wsgi
quarkus      cloudevents
quarkus      http
rust         cloudevents
rust         http
springboot   cloudevents
springboot   http
typescript   cloudevents
typescript   http


func build --help

NAME
        func build - Build a function container locally without deploying

SYNOPSIS
        func build [-r|--registry] [--builder] [--builder-image]
                         [--push] [--username] [--password] [--token]
                     [--platform] [-p|--path] [-c|--confirm] [-v|--verbose]
                         [--build-timestamp] [--registry-insecure]

DESCRIPTION

        Builds a function's container image and optionally pushes it to the
        configured container registry.

        By default building is handled automatically when deploying (see the deploy
        subcommand). However, sometimes it is useful to build a function container
        outside of this normal deployment process, for example for testing or during
        composition when integrating with other systems. Additionally, the container
        can be pushed to the configured registry using the --push option.

        When building a function for the first time, either a registry or explicit
        image name is required.  Subsequent builds will reuse these option values.

EXAMPLES

        o Build a function container using the given registry.
          The full image name will be calculated using the registry and function name.
          $ func build --registry registry.example.com/alice

        o Build a function container using an explicit image name, ignoring registry
          and function name.
          $ func build --image registry.example.com/alice/f:latest

        o Rebuild a function using prior values to determine container name.
          $ func build

        o Build a function specifying the Source-to-Image (S2I) builder
          $ func build --builder=s2i

        o Build a function specifying the Pack builder with a custom Buildpack
          builder image.
          $ func build --builder=pack --builder-image=cnbs/sample-builder:bionic



Usage:
  func build [flags]

Flags:
      --build-timestamp        Use the actual time as the created time for the docker image. This is only useful
                               for buildpacks builder.
  -b, --builder string         Builder to use when creating the function's container. Currently supported builders
                               are "pack" and "s2i". ($FUNC_BUILDER) (default "pack")
      --builder-image string   Specify a custom builder image for use by the builder other than its default.
                               ($FUNC_BUILDER_IMAGE)
  -c, --confirm                Prompt to confirm options interactively ($FUNC_CONFIRM)
  -i, --image string           Full image name in the form [registry]/[namespace]/[name]:[tag] (optional). This
                               option takes precedence over --registry ($FUNC_IMAGE)
  -p, --path string            Path to the function.  Default is current directory ($FUNC_PATH)
      --platform string        Optionally specify a target platform, for example "linux/amd64" when using the s2i
                               build strategy
  -u, --push                   Attempt to push the function image to the configured registry after being
                               successfully built
  -r, --registry string        Container registry + registry namespace. (ex 'ghcr.io/myuser').  The full image
                               name is automatically determined using this along with function name. ($FUNC_REGISTRY)
      --registry-insecure      Skip TLS certificate verification when communicating in HTTPS with the registry
                               ($FUNC_REGISTRY_INSECURE)
  -v, --verbose                Print verbose logs ($FUNC_VERBOSE)