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
