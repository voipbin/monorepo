package audiohandler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/polly"
	"github.com/gofrs/uuid"
)

// failingReader returns the given bytes once and then an error.
type failingReader struct {
	data []byte
	read bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, errors.New("synthetic read failure")
	}
	r.read = true
	return copy(p, r.data), nil
}

func Test_savePCMAsWAV(t *testing.T) {
	tests := []struct {
		name        string
		pcm         []byte
		sampleRate  int
		numChannels int

		// expectHeader is the full 44 byte canonical WAV header, written out by hand.
		expectHeader []byte
	}{
		{
			name:        "8kHz mono, 6 bytes of audio",
			pcm:         []byte{1, 2, 3, 4, 5, 6},
			sampleRate:  8000,
			numChannels: 1,
			expectHeader: []byte{
				'R', 'I', 'F', 'F',
				42, 0, 0, 0, // ChunkSize = 36 + 6
				'W', 'A', 'V', 'E',
				'f', 'm', 't', ' ',
				16, 0, 0, 0, // Subchunk1Size
				1, 0, // PCM
				1, 0, // channels
				0x40, 0x1f, 0, 0, // 8000 Hz
				0x80, 0x3e, 0, 0, // byte rate 16000
				2, 0, // block align
				16, 0, // bits per sample
				'd', 'a', 't', 'a',
				6, 0, 0, 0, // Subchunk2Size
			},
		},
		{
			name:        "16kHz stereo, 300 bytes of audio",
			pcm:         make([]byte, 300),
			sampleRate:  16000,
			numChannels: 2,
			expectHeader: []byte{
				'R', 'I', 'F', 'F',
				0x50, 0x01, 0, 0, // ChunkSize = 36 + 300 = 336
				'W', 'A', 'V', 'E',
				'f', 'm', 't', ' ',
				16, 0, 0, 0,
				1, 0,
				2, 0, // channels
				0x80, 0x3e, 0, 0, // 16000 Hz
				0x00, 0xfa, 0, 0, // byte rate 64000
				4, 0, // block align
				16, 0,
				'd', 'a', 't', 'a',
				0x2c, 0x01, 0, 0, // Subchunk2Size = 300
			},
		},
		{
			name:        "no audio",
			pcm:         nil,
			sampleRate:  8000,
			numChannels: 1,
			expectHeader: []byte{
				'R', 'I', 'F', 'F',
				36, 0, 0, 0,
				'W', 'A', 'V', 'E',
				'f', 'm', 't', ' ',
				16, 0, 0, 0,
				1, 0,
				1, 0,
				0x40, 0x1f, 0, 0,
				0x80, 0x3e, 0, 0,
				2, 0,
				16, 0,
				'd', 'a', 't', 'a',
				0, 0, 0, 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "out.wav")
			h := &audioHandler{}

			if err := h.savePCMAsWAV(strings.NewReader(string(tt.pcm)), path, tt.sampleRate, tt.numChannels); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("could not read the output: %v", err)
			}
			if len(got) != len(tt.expectHeader)+len(tt.pcm) {
				t.Fatalf("Wrong file size. expected: %d, got: %d", len(tt.expectHeader)+len(tt.pcm), len(got))
			}
			if string(got[:len(tt.expectHeader)]) != string(tt.expectHeader) {
				t.Errorf("Wrong WAV header.\nexpected: %v\ngot:      %v", tt.expectHeader, got[:len(tt.expectHeader)])
			}
			if string(got[len(tt.expectHeader):]) != string(tt.pcm) {
				t.Errorf("the PCM payload must follow the header unchanged")
			}
		})
	}
}

func Test_savePCMAsWAV_errors(t *testing.T) {
	h := &audioHandler{}

	t.Run("target directory does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing-dir", "out.wav")
		err := h.savePCMAsWAV(strings.NewReader("pcm"), path, 8000, 1)
		if err == nil || !strings.Contains(err.Error(), "failed to create file") {
			t.Fatalf("expected a file creation error, got: %v", err)
		}
	})

	t.Run("reading the audio fails", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "out.wav")
		err := h.savePCMAsWAV(&failingReader{data: []byte{1, 2}}, path, 8000, 1)
		if err == nil || !strings.Contains(err.Error(), "failed to write PCM data") {
			t.Fatalf("expected a PCM write error, got: %v", err)
		}
	})
}

func newLocalPollyClient(endpoint string) *polly.Client {
	return polly.New(polly.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       "test-region-1",
		Credentials:  credentials.NewStaticCredentialsProvider("test-access-key", "test-secret-key", ""),
		Retryer:      aws.NopRetryer{},
	})
}

func Test_awsAudioCreate(t *testing.T) {
	callID := uuid.FromStringOrNil("9e4f3000-0000-4000-8000-000000000001")

	t.Run("synthesized audio is saved as a WAV file using the requested voice", func(t *testing.T) {
		pcm := []byte{10, 20, 30, 40}
		var gotBody string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			gotBody = string(body)
			w.Header().Set("Content-Type", "audio/pcm")
			_, _ = w.Write(pcm)
		}))
		defer srv.Close()

		h := &audioHandler{awsClient: newLocalPollyClient(srv.URL)}
		path := filepath.Join(t.TempDir(), "speech.wav")

		if err := h.awsAudioCreate(context.Background(), callID, "<speak>hi</speak>", "ko-KR", "Kendra", path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, want := range []string{`"VoiceId":"Kendra"`, `"TextType":"ssml"`, `"SampleRate":"8000"`, `"OutputFormat":"pcm"`} {
			if !strings.Contains(gotBody, want) {
				t.Errorf("the request must contain %s. got: %s", want, gotBody)
			}
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("could not read the output: %v", err)
		}
		if len(got) != 44+len(pcm) || string(got[44:]) != string(pcm) || string(got[:4]) != "RIFF" {
			t.Errorf("unexpected WAV output of %d bytes", len(got))
		}
	})

	t.Run("the default voice for the language is used when no voice is given", func(t *testing.T) {
		var gotBody string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			gotBody = string(body)
			w.Header().Set("Content-Type", "audio/pcm")
			_, _ = w.Write([]byte{1, 1})
		}))
		defer srv.Close()

		h := &audioHandler{awsClient: newLocalPollyClient(srv.URL)}
		path := filepath.Join(t.TempDir(), "speech.wav")

		if err := h.awsAudioCreate(context.Background(), callID, "hi", "ko-KR", "", path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(gotBody, `"VoiceId":"Seoyeon"`) {
			t.Errorf("expected the Korean default voice. got: %s", gotBody)
		}
	})

	t.Run("service failure is reported and no file is written", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		h := &audioHandler{awsClient: newLocalPollyClient(srv.URL)}
		path := filepath.Join(t.TempDir(), "speech.wav")

		err := h.awsAudioCreate(context.Background(), callID, "hi", "en-US", "", path)
		if err == nil || !strings.Contains(err.Error(), "failed to synthesize speech") {
			t.Fatalf("expected a synthesis error, got: %v", err)
		}
		if _, errStat := os.Stat(path); !os.IsNotExist(errStat) {
			t.Errorf("no file may be created on a synthesis failure. stat error: %v", errStat)
		}
	})

	t.Run("an unwritable target is reported", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/pcm")
			_, _ = w.Write([]byte{1})
		}))
		defer srv.Close()

		h := &audioHandler{awsClient: newLocalPollyClient(srv.URL)}
		path := filepath.Join(t.TempDir(), "missing-dir", "speech.wav")

		err := h.awsAudioCreate(context.Background(), callID, "hi", "en-US", "", path)
		if err == nil || !strings.Contains(err.Error(), "failed to save PCM as WAV") {
			t.Fatalf("expected a save error, got: %v", err)
		}
	})
}

func Test_AudioCreate_dispatchesToAWS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write([]byte{7, 7})
	}))
	defer srv.Close()

	h := &audioHandler{awsClient: newLocalPollyClient(srv.URL)}
	path := filepath.Join(t.TempDir(), "speech.wav")

	err := h.AudioCreate(context.Background(), uuid.Nil, "hi", "en-US", "aws", "", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, errStat := os.Stat(path); errStat != nil {
		t.Errorf("the AWS path must produce the file: %v", errStat)
	}
}

func Test_awsGetClient(t *testing.T) {
	// keep the shared config lookup away from the host's real AWS files
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials"))
	t.Setenv("AWS_PROFILE", "")

	client, err := awsGetClient(context.Background(), "test-access-key", "test-secret-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected a client")
	}
	if got := client.Options().Region; got != defaultAWSRegion {
		t.Errorf("Wrong region. expected: %s, got: %s", defaultAWSRegion, got)
	}
}
