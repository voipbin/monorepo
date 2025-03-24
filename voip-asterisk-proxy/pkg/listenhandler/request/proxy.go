package request

// V1DataCallsPost is
// v1 data type request struct for
// /proxy/recording_file_move POST
type ProxyDataRecordingFileMovePost struct {
	Filenames []string `json:"filenames,omitempty"`
}
