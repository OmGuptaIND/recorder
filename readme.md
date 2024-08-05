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

SSH into the container to run the server.

```base
make run
```

`NOTE: Check you are inside the app working directory.`

Check the server is running by visiting the `http://localhost:3000/`.

```
Sometimes Pulse Audio just doesn't want to start, so to give it a little push, 
After you SSH inside the container, run the following command.
`./pulseaudio.sh`
```

### API ENDPOINTS

- `/ping` - To check the server is running.

```curl
curl --location 'http://localhost:3000/ping'
```

- `/start-recording` - To start the recording.

```curl
curl --location 'http://localhost:3000/start-recording' \
--header 'Content-Type: application/json' \
--data '{
    "url": "https://www.youtube.com/watch?v=cii6ruuycQA&ab_channel=OliviaRodrigoVEVO"
}'
```

- `/stop-recording` - To stop the recording.
  Use the id from the start-recording response.

```curl
curl --location --request PATCH 'http://localhost:3000/stop-recording' \
--header 'Content-Type: application/json' \
--data '{
    "id": "3f20a10a-2768-4080-ab5f-341cdc489eb2"
}'
```

## TODO

- [x] Add API server Capabilties to make custom recording calls.
- [x] Ability to record multiple recordings at the same time. ( Took some time to figure out the pulse audio thing )
- [x] Do proper file system implementation to handle different recordings.
- [ ] Add Capabilties to do livestreaming onto multiple rtmp, rtmps endpoints ( Hopefully ffmpeg will make it easy )
- [ ] Maybe do the kubernetes thing for auto scaling.
- [ ] Benchmarking, because I know sometimes recordings will fail to record propely, because its life.
- [ ] Add more documentation.
- [ ] Add more tests


## Scripts
- To get all the running pulse audio
```bash
pactl list short sources
```

- To Close the pulse audio
```bash
pulseaudio -k
```

- To Close all the Pulse Sink
```bash
pactl unload-module module-null-sink
```