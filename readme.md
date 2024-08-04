# Recorder System using Xvfb, ffmpeg, and PulseAudio.

System works very simply, it uses Xvfb to create a virtual display, PulseAudio to create a virtual audio device, and ffmpeg to record the screen and audio. Setups chromedp for the browser automation to record any browser url.

## Requirements

- docker.

## Usage

```base
docker build -t recorder-dev .
```

Preffred to use the docker-compose to run the container.

```base
docker-compose up
```

## TODO

- [ ] Add API server Capabilties to make custom recording calls.
- [ ] Do proper file system implementation to handle different recordings.
- [ ] Add Capabilties to do livestreaming onto multiple rtmp, rtmps endpoints ( Hopefully ffmpeg will make it easy )
- [ ] Maybe do the kubernetes thing for auto scaling.
- [ ] Benchmarking, because I know sometimes recordings will fail to record propely, because its life.
- [ ] Add more documentation.
- [ ] Add more tests
