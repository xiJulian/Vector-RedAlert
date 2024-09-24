package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	sdk_wrapper "github.com/fforchino/vector-go-sdk/pkg/sdk-wrapper"
	"github.com/fforchino/vector-go-sdk/pkg/vector"
	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
)

const (
	ip     = "..."
	serial = "..."
	guid   = "..."
)

func main() {

	v, err := vector.New(vector.WithTarget(ip), vector.WithSerialNo(serial), vector.WithToken(guid))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	r, _ := v.Conn.BehaviorControl(ctx)
	defer r.Send(&vectorpb.BehaviorControlRequest{RequestType: &vectorpb.BehaviorControlRequest_ControlRelease{ControlRelease: &vectorpb.ControlRelease{}}})

	var checkAlert func()
	checkAlert = func() {
		res, err := http.Get("https://www.oref.org.il/WarningMessages/Alert/alerts.json")
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		println(len(body))
		if len(body) <= 5 {
			time.Sleep(time.Millisecond * 2000)
			checkAlert()
			return
		}

		// testdata := []byte("{\"data\":[\"...\", \"test\"]}")
		data := map[string][]string{}
		json.Unmarshal(body, &data)

		for i := 0; i < len(data["data"]); i++ {
			if data["data"][i] == "..." {
				println("ALERT")
				r.Send(&vectorpb.BehaviorControlRequest{RequestType: &vectorpb.BehaviorControlRequest_ControlRequest{ControlRequest: &vectorpb.ControlRequest{Priority: vectorpb.ControlRequest_OVERRIDE_BEHAVIORS}}})
				defer r.Send(&vectorpb.BehaviorControlRequest{RequestType: &vectorpb.BehaviorControlRequest_ControlRelease{ControlRelease: &vectorpb.ControlRelease{}}})
				faceBytes := sdk_wrapper.DataOnImg("./resources/redalert.png")
				v.Conn.DisplayFaceImageRGB(
					ctx,
					&vectorpb.DisplayFaceImageRGBRequest{
						FaceData:         faceBytes,
						DurationMs:       uint32(1000),
						InterruptRunning: true,
					},
				)

				pcmFile, _ := os.ReadFile("resources/siren.pcm")
				var audioChunks [][]byte
				for len(pcmFile) >= 1024 {
					audioChunks = append(audioChunks, pcmFile[:1024])
					pcmFile = pcmFile[1024:]
				}
				var audioClient vectorpb.ExternalInterface_ExternalAudioStreamPlaybackClient
				audioClient, _ = v.Conn.ExternalAudioStreamPlayback(ctx)
				audioClient.SendMsg(&vectorpb.ExternalAudioStreamRequest{
					AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamPrepare{
						AudioStreamPrepare: &vectorpb.ExternalAudioStreamPrepare{
							AudioFrameRate: 16000,
							AudioVolume:    uint32(100),
						},
					},
				})
				for _, chunk := range audioChunks {
					audioClient.SendMsg(&vectorpb.ExternalAudioStreamRequest{
						AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamChunk{
							AudioStreamChunk: &vectorpb.ExternalAudioStreamChunk{
								AudioChunkSizeBytes: 1024,
								AudioChunkSamples:   chunk,
							},
						},
					})
					time.Sleep(time.Millisecond * 30)
				}
				audioClient.SendMsg(&vectorpb.ExternalAudioStreamRequest{
					AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamComplete{
						AudioStreamComplete: &vectorpb.ExternalAudioStreamComplete{},
					},
				})
			}
		}

		time.Sleep(time.Millisecond * 1000)
		checkAlert()
	}
	checkAlert()
}
