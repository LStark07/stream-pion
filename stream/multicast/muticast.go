package multicast

import (
	"bufio"
	"io"
	"log"
	"os/exec"
)

type Options struct {
	Outputs map[string][]string
}

var (
	options            Options
	ffmpegInstances    = make(map[string]*exec.Cmd)
	ffmpegInputStreams = make(map[string]*io.WriteCloser)
)

// New Load configuration
func New(cfg *Options) error {
	options = *cfg
	return nil
}

// RegisterStream Declare a new open stream and create ffmpeg instances
func RegisterStream(streamKey string) {
	if len(options.Outputs[streamKey]) == 0 {
		return
	}

	params := []string{"-re", "-i", "pipe:0"}
	for _, stream := range options.Outputs[streamKey] {
		// TODO Set optimal parameters
		params = append(params, "-f", "flv", "-c:v", "libx264", "-preset",
			"veryfast", "-maxrate", "3000k", "-bufsize", "6000k", "-pix_fmt", "yuv420p", "-g", "50", "-c:a", "aac",
			"-b:a", "160k", "-ac", "2", "-ar", "44100", stream)
	}
	// Launch FFMPEG instance
	ffmpeg := exec.Command("ffmpeg", params...)

	// Open pipes
	input, err := ffmpeg.StdinPipe()
	if err != nil {
		panic(err)
	}
	output, err := ffmpeg.StdoutPipe()
	if err != nil {
		panic(err)
	}
	errOutput, err := ffmpeg.StderrPipe()
	if err != nil {
		panic(err)
	}
	ffmpegInstances[streamKey] = ffmpeg
	ffmpegInputStreams[streamKey] = &input

	if err := ffmpeg.Start(); err != nil {
		panic(err)
	}

	// Log ffmpeg output
	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			log.Println("[FFMPEG " + streamKey + "] " + scanner.Text())
		}
	}()
	// Log also error output
	go func() {
		scanner := bufio.NewScanner(errOutput)
		for scanner.Scan() {
			log.Println("[FFMPEG ERROR " + streamKey + "] " + scanner.Text())
		}
	}()
}

// SendPacket When a SRT packet is received, transmit it to all FFMPEG instances related to the stream key
func SendPacket(streamKey string, data []byte) {
	stdin := ffmpegInputStreams[streamKey]
	_, err := (*stdin).Write(data)
	if err != nil {
		log.Println("Error while sending a packet to external streaming server for key "+streamKey, err)
	}
}

// CloseConnection When the stream is ended, close FFMPEG instances
func CloseConnection(streamKey string) {
	ffmpeg := ffmpegInstances[streamKey]
	if err := ffmpeg.Process.Kill(); err != nil {
		panic(err)
	}
	delete(ffmpegInstances, streamKey)
	delete(ffmpegInputStreams, streamKey)
}
