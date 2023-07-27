<!-- markdownlint-disable first-line-h1 no-inline-html -->
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/estesp/onimage/main/docs/logo/on-image-logo-dark-mode-medium.png">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/estesp/onimage/main/docs/logo/on-image-logo-medium.png">
  <img alt="Finch logo" width=40% height=auto src="https://raw.githubusercontent.com/estesp/onimage/main/docs/logo/on-image-logo-medium.png">
</picture>

## An image processing pipeline service for weathercams

OnImage() is a configurable image processing daemon used to handle specific image processing
steps to a flow of captured images being generated from a camera. It's specific use is to
handle the regular capture of daytime photos from a Raspberry Pi-based weathercam service that
publishes to an Amazon S3-backed single page website. See [kwcam.live](https://kwcam.live) for
an operating example of this project.

This daemon is written in Go and is configurable via a simple TOML configuration file. While
the image processing steps are specifically written for the current use case of a webcam
taking weather photos at a high frequency (every few minutes), the configuration file abstracts
all the specifics of AWS accounts/S3 buckets, monitor service (cronitor.io), Weather API service,
and specific local paths/configuration. It is capable to be operated on a substantial Raspberry
Pi (rpi4 w/8GB RAM is being used today with this code), or it can be run on a separate system
from the IoT or small(er) device taking the photos as long as the photo storage is accessible
over a filesystem. Prior versions of this software were used with an NFS mount to a much less
powerful Raspberry Pi taking the photos.

The core feature set includes:
 - Filesystem-watch based triggering of image processing pipeline
 - Heartbeat monitoring and error/incident reporting to cronitor.io (even with free tier)
 - Weather API w/configurable location and units to overlay images with current temperature
 - Runs `enfuse` against a multi-photo capture using the rPi HQ camera to implement "poor man's HDR"
 - Provides simple API endpoint for camera-capture script/device to know when to start/stop
   taking photos based on sunrise/sunset and, optionally, dark percent of captured photos
   (for after sunset), which uses the python-based OpenCV2 image model software

The example TOML configuration in the root of this repository is fully documented to provide
all the details you need to run OnImage() in your own environment.

### Building

This is a simple Go project that can be easily built with any recent Go version:

```shell
$ go build -o onimage .
```

This command will produce a binary for your host's OS and architecture. To cross-compile for a
unique target from your host, use the standard Go environment variable features. For example, to
build `onimage` for a Raspberry Pi you can use:

```shell
$ GOOS=linux GOARCH=arm64 go build -o onimage .
```

### Installing

The `onimage` binary can be installed in the `$PATH` and run as a standalone program. On start, the
program will look in the current directory and `/etc/onimage` for a file named `onimage.toml`. If
not found, the program will terminate as the configuration file and its settings are required for
operation.

A systemd unit would be a good contribution so that `onimage` can be run as a
service. A unit file does not exist at this time.

### What else is required to run a weather/webcam site?

It's probably clear that, by itself, this software daemon will not provide all the required
pieces to have a fully operating single-page website displaying captured photos. This program
is effectively the heart of a processing pipeline that requires additional pieces and
configuration at both ends of the pipeline.

On the input side, you need a camera and a capture script or program which writes images
from the camera to a directory on the local filesystem on some periodic interval. This
software program will expect a certain number of images placed in a specific directory
structure to work properly.

On the website side, you will need to connect the configured S3 bucket to a registered
domain which serves up the bucket as public content. There are many helpful guides for
using CloudFront and a publicly-readable S3 bucket to create a simple HTTPS website.

Looking for a detailed guide on how to set up these additional pieces? A HOWTO that
describes the exact setup used to operate [kwcam.live](https://kwcam.live)
is located in this repository at [docs/weathercam-howto.md](/docs/weathercam-howto.md)
