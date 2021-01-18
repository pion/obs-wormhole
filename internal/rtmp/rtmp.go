package rtmp

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	flvtag "github.com/yutopp/go-flv/tag"
	"github.com/yutopp/go-rtmp"
	rtmpmsg "github.com/yutopp/go-rtmp/message"
)

func StartServer(peerConnection *webrtc.PeerConnection, videoTrack, audioTrack *webrtc.TrackLocalStaticSample) *rtmp.Server {
	log.Println("Starting RTMP Server")

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":1935")
	if err != nil {
		log.Panicf("Failed: %+v", err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Panicf("Failed: %+v", err)
	}

	srv := rtmp.NewServer(&rtmp.ServerConfig{
		OnConnect: func(conn net.Conn) (io.ReadWriteCloser, *rtmp.ConnConfig) {
			return conn, &rtmp.ConnConfig{
				Handler: &Handler{
					peerConnection: peerConnection,
					videoTrack:     videoTrack,
					audioTrack:     audioTrack,
				},

				ControlState: rtmp.StreamControlStateConfig{
					DefaultBandwidthWindowSize: 6 * 1024 * 1024 / 8,
				},
				Logger: logrus.StandardLogger(),
			}
		},
	})

	go func() {
		err := srv.Serve(listener)
		if err != nil && err != rtmp.ErrClosed {
			log.Panicf("Failed: %+v", err)
		}
	}()

	return srv
}

type Handler struct {
	rtmp.DefaultHandler
	peerConnection         *webrtc.PeerConnection
	videoTrack, audioTrack *webrtc.TrackLocalStaticSample

	sps []byte
	pps []byte
}

func (h *Handler) OnServe(conn *rtmp.Conn) {
}

func (h *Handler) OnConnect(timestamp uint32, cmd *rtmpmsg.NetConnectionConnect) error {
	log.Printf("OnConnect: %#v", cmd)
	return nil
}

func (h *Handler) OnCreateStream(timestamp uint32, cmd *rtmpmsg.NetConnectionCreateStream) error {
	log.Printf("OnCreateStream: %#v", cmd)
	return nil
}

func (h *Handler) OnPublish(timestamp uint32, cmd *rtmpmsg.NetStreamPublish) error {
	log.Printf("OnPublish: %#v", cmd)

	if cmd.PublishingName == "" {
		return errors.New("PublishingName is empty")
	}
	return nil
}

func (h *Handler) OnAudio(timestamp uint32, payload io.Reader) error {
	var audio flvtag.AudioData
	if err := flvtag.DecodeAudioData(payload, &audio); err != nil {
		return err
	}

	data := new(bytes.Buffer)
	if _, err := io.Copy(data, audio.Data); err != nil {
		return err
	}

	return h.audioTrack.WriteSample(media.Sample{
		Data:     data.Bytes(),
		Duration: 20 * time.Millisecond,
	})
}

const (
	headerLengthField = 4
	spsId             = 0x67
	ppsId             = 0x68
)

func annexBPrefix() []byte {
	return []byte{0x00, 0x00, 0x00, 0x01}
}

func (h *Handler) OnVideo(timestamp uint32, payload io.Reader) error {
	var video flvtag.VideoData
	if err := flvtag.DecodeVideoData(payload, &video); err != nil {
		return err
	}

	data := new(bytes.Buffer)
	if _, err := io.Copy(data, video.Data); err != nil {
		return err
	}

	hasSpsPps := false
	outBuf := []byte{}
	videoBuffer := data.Bytes()
	if video.AVCPacketType == flvtag.AVCPacketTypeNALU {
		for offset := 0; offset < len(videoBuffer); {

			bufferLength := int(binary.BigEndian.Uint32(videoBuffer[offset : offset+headerLengthField]))
			if offset+bufferLength >= len(videoBuffer) {
				break
			}

			offset += headerLengthField

			if videoBuffer[offset] == spsId {
				hasSpsPps = true
				h.sps = append(annexBPrefix(), videoBuffer[offset:offset+bufferLength]...)
			} else if videoBuffer[offset] == ppsId {
				hasSpsPps = true
				h.pps = append(annexBPrefix(), videoBuffer[offset:offset+bufferLength]...)
			}

			outBuf = append(outBuf, annexBPrefix()...)
			outBuf = append(outBuf, videoBuffer[offset:offset+bufferLength]...)

			offset += int(bufferLength)

		}
	} else if video.AVCPacketType == flvtag.AVCPacketTypeSequenceHeader {
		const spsCountOffset = 5
		spsCount := videoBuffer[spsCountOffset] & 0x1F
		offset := 6
		h.sps = []byte{}
		for i := 0; i < int(spsCount); i++ {
			spsLen := binary.BigEndian.Uint16(videoBuffer[offset : offset+2])
			offset += 2
			if videoBuffer[offset] != spsId {
				panic("Failed to parse SPS")
			}
			h.sps = append(h.sps, annexBPrefix()...)
			h.sps = append(h.sps, videoBuffer[offset:offset+int(spsLen)]...)
			offset += int(spsLen)
		}
		ppsCount := videoBuffer[offset]
		offset++
		for i := 0; i < int(ppsCount); i++ {
			ppsLen := binary.BigEndian.Uint16(videoBuffer[offset : offset+2])
			offset += 2
			if videoBuffer[offset] != ppsId {
				panic("Failed to parse PPS")
			}
			h.sps = append(h.sps, annexBPrefix()...)
			h.sps = append(h.sps, videoBuffer[offset:offset+int(ppsLen)]...)
			offset += int(ppsLen)
		}
		return nil
	}

	// We have an unadorned keyframe, append SPS/PPS
	if video.FrameType == flvtag.FrameTypeKeyFrame && !hasSpsPps {
		outBuf = append(append(h.sps, h.pps...), outBuf...)
	}

	return h.videoTrack.WriteSample(media.Sample{
		Data:     outBuf,
		Duration: time.Second / 30,
	})
}

func (h *Handler) OnClose() {
	log.Printf("OnClose")
}
