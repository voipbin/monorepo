package transcripthandler

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	smfile "monorepo/bin-storage-manager/models/file"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
)

// fakeSpeechServer is an in-memory speech service. It records every LongRunningRecognize request and
// answers with whatever the responder returns, so the handler can be driven without any network.
type fakeSpeechServer struct {
	speechpb.UnimplementedSpeechServer

	responder func(req *speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error)

	mu       sync.Mutex
	requests []*speechpb.LongRunningRecognizeRequest
}

func (f *fakeSpeechServer) LongRunningRecognize(_ context.Context, req *speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()

	return f.responder(req)
}

func (f *fakeSpeechServer) recorded() []*speechpb.LongRunningRecognizeRequest {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*speechpb.LongRunningRecognizeRequest{}, f.requests...)
}

// newFakeSpeech starts the fake on a bufconn listener and returns a speech client connected to it.
func newFakeSpeech(t *testing.T, responder func(req *speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error)) (*speech.Client, *fakeSpeechServer) {
	t.Helper()

	fake := &fakeSpeechServer{responder: responder}
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	speechpb.RegisterSpeechServer(srv, fake)
	go func() { _ = srv.Serve(lis) }()

	client, err := speech.NewClient(
		context.Background(),
		option.WithGRPCDialOption(grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() })),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithEndpoint("passthrough:///bufnet"),
		option.WithoutAuthentication(),
	)
	if err != nil {
		srv.Stop()
		t.Fatalf("could not create the speech client. err: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
		srv.Stop()
	})

	return client, fake
}

func fakeWord(word string, startMs int64) *speechpb.WordInfo {
	return &speechpb.WordInfo{
		Word:      word,
		StartTime: durationpb.New(time.Duration(startMs) * time.Millisecond),
	}
}

// fakeDoneOperation wraps results into an already finished long running operation.
func fakeDoneOperation(t *testing.T, results ...*speechpb.SpeechRecognitionResult) *longrunningpb.Operation {
	t.Helper()

	res, err := anypb.New(&speechpb.LongRunningRecognizeResponse{Results: results})
	if err != nil {
		t.Fatalf("could not pack the response. err: %v", err)
	}

	return &longrunningpb.Operation{
		Name:   "operations/test",
		Done:   true,
		Result: &longrunningpb.Operation_Response{Response: res},
	}
}

func fakeResult(words ...*speechpb.WordInfo) *speechpb.SpeechRecognitionResult {
	return &speechpb.SpeechRecognitionResult{
		Alternatives: []*speechpb.SpeechRecognitionAlternative{{Words: words}},
	}
}

func Test_processFromRecording(t *testing.T) {
	tests := []struct {
		name string

		results   []*speechpb.SpeechRecognitionResult
		direction transcript.Direction

		expectRes []*transcript.Transcript
	}{
		{
			// each terminator is followed by another word so that the sentence split cannot be
			// mistaken for the flush of the trailing sentence.
			name: "splits on the three terminators, joins words across results and flushes the tail",

			results: []*speechpb.SpeechRecognitionResult{
				fakeResult(fakeWord("Hello", 500), fakeWord("there.", 900), fakeWord("How", 1500)),
				fakeResult(fakeWord("are", 1800), fakeWord("you?", 2100), fakeWord("Great!", 3000), fakeWord("bye", 3500)),
			},
			direction: transcript.DirectionIn,

			expectRes: []*transcript.Transcript{
				{Direction: transcript.DirectionIn, Message: "Hello there.", OffsetMs: 500},
				{Direction: transcript.DirectionIn, Message: "How are you?", OffsetMs: 1500},
				{Direction: transcript.DirectionIn, Message: "Great!", OffsetMs: 3000},
				{Direction: transcript.DirectionIn, Message: "bye", OffsetMs: 3500},
			},
		},
		{
			name: "a sentence that ends exactly at the last word is not duplicated",

			results: []*speechpb.SpeechRecognitionResult{
				fakeResult(fakeWord("Hi", 100), fakeWord("there.", 400)),
			},
			direction: transcript.DirectionOut,

			expectRes: []*transcript.Transcript{
				{Direction: transcript.DirectionOut, Message: "Hi there.", OffsetMs: 100},
			},
		},
		{
			name: "a sentence offset is the start of its first word in whole milliseconds",

			results: []*speechpb.SpeechRecognitionResult{
				fakeResult(fakeWord("One.", 1234), fakeWord("Two.", 2999)),
			},
			direction: transcript.DirectionBoth,

			expectRes: []*transcript.Transcript{
				{Direction: transcript.DirectionBoth, Message: "One.", OffsetMs: 1234},
				{Direction: transcript.DirectionBoth, Message: "Two.", OffsetMs: 2999},
			},
		},
		{
			name: "a response without results yields an empty list",

			results:   nil,
			direction: transcript.DirectionBoth,

			expectRes: []*transcript.Transcript{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, fake := newFakeSpeech(t, func(*speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error) {
				return fakeDoneOperation(t, tt.results...), nil
			})
			h := &transcriptHandler{clientSpeech: client}

			res, err := h.processFromRecording(context.Background(), "gs://bucket-a/path/rec_in.wav", "ko-KR", tt.direction)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != len(tt.expectRes) {
				t.Fatalf("Wrong match. expect: %d transcripts, got: %d (%v)", len(tt.expectRes), len(res), res)
			}
			for i, expect := range tt.expectRes {
				if res[i].Direction != expect.Direction || res[i].Message != expect.Message || res[i].OffsetMs != expect.OffsetMs {
					t.Errorf("Wrong match. index: %d, expect: %v/%q/%d, got: %v/%q/%d", i,
						expect.Direction, expect.Message, expect.OffsetMs, res[i].Direction, res[i].Message, res[i].OffsetMs)
				}
			}

			reqs := fake.recorded()
			if len(reqs) != 1 {
				t.Fatalf("Wrong match. expect: 1 request, got: %d", len(reqs))
			}
			cfg := reqs[0].GetConfig()
			if reqs[0].GetAudio().GetUri() != "gs://bucket-a/path/rec_in.wav" ||
				cfg.GetLanguageCode() != "ko-KR" ||
				cfg.GetEncoding() != speechpb.RecognitionConfig_LINEAR16 ||
				cfg.GetSampleRateHertz() != 8000 ||
				!cfg.GetEnableWordTimeOffsets() ||
				!cfg.GetEnableAutomaticPunctuation() {
				t.Errorf("Wrong match. unexpected recognize request: %v", reqs[0])
			}
		})
	}
}

func Test_processFromRecording_errors(t *testing.T) {
	tests := []struct {
		name string

		responder func(req *speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error)
	}{
		{
			name: "the recognize call is rejected",

			responder: func(*speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error) {
				return nil, grpcstatus.Error(codes.PermissionDenied, "rejected")
			},
		},
		{
			name: "the finished operation carries an error",

			responder: func(*speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error) {
				return &longrunningpb.Operation{
					Name:   "operations/test",
					Done:   true,
					Result: &longrunningpb.Operation_Error{Error: &status.Status{Code: int32(codes.InvalidArgument), Message: "bad audio"}},
				}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newFakeSpeech(t, tt.responder)
			h := &transcriptHandler{clientSpeech: client}

			res, err := h.processFromRecording(context.Background(), "gs://bucket-a/path/rec.wav", "en-US", transcript.DirectionBoth)
			if err == nil {
				t.Fatalf("Wrong match. expect: err, got: ok (%v)", res)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_Recording_withFakeSTT(t *testing.T) {
	customerID := uuid.FromStringOrNil("6c1e2b10-0000-11ee-9d23-000000000001")
	transcribeID := uuid.FromStringOrNil("6c1e2b10-0000-11ee-9d23-000000000002")
	recordingID := uuid.FromStringOrNil("6c1e2b10-0000-11ee-9d23-000000000003")

	files := []smfile.File{
		{BucketName: "bucket-a", Filepath: "rec/call_in.wav", Filename: "call_in.wav"},
		{BucketName: "bucket-a", Filepath: "rec/call_out.wav", Filename: "call_out.wav"},
		{BucketName: "bucket-a", Filepath: "rec/call.wav", Filename: "call.wav"},
	}
	expectFilters := map[smfile.Field]any{
		smfile.FieldDeleted:     false,
		smfile.FieldReferenceID: recordingID.String(),
	}

	type created struct {
		direction transcript.Direction
		message   string
		offsetMs  int64
	}

	t.Run("transcripts of every file are created in offset order with the direction of their file", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		client, fake := newFakeSpeech(t, func(req *speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error) {
			switch req.GetAudio().GetUri() {
			case "gs://bucket-a/rec/call_in.wav":
				return fakeDoneOperation(t, fakeResult(fakeWord("In", 2000), fakeWord("one.", 2100))), nil
			case "gs://bucket-a/rec/call_out.wav":
				return fakeDoneOperation(t, fakeResult(fakeWord("Out", 1000), fakeWord("one.", 1100), fakeWord("Out", 3000), fakeWord("two.", 3100))), nil
			default:
				return fakeDoneOperation(t, fakeResult(fakeWord("Both", 1000), fakeWord("one.", 1100))), nil
			}
		})
		h := &transcriptHandler{
			utilHandler:   mockUtil,
			reqHandler:    mockReq,
			db:            mockDB,
			notifyHandler: mockNotify,
			clientSpeech:  client,
		}
		ctx := context.Background()

		mockReq.EXPECT().StorageV1FileList(ctx, "", uint64(100), expectFilters).Return(files, nil)

		var got []created
		mockUtil.EXPECT().UUIDCreate().DoAndReturn(func() uuid.UUID { return uuid.Must(uuid.NewV4()) }).Times(4)
		mockDB.EXPECT().TranscriptCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, tr *transcript.Transcript) error {
			got = append(got, created{direction: tr.Direction, message: tr.Message, offsetMs: tr.OffsetMs})
			return nil
		}).Times(4)
		mockDB.EXPECT().TranscriptGet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*transcript.Transcript, error) {
			return &transcript.Transcript{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, TranscribeID: transcribeID}, nil
		}).Times(4)
		mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, transcript.EventTypeTranscriptCreated, gomock.Any()).Times(4)

		res, err := h.Recording(ctx, customerID, transcribeID, recordingID, "en-US")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(res) != 4 {
			t.Fatalf("Wrong match. expect: 4 transcripts, got: %d", len(res))
		}

		// same offset keeps the file order: the out file comes before the unsuffixed file
		expect := []created{
			{transcript.DirectionOut, "Out one.", 1000},
			{transcript.DirectionBoth, "Both one.", 1000},
			{transcript.DirectionIn, "In one.", 2000},
			{transcript.DirectionOut, "Out two.", 3000},
		}
		if len(got) != len(expect) {
			t.Fatalf("Wrong match. expect: %v, got: %v", expect, got)
		}
		for i := range expect {
			if got[i] != expect[i] {
				t.Errorf("Wrong match. index: %d, expect: %v, got: %v", i, expect[i], got[i])
			}
		}

		uris := []string{}
		for _, r := range fake.recorded() {
			uris = append(uris, r.GetAudio().GetUri())
		}
		if strings.Join(uris, ",") != "gs://bucket-a/rec/call_in.wav,gs://bucket-a/rec/call_out.wav,gs://bucket-a/rec/call.wav" {
			t.Errorf("Wrong match. files must be transcribed in list order, got: %v", uris)
		}
	})

	t.Run("a file that cannot be transcribed fails the whole call and creates nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)

		client, _ := newFakeSpeech(t, func(req *speechpb.LongRunningRecognizeRequest) (*longrunningpb.Operation, error) {
			if strings.HasSuffix(req.GetAudio().GetUri(), "call_out.wav") {
				return nil, grpcstatus.Error(codes.PermissionDenied, "rejected")
			}
			return fakeDoneOperation(t, fakeResult(fakeWord("Fine.", 100))), nil
		})
		// db, util and notify are nil: any attempt to create a transcript would panic
		h := &transcriptHandler{reqHandler: mockReq, clientSpeech: client}
		ctx := context.Background()

		mockReq.EXPECT().StorageV1FileList(ctx, "", uint64(100), expectFilters).Return(files, nil)

		res, err := h.Recording(ctx, customerID, transcribeID, recordingID, "en-US")
		if err == nil {
			t.Fatalf("Wrong match. expect: err, got: ok (%v)", res)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})
}
