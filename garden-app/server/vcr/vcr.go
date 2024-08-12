package vcr

import (
	"github.com/calvinmclean/babyapi"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

var rec *recorder.Recorder

// StopRecorder stops the Recorder to write results to file
func StopRecorder() {
	if rec == nil {
		return
	}

	err := rec.Stop()
	if err != nil {
		panic(err)
	}
}

// MustSetupVCR will create a new Recorder and add it to babyapi.DefaultMiddleware.
// Panics if there is an error
func MustSetupVCR(cassetteName string) {
	var err error
	rec, err = recorder.New(
		recorder.WithCassette(cassetteName),
		recorder.WithMode(recorder.ModeRecordOnly),
	)
	if err != nil {
		panic(err)
	}

	babyapi.DefaultMiddleware = append(babyapi.DefaultMiddleware, rec.HTTPMiddleware)
}
